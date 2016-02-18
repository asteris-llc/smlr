package smlr

// Status is returned from waiters
type Status struct {
	Done    bool
	Message string
	Error   error
}
