package internalerror

import "fmt"

// ErrAccessTokenNotFound is an error type for errors where an access token is not found.
type ErrAccessTokenNotFound struct {
	Err error
}

func (e ErrAccessTokenNotFound) Error() string {
	result := fmt.Sprintf("error access token not found: %v", e.Err)
	return result
}

func (e ErrAccessTokenNotFound) Unwrap() error {
	return e.Err
}
