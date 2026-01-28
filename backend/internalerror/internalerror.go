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
	return fmt.Sprintf("error non-recoverable: %v", e.Err)
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
	return fmt.Sprintf("expected error: %v", e.Err)
}

func (e ErrExpected) Unwrap() error {
	return e.Err
}

// ErrAlreadyVoted is an error type for errors where a user has already voted on a song.
type ErrAlreadyVoted struct {
	Err error
}

func (e ErrAlreadyVoted) Error() string {
	return fmt.Sprintf("Already voted error: %v", e.Err)
}

func (e ErrAlreadyVoted) Unwrap() error {
	return e.Err
}

// ErrDuplicateSong is an error type for errors where a duplicate song is added.
type ErrDuplicateSong struct {
	Err error
}

func (e ErrDuplicateSong) Error() string {
	return fmt.Sprintf("Duplicate song error: %v", e.Err)
}

func (e ErrDuplicateSong) Unwrap() error {
	return e.Err
}

// ErrHostModeSkipOnly is an error type for errors where a user tries to skip in host mode.
type ErrHostModeSkipOnly struct {
	Err error
}

func (e ErrHostModeSkipOnly) Error() string {
	return fmt.Sprintf("Host mode skip only error: %v", e.Err)
}

func (e ErrHostModeSkipOnly) Unwrap() error {
	return e.Err
}

// ErrSkipDisabled is an error type for errors where skipping is disabled.
type ErrSkipDisabled struct {
	Err error
}

func (e ErrSkipDisabled) Error() string {
	return fmt.Sprintf("Skip disabled error: %v", e.Err)
}

func (e ErrSkipDisabled) Unwrap() error {
	return e.Err
}
