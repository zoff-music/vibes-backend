// Package event handles configuration and setup for receiving events.
//
// Events to subscribe to should be defined in GetPubSubEvents
package event

import (
	"context"
	"time"

	"github.com/zoff-music/vibes-backend/client/database"
	"github.com/zoff-music/vibes-backend/client/internalpubsub"
	"github.com/zoff-music/vibes-backend/client/spotify"
	"github.com/zoff-music/vibes-backend/client/youtube"
	"github.com/zoff-music/vibes-backend/server/internal/handler"
)

// Handler is an interface that all event handles must implement.
type Handler interface {
	Handle(ctx context.Context, data []byte) error
}

// GetAppEvents describes all the app events to listen to.
func GetAppEvents(
	db *database.Client,
	ips *internalpubsub.Client,
	spotifyClient *spotify.Client,
	youtubeClient *youtube.Client,
) AppEvents {
	return AppEvents{
		{
			Name: "ReviewRoomPlayback",
			Rate: 500 * time.Millisecond,
			Handler: &handler.ReviewRoomPlayback{
				DB:  db,
				IPS: ips,
			},
		},
		{
			Name: "ReviewHostHealth",
			Rate: 500 * time.Millisecond,
			Handler: &handler.ReviewHostHealth{
				DB:  db,
				IPS: ips,
			},
		},
		{
			Name: "CleanupInactiveParticipants",
			Rate: 10 * time.Second,
			Handler: &handler.CleanupInactiveParticipants{
				DB: db,
			},
		},
		{
			Name: "CleanupExpiredTokens",
			Rate: 10 * time.Second,
			Handler: &handler.CleanupExpiredTokens{
				DB: db,
			},
		},
		{
			Name: "RefreshSpotifyTokens",
			Rate: 10 * time.Second,
			Handler: &handler.RefreshSpotifyTokens{
				DB:       db,
				Provider: spotifyClient,
			},
		},
		{
			Name: "RefreshYouTubeTokens",
			Rate: 10 * time.Second,
			Handler: &handler.RefreshYouTubeTokens{
				DB:       db,
				Provider: youtubeClient,
			},
		},
		{
			Name: "CleanupExpiredPendingOAuthStates",
			Rate: 10 * time.Second,
			Handler: &handler.CleanupExpiredPendingOAuthStates{
				DB: db,
			},
		},
		{
			Name: "ReviewAdminRooms",
			Rate: 15 * time.Second,
			Handler: &handler.ReviewAdminRooms{
				DB:  db,
				IPS: ips,
			},
		},
	}
}
