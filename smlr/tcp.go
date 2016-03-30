package smlr

import (
	"bufio"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	"bytes"

	"github.com/Sirupsen/logrus"
	"github.com/jpillora/backoff"
	"golang.org/x/net/context"
)

var errNoMatch = errors.New("no content match")
var errNoMatchTimeout = errors.New("no content match within iotimeout")

// TCPWaiter waits for an TCP call to return with specified content
type TCPWaiter struct {
	URL           string
	Content       string
	Write         string        // write this to the connection before listening
	IOTimeout     time.Duration // how long to wait for r/w actions to complete
	EntireContent bool
}

// Wait starts the waiting (this is entirely copied from http)
func (t *TCPWaiter) Wait(ctx context.Context, interval, timeout time.Duration) chan *Status {
	out := make(chan *Status, 1)

	go t.startWaiting(ctx, out, interval, timeout)

	return out
}

// startWaiting loops until the timeout is reached or we're done waiting
// (this is entirely copied from http)
func (t *TCPWaiter) startWaiting(ctx context.Context, out chan *Status, interval, timeout time.Duration) {
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
			status = t.request(ctx)
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

// request actually sends the request over the network
// TODO: how to use context?
func (t *TCPWaiter) request(ctx context.Context) *Status {
	conn, err := net.Dial("tcp", t.URL)
	if conn != nil {
		defer conn.Close()
	}

	// error matching
	if err != nil {
		switch err.(type) {
		case *net.OpError:
			switch err.(*net.OpError).Err.(type) {
			case *net.DNSError:
				return &Status{Done: false, Message: "could not reach host"}
			case *os.SyscallError:
				return &Status{Done: false, Message: "connection refused"}
			case *net.AddrError:
				return &Status{Done: false, Message: "connection refused"}
			default:
				return &Status{Error: err}
			}
		default:
			return &Status{Error: err}
		}
	}

	// writing to the connection
	if t.Write != "" {
		// add a newline to emulate the behavior of echo | nc
		if !strings.HasSuffix(t.Write, "\n") {
			t.Write += "\n"
		}
		// wait a maximum of IOtimeout to write our message
		conn.SetDeadline(time.Now().Add(t.IOTimeout))
		_, err = conn.Write([]byte(t.Write))
		if err != nil {
			return &Status{Done: true, Error: err}
		}
	}

	// content matching
	if t.Content != "" {
		var content []byte

		// handle the simple case: either match all or time out
		if t.EntireContent {
			conn.SetDeadline(time.Now().Add(t.IOTimeout))
			content, err := ioutil.ReadAll(conn)
			if err != nil {
				if strings.Contains(err.Error(), "i/o timeout") {
					return &Status{Done: true, Error: errNoMatchTimeout}
				}
				return &Status{Done: true, Error: err}
			}
			if bytes.Equal(content, []byte(t.Content)) {
				return &Status{Done: true, Message: "service available"}
			}
			return &Status{Done: true, Error: errNoMatch}
		}

		// match the content progressively as it arrives

		read := make(chan byte)  // the bytes we've read so far
		eof := make(chan bool)   // have we read all of them?
		errs := make(chan error) // have we encountered an error while reading?
		reader := bufio.NewReader(conn)

		// send all the bytes down a channel as they come over the network
		go func(rdr *bufio.Reader, out chan byte, done chan bool) {
			brk := false
			for {
				if brk {
					break
				}
				conn.SetDeadline(time.Now().Add(t.IOTimeout))
				b, err := rdr.ReadByte()
				switch {
				case err == nil: // breaks switch
				case err == io.EOF:
					eof <- true
					brk = true
				case strings.Contains(err.Error(), "i/o timeout"):
					errs <- errNoMatchTimeout
					brk = true
				default:
					errs <- err
					brk = true
				}
				out <- b
			}
		}(reader, read, eof)

		matched := make(chan bool)
		// progressively read data until we hit EOF, timeout, or desired string
		for {
			// this won't loop infinitely b/c there's a timeout on the err chan
			select {
			case <-matched: // have we already matched?
				return &Status{Done: true, Message: "service available"}
			case err := <-errs: // or encountered an error in reading?
				return &Status{Done: true, Error: err}
			case b := <-read:
				// read a byte
				content = append(content, b)
				// does it match?
				go func(current, expected []byte, match chan bool) {
					if bytes.Contains(current, expected) {
						match <- true
					}
				}(content, []byte(t.Content), matched)
			case <-eof: // we've read everything and not gotten a match
				return &Status{Done: true, Error: errNoMatch}
			}
		}
	}

	// we're not matching content, and there was no connection error!
	return &Status{Done: true, Message: "service available"}
}
