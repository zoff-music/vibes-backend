package internalerror

import "fmt"

// ErrCastTokenInvalid is an error type for errors where a cast token is malformed or has an invalid signature.
type ErrCastTokenInvalid struct {
	Err error
}

func (e ErrCastTokenInvalid) Error() string {
	result := fmt.Sprintf("error cast token invalid: %v", e.Err)
	return result
}

func (e ErrCastTokenInvalid) Unwrap() error {
	return e.Err
}

// ErrCastTokenExpired is an error type for errors where a cast token is expired.
type ErrCastTokenExpired struct {
	Err error
}

func (e ErrCastTokenExpired) Error() string {
	result := fmt.Sprintf("error cast token expired: %v", e.Err)
	return result
}

func (e ErrCastTokenExpired) Unwrap() error {
	return e.Err
}
