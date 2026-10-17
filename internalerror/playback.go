package internalerror

import "fmt"

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

// ErrDuplicatePlaylistItem is an error type for errors where a duplicate song is added.
type ErrDuplicatePlaylistItem struct {
	Err error
}

func (e ErrDuplicatePlaylistItem) Error() string {
	return fmt.Sprintf("error duplicate song: %v", e.Err)
}

func (e ErrDuplicatePlaylistItem) Unwrap() error {
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
