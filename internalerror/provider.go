package internalerror

import "fmt"
import "time"

// ErrProviderQuotaExceeded indicates that an external provider's quota is exhausted.
type ErrProviderQuotaExceeded struct {
	Err      error
	Provider string
	ResetAt  time.Time
}

func (e ErrProviderQuotaExceeded) Error() string {
	return fmt.Sprintf("error %s provider quota exceeded: %v", e.Provider, e.Err)
}

func (e ErrProviderQuotaExceeded) Unwrap() error {
	return e.Err
}

// ErrProviderPlaylistInvalid indicates that a URL resolves to a different resource.
type ErrProviderPlaylistInvalid struct {
	Err error
}

func (e ErrProviderPlaylistInvalid) Error() string {
	return e.Err.Error()
}

func (e ErrProviderPlaylistInvalid) Unwrap() error {
	return e.Err
}

// ErrProviderItemNotFound indicates that a provider no longer returns a track.
type ErrProviderItemNotFound struct {
	Err error
}

func (e ErrProviderItemNotFound) Error() string {
	return fmt.Sprintf("error music track not found: %v", e.Err)
}

func (e ErrProviderItemNotFound) Unwrap() error {
	return e.Err
}

// ErrMadeForKids indicates that YouTube designates a video as made for kids.
type ErrMadeForKids struct {
	Err error
}

func (e ErrMadeForKids) Error() string {
	return fmt.Sprintf("error made-for-kids video: %v", e.Err)
}

func (e ErrMadeForKids) Unwrap() error {
	return e.Err
}

// ErrLiveVideo indicates that a provider track is a live or upcoming video.
type ErrLiveVideo struct {
	Err error
}

func (e ErrLiveVideo) Error() string {
	return fmt.Sprintf("error live video: %v", e.Err)
}

func (e ErrLiveVideo) Unwrap() error {
	return e.Err
}
