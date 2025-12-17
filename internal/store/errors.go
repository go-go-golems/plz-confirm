package store

import "github.com/pkg/errors"

var (
	// ErrNotFound is returned when a request does not exist.
	ErrNotFound = errors.New("request not found")

	// ErrAlreadyCompleted is returned when completing a non-pending request.
	ErrAlreadyCompleted = errors.New("request already completed")

	// ErrWaitTimeout is returned when waiting for a request times out.
	ErrWaitTimeout = errors.New("timeout waiting for response")
)
