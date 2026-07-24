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
	return fmt.Sprintf("error already voted: %v", e.Err)
}

func (e ErrAlreadyVoted) Unwrap() error {
	return e.Err
}

// ErrDuplicateSong is an error type for errors where a duplicate song is added.
type ErrDuplicateSong struct {
	Err error
}

func (e ErrDuplicateSong) Error() string {
	return fmt.Sprintf("error duplicate song: %v", e.Err)
}

func (e ErrDuplicateSong) Unwrap() error {
	return e.Err
}

// ErrHostModeSkipOnly is an error type for errors where a user tries to skip in host mode.
type ErrHostModeSkipOnly struct {
	Err error
}

func (e ErrHostModeSkipOnly) Error() string {
	return fmt.Sprintf("error host mode skip only: %v", e.Err)
}

func (e ErrHostModeSkipOnly) Unwrap() error {
	return e.Err
}

// ErrSkipDisabled is an error type for errors where skipping is disabled.
type ErrSkipDisabled struct {
	Err error
}

func (e ErrSkipDisabled) Error() string {
	return fmt.Sprintf("error skip disabled: %v", e.Err)
}

func (e ErrSkipDisabled) Unwrap() error {
	return e.Err
}

// ErrAccessTokenNotFound is an error type for errors where an access token is not found.
type ErrAccessTokenNotFound struct {
	Err error
}

func (e ErrAccessTokenNotFound) Error() string {
	return fmt.Sprintf("error access token not found: %v", e.Err)
}

func (e ErrAccessTokenNotFound) Unwrap() error {
	return e.Err
}

// ErrMissingAdminPassword is an error type for errors where an admin password is required but missing.
type ErrMissingAdminPassword struct {
	Err error
}

func (e ErrMissingAdminPassword) Error() string {
	return fmt.Sprintf("error missing admin password: %v", e.Err)
}

func (e ErrMissingAdminPassword) Unwrap() error {
	return e.Err
}

// ErrRoomGenerationBusy is an error type for when another room generation is active.
type ErrRoomGenerationBusy struct {
	Err error
}

func (e ErrRoomGenerationBusy) Error() string {
	return fmt.Sprintf("error room generation busy: %v", e.Err)
}

func (e ErrRoomGenerationBusy) Unwrap() error {
	return e.Err
}

// ErrCastTokenInvalid is an error type for errors where a cast token is malformed or has an invalid signature.
type ErrCastTokenInvalid struct {
	Err error
}

func (e ErrCastTokenInvalid) Error() string {
	return fmt.Sprintf("error cast token invalid: %v", e.Err)
}

func (e ErrCastTokenInvalid) Unwrap() error {
	return e.Err
}

// ErrCastTokenExpired is an error type for errors where a cast token is expired.
type ErrCastTokenExpired struct {
	Err error
}

func (e ErrCastTokenExpired) Error() string {
	return fmt.Sprintf("error cast token expired: %v", e.Err)
}

func (e ErrCastTokenExpired) Unwrap() error {
	return e.Err
}
