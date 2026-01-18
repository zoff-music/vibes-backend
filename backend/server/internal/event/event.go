// Package event handles configuration and setup for receiving events.
//
// Events to subscribe to should be defined in GetPubSubEvents
package event

import (
	"context"
	"time"

	"github.com/zoff-music/vibes/vibe"
)

// Handler is an interface that all event handles must implement.
type Handler interface {
	Handle(ctx context.Context, data []byte) error
}

// GetAppEvents describes all the app events to listen to.
func GetAppEvents(db vibe.RoomActioner, ips vibe.RoomEventNotifier, ps vibe.ParticipantStorage) AppEvents {
	appEvents := AppEvents{}

	appEvents = append(appEvents, AppEvent{
		Name:    "ReviewRoomPlayback",
		Rate:    500 * time.Millisecond,
		Handler: NewPlaybackMonitorHandler(db, ips),
	})

	appEvents = append(appEvents, AppEvent{
		Name:    "CleanupInactiveParticipants",
		Rate:    5 * time.Minute,
		Handler: NewCleanupHandler(ps),
	})

	return appEvents
}
