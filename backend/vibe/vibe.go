package vibe

import (
	"context"
	"time"
)

// Subscription defines interface to use the internalpubsub client
type Subscription interface {
	Listen() chan []byte
	Destroy()
}

type SubscriptionContainer struct {
	Subscription Subscription
}

// Publisher relays messages to subscribed clients
type Publisher interface {
	PublishToInternalSubscription(ctx context.Context, topic string, data []byte) error
}

// Subscriber subscribes to listen for room events
type Subscriber interface {
	Subscribe(topic string) (*SubscriptionContainer, error)
}

// SourceType represents the type of music source
type SourceType string

const (
	SourceTypeYouTube    SourceType = "youtube"
	SourceTypeSpotify    SourceType = "spotify"
	SourceTypeSoundCloud SourceType = "soundcloud"
)

// RoomSettings holds configuration for a room
type RoomSettings struct {
	SkipAllowed       bool    `json:"skipAllowed"`
	DemocraticSkip    bool    `json:"democraticSkip"`
	SkipVoteThreshold float64 `json:"skipVoteThreshold"`
	MaxContinuousAdds int     `json:"maxContinuousAdds"`
	RemoveOnPlay      bool    `json:"removeOnPlay"`
	LoopQueue         bool    `json:"loopQueue"`
	AllowDuplicates   bool    `json:"allowDuplicates"`
}

// DefaultRoomSettings returns sensible defaults
func DefaultRoomSettings() RoomSettings {
	return RoomSettings{
		SkipAllowed:       true,
		DemocraticSkip:    true,
		SkipVoteThreshold: 0.5,
		MaxContinuousAdds: 3,
		RemoveOnPlay:      true,
		LoopQueue:         false,
		AllowDuplicates:   false,
	}
}

// Room represents a music room
type Room struct {
	ID                string       `json:"id"`
	Name              string       `json:"name"`
	AdminPasswordHash string       `json:"-"`
	HasPassword       bool         `json:"hasPassword"`
	Settings          RoomSettings `json:"settings"`
	CreatedAt         time.Time    `json:"createdAt"`
	UserCount         int          `json:"userCount,omitempty"`
}

// IsEmpty returns true if the room is empty/not found
func (r *Room) IsEmpty() bool {
	return r.ID == ""
}

// Song represents a song in the queue
type Song struct {
	ID              string     `json:"id"`
	RoomID          string     `json:"-"`
	SourceType      SourceType `json:"sourceType"`
	SourceID        string     `json:"sourceId"`
	Title           string     `json:"title"`
	Artist          *string    `json:"artist,omitempty"`
	ThumbnailURL    string     `json:"thumbnailUrl"`
	Duration        int        `json:"duration"`
	AddedBy         string     `json:"addedBy"`
	AddedByNickname *string    `json:"addedByNickname,omitempty"`
	AddedAt         time.Time  `json:"addedAt"`
	Position        int        `json:"position"`
}

// IsEmpty returns true if the song is empty/not found
func (s *Song) IsEmpty() bool {
	return s.ID == ""
}

// PlaybackState represents the current playback state of a room
type PlaybackState struct {
	RoomID        string    `json:"-"`
	CurrentSongID *string   `json:"currentSongId"`
	CurrentSong   *Song     `json:"currentSong,omitempty"`
	IsPlaying     bool      `json:"isPlaying"`
	PositionMs    int64     `json:"positionMs"`
	UpdatedAt     time.Time `json:"updatedAt"`
	ServerTimeMs  int64     `json:"serverTimeMs"`
}

