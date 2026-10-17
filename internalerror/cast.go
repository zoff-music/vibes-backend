package internalerror

import "fmt"

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
