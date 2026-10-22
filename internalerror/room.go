package internalerror

import "fmt"

// ErrRoomNameUnavailable is an error type for unavailable or expired room names.
type ErrRoomNameUnavailable struct {
	Err error
}

func (e ErrRoomNameUnavailable) Error() string {
	result := fmt.Sprintf("error room name unavailable: %v", e.Err)
	return result
}

func (e ErrRoomNameUnavailable) Unwrap() error {
	return e.Err
}

// ErrRoomGenerationBusy is an error type for when another room generation is active.
type ErrRoomGenerationBusy struct {
	Err error
}

func (e ErrRoomGenerationBusy) Error() string {
	result := fmt.Sprintf("error room generation busy: %v", e.Err)
	return result
}

func (e ErrRoomGenerationBusy) Unwrap() error {
	return e.Err
}

// ErrRoomGenerationPlaylistItemLimit is an error type for rooms with too many songs to generate a playlist.
type ErrRoomGenerationPlaylistItemLimit struct {
	Err error
}

func (e ErrRoomGenerationPlaylistItemLimit) Error() string {
	result := fmt.Sprintf("error room generation song limit exceeded: %v", e.Err)
	return result
}

func (e ErrRoomGenerationPlaylistItemLimit) Unwrap() error {
	return e.Err
}

// ErrRoomGenerationDailyLimit is an error type for rooms that reached their daily generation limit.
type ErrRoomGenerationDailyLimit struct {
	Err error
}

func (e ErrRoomGenerationDailyLimit) Error() string {
	result := fmt.Sprintf("error room generation daily limit exceeded: %v", e.Err)
	return result
}

func (e ErrRoomGenerationDailyLimit) Unwrap() error {
	return e.Err
}