// User represents a user session in a room
type User struct {
	ID         string    `json:"id"`
	RoomID     string    `json:"-"`
	Nickname   *string   `json:"nickname,omitempty"`
	IsAdmin    bool      `json:"isAdmin"`
	JoinedAt   time.Time `json:"joinedAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
}

// IsEmpty returns true if the user is empty/not found
func (u *User) IsEmpty() bool {
	return u.ID == ""
}

// SessionResponse is returned when creating a session
type SessionResponse struct {
	UserID   string  `json:"userId"`
	Nickname *string `json:"nickname,omitempty"`
	IsAdmin  bool    `json:"isAdmin"`
	Room     *Room   `json:"room"`
}

// SkipVote tracks skip votes for a song
type SkipVote struct {
	SongID string
	UserID string
}

// --- Interfaces for Room operations ---

// RoomFetcher fetches room data
type RoomFetcher interface {
	GetRoom(ctx context.Context, id string) (*Room, error)
}

// RoomCreator creates rooms
type RoomCreator interface {
	CreateRoom(ctx context.Context, room *Room) (*Room, error)
}

// RoomUpdater updates room data
type RoomUpdater interface {
	UpdateRoom(ctx context.Context, room *Room) (*Room, error)
}

// --- Interfaces for Song operations ---

// SongsFetcher fetches songs from the queue
type SongsFetcher interface {
	GetSongs(ctx context.Context, roomID string) ([]Song, error)
	GetSong(ctx context.Context, roomID, songID string) (*Song, error)
}

// SongsModifier modifies the song queue
type SongsModifier interface {
	AddSong(ctx context.Context, song *Song) (*Song, error)
	RemoveSong(ctx context.Context, roomID, songID string) error
	ReorderSongs(ctx context.Context, roomID, songID string, newPosition int) error
	GetNextSong(ctx context.Context, roomID string, currentPosition int) (*Song, error)
}

// --- Interfaces for Playback operations ---

// PlaybackFetcher fetches playback state
type PlaybackFetcher interface {
	GetPlaybackState(ctx context.Context, roomID string) (*PlaybackState, error)
}

// PlaybackController controls playback
type PlaybackController interface {
	UpsertPlaybackState(ctx context.Context, state *PlaybackState) error
}

// --- Interfaces for User operations ---

// UserFetcher fetches user data
type UserFetcher interface {
	GetUser(ctx context.Context, roomID, userID string) (*User, error)
	GetUsersInRoom(ctx context.Context, roomID string) ([]User, error)
	CountUsersInRoom(ctx context.Context, roomID string) (int, error)
}

// UserManager manages users in rooms
type UserManager interface {
	CreateUser(ctx context.Context, user *User) (*User, error)
	UpdateUserLastSeen(ctx context.Context, roomID, userID string) error
	RemoveUser(ctx context.Context, roomID, userID string) error
	CleanupInactiveUsers(ctx context.Context, roomID string, threshold time.Duration) error
}

// --- Interfaces for Skip Vote operations ---

// SkipVoteFetcher fetches skip votes
type SkipVoteFetcher interface {
	GetSkipVotes(ctx context.Context, roomID, songID string) ([]SkipVote, error)
	HasUserVoted(ctx context.Context, roomID, songID, userID string) (bool, error)
}

// SkipVoteManager manages skip votes
type SkipVoteManager interface {
	AddSkipVote(ctx context.Context, roomID, songID, userID string) error
	ClearSkipVotes(ctx context.Context, roomID, songID string) error
}

// --- SSE Event Types ---

// EventType represents the type of SSE event
type EventType string

const (
	EventTypePlaybackUpdate EventType = "playback_update"
	EventTypeSongAdded      EventType = "song_added"
	EventTypeSongRemoved    EventType = "song_removed"
	EventTypeQueueReordered EventType = "queue_reordered"
	EventTypeUserJoined     EventType = "user_joined"
	EventTypeUserLeft       EventType = "user_left"
	EventTypeSettingsUpdate EventType = "settings_update"
)

// RoomEvent represents an SSE event for a room
type RoomEvent struct {
	Type    EventType   `json:"type"`
	Payload interface{} `json:"payload"`
}

// RoomEventBroadcaster broadcasts events to room subscribers
type RoomEventBroadcaster interface {
	BroadcastToRoom(ctx context.Context, roomID string, event *RoomEvent) error
}
