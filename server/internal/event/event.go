// Package event handles configuration and setup for receiving events.
//
// Scheduled application events are defined in GetAppEvents.
package event

import (
	"context"
	"time"

	"github.com/zoff-music/vibes-backend/client/database"
	redisclient "github.com/zoff-music/vibes-backend/client/redis"
	"github.com/zoff-music/vibes-backend/client/soundcloud"
	"github.com/zoff-music/vibes-backend/client/youtube"
	"github.com/zoff-music/vibes-backend/server/internal/handler"
	"github.com/zoff-music/vibes-backend/vibe"
)

// Handler is an interface that all event handles must implement.
type Handler interface {
	Handle(ctx context.Context, data []byte) error
}

// GetAppEvents describes all the app events to listen to.
func GetAppEvents(
	db *database.Client,
	redisClient *redisclient.Client,
	soundcloudClient *soundcloud.Client,
	youtubeClient *youtube.Client,
	ai vibe.PlaylistGenerator,
	enabledProviders []string,
) AppEvents {
	events := AppEvents{
		{
			Name: "ImportPlaylistSong",
			Rate: 100 * time.Millisecond,
			Handler: &handler.ImportPlaylistSong{
				DB:     db,
				Events: redisClient,
			},
		},
		{
			Name: "GenerateRoomPlaylist",
			Rate: 5 * time.Second,
			Handler: &handler.GenerateRoomPlaylist{
				AI:       ai,
				Cache:    redisClient,
				DB:       db,
				Searcher: youtubeClient,
			},
		},
		{
			Name: "ReviewRoomPlayback",
			Rate: 100 * time.Millisecond,
			Handler: &handler.ReviewRoomPlayback{
				DB:     db,
				Events: redisClient,
			},
		},
		{
			Name: "ReviewHostHealth",
			Rate: 500 * time.Millisecond,
			Handler: &handler.ReviewHostHealth{
				DB:     db,
				Events: redisClient,
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
			Name: "TrackListenerUsage",
			Rate: time.Minute,
			Handler: &handler.TrackListenerUsage{
				DB: db,
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
				DB:     db,
				Events: redisClient,
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
					Events:       redisClient,
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
					Events:       redisClient,
					Provider:     soundcloudClient,
					ProviderName: vibe.SourceTypeSoundCloud,
				},
			})
		}
	}

	return events
}
