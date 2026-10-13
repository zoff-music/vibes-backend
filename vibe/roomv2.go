package vibe

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// RoomSettingsV2 configures the room's playlist and playback behavior.
type RoomSettingsV2 struct {
	SkipAllowed               bool     `json:"skipAllowed"`
	DemocraticSkip            bool     `json:"democraticSkip"`
	SkipVoteThreshold         float64  `json:"skipVoteThreshold"`
	MaxContinuousAdds         int      `json:"maxContinuousAdds"`
	RemoveOnPlay              bool     `json:"removeOnPlay"`
	AllowDuplicates           bool     `json:"allowDuplicates"`
	EnabledSources            []string `json:"enabledSources"`
	OnlyAdminAddPlaylistItems bool     `json:"onlyAdminAddPlaylistItems"`
	Public                    bool     `json:"public"`
	PlaylistImport            bool     `json:"playlistImport"`
}

// MarshalJSON preserves empty collections in the public JSON contract.
func (v RoomSettingsV2) MarshalJSON() ([]byte, error) {
	type payload RoomSettingsV2
	result := payload(v)

	if result.EnabledSources == nil {
		result.EnabledSources = make([]string, 0)
	}

	body, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error marshaling RoomSettingsV2: %w", err)
	}

	return body, nil
}

type SessionResponseV2 struct {
	UserID    string  `json:"userId"`
	SessionID string  `json:"sessionId"`
	Nickname  *string `json:"nickname,omitempty"`
	IsAdmin   bool    `json:"isAdmin"`
	Room      *RoomV2 `json:"room"`
}

type RoomAdminSessionV2Creator interface {
	SessionProfileFetcherCreator
	RoomV2Fetcher
	AuthenticateAdmin(ctx context.Context, roomID, userID, password string) (*AdminAuthResult, error)
}

type RoomAdminSessionV2Deleter interface {
	RoomV2Fetcher
	ClearRoomAdmin(ctx context.Context, roomID, userID string) error
}

type RoomType string

const RoomTypeMusic RoomType = "MUSIC"

const RoomTypeWatch RoomType = "WATCH"

func (r RoomType) IsValid() bool {
	return r == RoomTypeMusic || r == RoomTypeWatch
}

func (r RoomType) AllowsSource(source string) bool {
	if r == RoomTypeWatch {
		return source == SourceTypeYouTube
	}

	return r == RoomTypeMusic && (source == SourceTypeYouTube || source == SourceTypeSoundCloud)
}

// RoomV2 is the typed room and playlist-item API contract.
type RoomV2 struct {
	ID                                     string         `json:"id"`
	Name                                   string         `json:"name"`
	Mode                                   string         `json:"mode"`
	RoomType                               RoomType       `json:"roomType"`
	HostID                                 string         `json:"hostId,omitempty"`
	AdminPasswordHash                      string         `json:"-"`
	HasPassword                            bool           `json:"hasPassword"`
	Settings                               RoomSettingsV2 `json:"settings"`
	CreatedAt                              time.Time      `json:"createdAt"`
	UserCount                              int            `json:"userCount,omitzero"`
	IsAdmin                                bool           `json:"isAdmin"`
	UserID                                 string         `json:"userId,omitempty"`
	ActiveSources                          []string       `json:"activeSources"`
	IsGenerating                           bool           `json:"isGenerating"`
	GenerationCount                        int            `json:"generationCount"`
	RoomGenerationMaxDailyCount            int            `json:"roomGenerationMaxDailyCount"`
	RoomGenerationMaxExistingPlaylistItems int            `json:"roomGenerationMaxExistingPlaylistItems"`
	GenerationError                        string         `json:"generationError,omitempty"`
	StoredEnabledSources                   []string       `json:"-"`
}

// MarshalJSON preserves empty collections in the public JSON contract.
func (v RoomV2) MarshalJSON() ([]byte, error) {
	type payload RoomV2
	result := payload(v)

	if result.ActiveSources == nil {
		result.ActiveSources = make([]string, 0)
	}

	body, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error marshaling RoomV2: %w", err)
	}

	return body, nil
}

// CreateRoomRequestV2 creates a MUSIC or WATCH room.
type CreateRoomRequestV2 struct {
	Name             string          `json:"name" minLength:"1" maxLength:"100"`
	RoomType         RoomType        `json:"roomType,omitempty" enums:"MUSIC,WATCH" default:"MUSIC"`
	Mode             string          `json:"mode,omitempty"`
	Password         string          `json:"password,omitempty"`
	ReservationToken string          `json:"reservationToken,omitempty"`
	Settings         *RoomSettingsV2 `json:"settings,omitempty"`
}

// UpdateRoomRequestV2 updates playback mode and room settings.
type UpdateRoomRequestV2 struct {
	Mode     string          `json:"mode,omitempty"`
	Settings *RoomSettingsV2 `json:"settings,omitempty"`
}

func (r *RoomV2) IsEmpty() bool { return r.ID == "" }

func (r CreateRoomRequestV2) Validate() bool {
	length := utf8.RuneCountInString(strings.TrimSpace(r.Name))

	return length > 0 && length <= RoomNameMaxLength
}

type RoomV2Fetcher interface {
	GetRoomV2(ctx context.Context, roomID string, userID string) (*RoomV2, error)
}

type RoomV2Creator interface {
	CreateRoomV2(ctx context.Context, room *RoomV2, reservationToken string) (*RoomV2, error)
	RoomExistenceChecker
}

type RoomV2SettingsUpdater interface {
	RoomV2Fetcher
	UpdateRoomV2(ctx context.Context, room *RoomV2) (*RoomV2, error)
	SessionProfileFetcherCreator
}

func DefaultRoomSettingsV2() (*RoomSettingsV2, error) {
	return &RoomSettingsV2{
		SkipAllowed:       true,
		DemocraticSkip:    true,
		SkipVoteThreshold: 0.5,
		MaxContinuousAdds: 3,
		EnabledSources:    []string{"youtube", "soundcloud"},
		PlaylistImport:    true,
	}, nil
}

func (r RoomSettingsV2) IsEmpty() bool {
	return !r.SkipAllowed &&
		!r.DemocraticSkip &&
		r.SkipVoteThreshold == 0 &&
		r.MaxContinuousAdds == 0 &&
		!r.RemoveOnPlay &&
		!r.AllowDuplicates &&
		len(r.EnabledSources) == 0 &&
		!r.OnlyAdminAddPlaylistItems &&
		!r.Public &&
		!r.PlaylistImport
}

type AddPlaylistRequestV2 struct {
	PlaylistItems []AddPlaylistItemRequest `json:"playlistItems"`
}

// MarshalJSON preserves empty collections in the public JSON contract.
func (v AddPlaylistRequestV2) MarshalJSON() ([]byte, error) {
	type payload AddPlaylistRequestV2
	result := payload(v)

	if result.PlaylistItems == nil {
		result.PlaylistItems = make([]AddPlaylistItemRequest, 0)
	}

	body, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error marshaling AddPlaylistRequestV2: %w", err)
	}

	return body, nil
}

type PlaylistImportRoomV2Creator interface {
	PlaylistImportCreator
	RoomV2Fetcher
}

type GeneratedRoomV2Creator interface {
	RoomNameSuggester
	RoomGenerationCreator
	RoomGenerationAvailabilityChecker
	CreateRoomV2(ctx context.Context, room *RoomV2, reservationToken string) (*RoomV2, error)
}
