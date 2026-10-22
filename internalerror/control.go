package internalerror

import "fmt"

// ErrNonRecoverable is an error type that indicates a permanent issue,
// meaning the related message will never lead to a successful outcome.
// This will prompt the pubsub event receiver to ack the message,
// removing it from the queue.
type ErrNonRecoverable struct {
	Err error
}

func (e ErrNonRecoverable) Error() string {
	result := fmt.Sprintf("error non-recoverable: %v", e.Err)
	return result
}

func (e ErrNonRecoverable) Unwrap() error {
	return e.Err
}

// ErrExpected is an error type for errors that are expected and don't
// need to appear in metrics.
type ErrExpected struct {
	Err error
}

func (e ErrExpected) Error() string {
	result := fmt.Sprintf("expected error: %v", e.Err)
	return result
}

func (e ErrExpected) Unwrap() error {
	return e.Err
}
