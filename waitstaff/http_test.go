package waitstaff

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func serverStatusBody(status int, body string) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(body))
	})
	return httptest.NewServer(mux)
}

func TestHTTPWaiterRequestNotUp(t *testing.T) {
	waiter := HTTPWaiter{Method: "GET", URL: "http://localhost:10000"}

	status := waiter.request(context.Background())
	assert.Nil(t, status.Error)
	assert.False(t, status.Done)
	assert.Equal(t, "connection refused", status.Message)
}

func TestHTTPWaiterRequestBadServer(t *testing.T) {
	waiter := HTTPWaiter{Method: "GET", URL: "http://some.bad.hostname/"}

	status := waiter.request(context.Background())
	assert.Nil(t, status.Error)
	assert.False(t, status.Done)
	assert.Equal(t, "could not reach host", status.Message)
}

func TestHTTPWaiterRequestBadStatus(t *testing.T) {
	t.Parallel()

	server := serverStatusBody(http.StatusServiceUnavailable, "")
	defer server.Close()

	waiter := HTTPWaiter{Method: "GET", URL: server.URL, ExpectedStatus: 200}

	status := waiter.request(context.Background())
	assert.Nil(t, status.Error)
	assert.False(t, status.Done)
	assert.Equal(t, `status "503 Service Unavailable" does not match expected status (200)`, status.Message)
}

func TestHTTPWaiterRequestGoodStatus(t *testing.T) {
	t.Parallel()

	server := serverStatusBody(http.StatusOK, "pong")
	defer server.Close()

	waiter := HTTPWaiter{
		Method:         "GET",
		URL:            server.URL,
		ExpectedStatus: 200,
		Content:        "pong",
		EntireContent:  true,
	}

	status := waiter.request(context.Background())
	assert.Nil(t, status.Error)
	assert.True(t, status.Done)
	assert.Equal(t, "service available", status.Message)
}

func TestHTTPWaiterRequestBadContent(t *testing.T) {
	t.Parallel()

	server := serverStatusBody(http.StatusOK, "")
	defer server.Close()

	waiter := HTTPWaiter{
		Method:         "GET",
		URL:            server.URL,
		ExpectedStatus: http.StatusOK,
		Content:        "pong",
		EntireContent:  true,
	}

	status := waiter.request(context.Background())
	assert.Nil(t, status.Error)
	assert.False(t, status.Done)
	assert.Equal(t, "response does not match content", status.Message)
}

func TestHTTPWaiterRequestBadContentPartial(t *testing.T) {
	t.Parallel()

	server := serverStatusBody(http.StatusOK, "")
	defer server.Close()

	waiter := HTTPWaiter{
		Method:         "GET",
		URL:            server.URL,
		ExpectedStatus: http.StatusOK,
		Content:        "pong",
		EntireContent:  false,
	}

	status := waiter.request(context.Background())
	assert.Nil(t, status.Error)
	assert.False(t, status.Done)
	assert.Equal(t, "response does not contain content", status.Message)
}
