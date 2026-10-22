package internalerror

import "fmt"

// ErrRemoteUnchanged indicates that a machine report has no state changes.
type ErrRemoteUnchanged struct {
	Err error
}

func (e ErrRemoteUnchanged) Error() string {
	result := fmt.Sprintf("error remote state unchanged: %v", e.Err)
	return result
}

func (e ErrRemoteUnchanged) Unwrap() error {
	return e.Err
}
