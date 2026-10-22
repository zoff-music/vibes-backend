package internalerror

import "fmt"

// ErrMissingAdminPassword is an error type for errors where an admin password is required but missing.
type ErrMissingAdminPassword struct {
	Err error
}

func (e ErrMissingAdminPassword) Error() string {
	result := fmt.Sprintf("error missing admin password: %v", e.Err)
	return result
}

func (e ErrMissingAdminPassword) Unwrap() error {
	return e.Err
}

// ErrAdminUsernameUnavailable indicates that an admin username is already used.
type ErrAdminUsernameUnavailable struct {
	Err error
}

func (e ErrAdminUsernameUnavailable) Error() string {
	result := fmt.Sprintf("error admin username unavailable: %v", e.Err)
	return result
}

func (e ErrAdminUsernameUnavailable) Unwrap() error {
	return e.Err
}
