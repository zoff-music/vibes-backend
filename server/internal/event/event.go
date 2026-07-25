// Package event handles configuration and setup for receiving events.
//
// SSE and scheduled app events are wired in this file.
package event

import (
	"context"
	"time"

	"github.com/zoff-music/vibes-backend/client/database"
	"github.com/zoff-music/vibes-backend/client/internalpubsub"
	redisclient "github.com/zoff-music/vibes-backend/client/redis"
	"github.com/zoff-music/vibes-backend/client/soundcloud"
	"github.com/zoff-music/vibes-backend/client/spotify"
	"github.com/zoff-music/vibes-backend/client/youtube"
	"github.com/zoff-music/vibes-backend/server/internal/handler"
	"github.com/zoff-music/vibes-backend/vibe"
)

// Handler is an interface that all event handles must implement.
type Handler interface {
	Handle(ctx context.Context, data []byte) error
}

// GetSSEEvents describes all named SSE events.
func GetSSEEvents(db *database.Client, ips *internalpubsub.Client) SSEEvents {
	events := SSEEvents{
		{
			Name:       RoomEvents,
			Subscriber: ips,
			Handler: handler.RoomEventStream{
				Participants: db,
				Notifier:     ips,
			},
		},
		{
			Name:       AdminEvents,
			Subscriber: ips,
			Handler: handler.AdminEventStream{
				Rooms: db,
			},
		},
	}

	return events
}

// GetAppEvents describes all the app events to listen to.
func GetAppEvents(
	db *database.Client,
	redisClient *redisclient.Client,
	ips *internalpubsub.Client,
	soundcloudClient *soundcloud.Client,
	spotifyClient *spotify.Client,
	youtubeClient *youtube.Client,
	ai vibe.PlaylistGenerator,
	enabledProviders []string,
) AppEvents {
	events := AppEvents{
		{
			Name: "GenerateRoomPlaylist",
			Rate: 5 * time.Second,
			Handler: &handler.GenerateRoomPlaylist{
				AI:       ai,
				Cache:    redisClient,
				DB:       db,
				IPS:      ips,
				Searcher: youtubeClient,
			},
		},
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
			Name: "CleanupRoomGenerations",
			Rate: 10 * time.Minute,
			Handler: &handler.CleanupRoomGenerations{
				DB: db,
			},
		},
		{
			Name: "CleanupRoomNameReservations",
			Rate: time.Minute,
			Handler: &handler.CleanupRoomNameReservations{
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

	for _, provider := range enabledProviders {
		switch provider {
		case vibe.SourceTypeYouTube:
			events = append(events, AppEvent{
				Name: "MetaRefreshYouTube",
				Rate: time.Second,
				Handler: &handler.MetaRefresh{
					DB:           db,
					IPS:          ips,
					Provider:     youtubeClient,
					ProviderName: vibe.SourceTypeYouTube,
				},
			})
		case vibe.SourceTypeSoundCloud:
			events = append(events, AppEvent{
				Name: "MetaRefreshSoundCloud",
				Rate: time.Second,
				Handler: &handler.MetaRefresh{
					DB:           db,
					IPS:          ips,
					Provider:     soundcloudClient,
					ProviderName: vibe.SourceTypeSoundCloud,
				},
			})
		case vibe.SourceTypeSpotify:
			events = append(events, AppEvent{
				Name: "MetaRefreshSpotify",
				Rate: time.Second,
				Handler: &handler.MetaRefresh{
					DB:           db,
					IPS:          ips,
					Provider:     spotifyClient,
					ProviderName: vibe.SourceTypeSpotify,
				},
			})
		}
	}

	return events
}

const RoomEvents = "RoomEvents"
const AdminEvents = "AdminEvents"
