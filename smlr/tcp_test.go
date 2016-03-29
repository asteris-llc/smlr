package smlr

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

// only let failing test take .5s
var half, _ = time.ParseDuration(".5s")

// tcpServer accepts a single connection and writes response, returning the port
// it's waiting on
func tcpServer(response []byte, t *testing.T) (port int) {
	// listen on a random port
	var listen net.Listener
	err := errors.New("")
	for i := 0; i < 5 && err != nil; i++ {
		port = rand.Intn(10000) + 1024
		listen, err = net.Listen("tcp", fmt.Sprintf(":%v", port))
	}
	if err != nil {
		t.Errorf("Error while creating TCP server: %v", err)
		return port
	}

	// accept a single connection
	go func(listener net.Listener) {
		if listener != nil {
			defer listener.Close()
			conn, err := listener.Accept()
			if err != nil {
				t.Errorf("Error while accepting TCP connection: %v", err)
			} else {
				defer conn.Close()
				if _, err = conn.Write(response); err != nil {
					t.Errorf("Error while writing response to TCP connection: %v", err)
				}
			}
		}
	}(listen)

	return port // return the randomly chosen port
}

func TestTCPNotUp(t *testing.T) {
	t.Parallel()
	waiter := TCPWaiter{URL: "http://localhost:10000", IOTimeout: half}

	status := waiter.request(context.Background())
	assert.Nil(t, status.Error)
	assert.False(t, status.Done)
	assert.Equal(t, "connection refused", status.Message)
}

func TestTCPBadServer(t *testing.T) {
	t.Parallel()
	waiter := TCPWaiter{URL: "tcp://some.bad.hostname/", IOTimeout: half}

	status := waiter.request(context.Background())
	assert.Nil(t, status.Error)
	assert.False(t, status.Done)
}

func TestTCPBadContent(t *testing.T) {
	t.Parallel()
	port := tcpServer([]byte("not pong"), t)
	waiter := TCPWaiter{
		URL:           fmt.Sprintf("localhost:%v", port),
		Content:       "pong",
		EntireContent: true,
		IOTimeout:     half,
	}

	status := waiter.request(context.Background())
	assert.True(t, status.Done)
	assert.Equal(t, "no content match", status.Error.Error())
}

func TestTCPBadContentPartial(t *testing.T) {
	t.Parallel()
	port := tcpServer([]byte("ping"), t)
	waiter := TCPWaiter{
		URL:           fmt.Sprintf("localhost:%v", port),
		Content:       "pong",
		EntireContent: false,
		IOTimeout:     half,
	}

	status := waiter.request(context.Background())
	assert.True(t, status.Done)
	assert.Equal(t, "no content match", status.Error.Error())
}

func TestTCPGoodContent(t *testing.T) {
	t.Parallel()
	port := tcpServer([]byte("pong"), t)
	waiter := TCPWaiter{
		URL:           fmt.Sprintf("localhost:%v", port),
		Content:       "pong",
		EntireContent: true,
		IOTimeout:     half,
	}

	status := waiter.request(context.Background())
	assert.Nil(t, status.Error)
	assert.True(t, status.Done)
	assert.Equal(t, "service available", status.Message)
}

func TestTCPGoodContentPartial(t *testing.T) {
	t.Parallel()
	port := tcpServer([]byte("pong etc"), t)
	waiter := TCPWaiter{
		URL:           fmt.Sprintf("localhost:%v", port),
		Content:       "pong",
		EntireContent: false,
		IOTimeout:     half,
	}

	status := waiter.request(context.Background())
	assert.Nil(t, status.Error)
	assert.True(t, status.Done)
	assert.Equal(t, "service available", status.Message)
}

func TestTCPIOTimeout(t *testing.T) {
	t.Parallel()
	port := tcpServer([]byte("pong"), t)
	waiter := TCPWaiter{
		URL:       fmt.Sprintf("localhost:%v", port),
		IOTimeout: 0,
		Content:   "pong",
	}

	status := waiter.request(context.Background())
	fmt.Println(status) // printing "service available"
	assert.True(t, status.Done)
	assert.NotNil(t, status.Error)
	if status.Error != nil {
		assert.Equal(t, "no content match within iotimeout", status.Error.Error())
	}
}

func TestTCPWrite(t *testing.T) {
	t.Parallel()
	port := tcpServer([]byte("pong"), t)
	waiter := TCPWaiter{
		URL:       fmt.Sprintf("localhost:%v", port),
		IOTimeout: half,
		Write:     "ping",
	}

	status := waiter.request(context.Background())
	assert.Nil(t, status.Error)
	assert.True(t, status.Done)
	assert.Equal(t, "service available", status.Message)
}
