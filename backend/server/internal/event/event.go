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
func GetAppEvents(pa vibe.ExpiredPlaybackProcessor, hm vibe.AbandonnedHostProcessor, ips vibe.RoomEventNotifier, ps vibe.ParticipantStorage) AppEvents {
	return AppEvents{
		{
			Name: "ReviewRoomPlayback",
			Rate: 500 * time.Millisecond,
			Handler: &PlaybackMonitor{
				db:  pa,
				ips: ips,
			},
		},
		{
			Name: "ReviewHostHealth",
			Rate: 500 * time.Millisecond,
			Handler: &HostMonitor{
				db:  hm,
				ips: ips,
			},
		},
		{
			Name:    "CleanupInactiveParticipants",
			Rate:    5 * time.Minute,
			Handler: &CleanupHandler{DB: ps},
		},
	}
}
