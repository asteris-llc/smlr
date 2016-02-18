package smlr

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"bytes"

	"github.com/Sirupsen/logrus"
	"github.com/jpillora/backoff"
	"golang.org/x/net/context"
)

// HTTPWaiter waits for an HTTP call to return
type HTTPWaiter struct {
	Method         string
	URL            string
	ExpectedStatus int
	Content        string
	EntireContent  bool
}

// Wait starts the waiting
func (h *HTTPWaiter) Wait(ctx context.Context, interval, timeout time.Duration) chan *Status {
	out := make(chan *Status, 1)

	go h.startWaiting(ctx, out, interval, timeout)

	return out
}

func (h *HTTPWaiter) startWaiting(ctx context.Context, out chan *Status, interval, timeout time.Duration) {
	defer close(out)

	boff := &backoff.Backoff{
		Min:    500 * time.Millisecond,
		Max:    3 * time.Second,
		Jitter: true,
	}

	timedOut := time.After(timeout)
	next := time.After(0)

	for {
		var status *Status
		select {
		case <-timedOut:
			status = &Status{Done: true, Error: errors.New("timed out")}
		case <-ctx.Done():
			status = &Status{Done: true, Error: errors.New("cancelled, ceasing wait")}
		case <-next:
			status = h.request(ctx)
		}

		out <- status

		if status.Done {
			break
		} else {
			d := boff.Duration()
			logrus.WithField("duration", d).Debug("backoff")
			next = time.After(d)
		}
	}
}

func (h *HTTPWaiter) request(ctx context.Context) *Status {
	req, err := http.NewRequest(h.Method, h.URL, nil)
	if err != nil {
		return &Status{Error: err}
	}

	return h.do(ctx, req, func(resp *http.Response, err error) *Status {
		// error matching
		if err != nil {
			if strings.Contains(err.Error(), "connection refused") {
				return &Status{Done: false, Message: "connection refused"}
			} else if strings.Contains(err.Error(), "no such host") {
				return &Status{Done: false, Message: "could not reach host"}
			}

			return &Status{Done: false, Message: err.Error()}
		}

		// status matching
		if resp.StatusCode != h.ExpectedStatus {
			return &Status{
				Done:    false,
				Message: fmt.Sprintf(`status "%s" does not match expected status (%d)`, resp.Status, h.ExpectedStatus),
			}
		}

		// content matching
		if h.Content != "" {
			content, err := ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()
			if err != nil {
				return &Status{
					Done:    false,
					Message: "could not read body",
				}
			}

			if h.EntireContent {
				if !bytes.Equal(content, []byte(h.Content)) {
					return &Status{
						Done:    false,
						Message: "response does not match content",
					}
				}
			} else {
				if !bytes.Contains(content, []byte(h.Content)) {
					return &Status{
						Done:    false,
						Message: "response does not contain content",
					}
				}
			}
		}

		return &Status{Done: true, Message: "service available"}
	})
}

func (h *HTTPWaiter) do(ctx context.Context, req *http.Request, cb func(*http.Response, error) *Status) *Status {
	trans := new(http.Transport)
	client := &http.Client{Transport: trans}

	errs := make(chan *Status, 1)

	go func() { errs <- cb(client.Do(req)) }()

	select {
	case <-ctx.Done():
		trans.CancelRequest(req)
		<-errs
		return &Status{Error: ctx.Err()}
	case err := <-errs:
		return err
	}
}
