package waitstaff

import (
	"time"

	"golang.org/x/net/context"
)

// Waiter implements
type Waiter interface {
	Wait(ctx context.Context, interval time.Duration, timeout time.Duration) chan *Status
}

// Status is returned from waiters
type Status struct {
	Done    bool
	Message string
	Error   error
}
