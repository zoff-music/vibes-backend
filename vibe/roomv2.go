package vibe

import (
	"context"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// RoomSettingsV2 is the playlist-item API contract; RoomSettings remains unchanged.
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

func (r RoomSettings) ToRoomSettingsV2() *RoomSettingsV2 {
	return &RoomSettingsV2{
		SkipAllowed:               r.SkipAllowed,
		DemocraticSkip:            r.DemocraticSkip,
		SkipVoteThreshold:         r.SkipVoteThreshold,
		MaxContinuousAdds:         r.MaxContinuousAdds,
		RemoveOnPlay:              r.RemoveOnPlay,
		AllowDuplicates:           r.AllowDuplicates,
		EnabledSources:            slices.Clone(r.EnabledSources),
		OnlyAdminAddPlaylistItems: r.OnlyAdminAddSongs,
		Public:                    r.Public,
		PlaylistImport:            r.PlaylistImport,
	}
}

func (r RoomSettingsV2) ToRoomSettings() *RoomSettings {
	return &RoomSettings{
		SkipAllowed:       r.SkipAllowed,
		DemocraticSkip:    r.DemocraticSkip,
		SkipVoteThreshold: r.SkipVoteThreshold,
		MaxContinuousAdds: r.MaxContinuousAdds,
		RemoveOnPlay:      r.RemoveOnPlay,
		AllowDuplicates:   r.AllowDuplicates,
		EnabledSources:    slices.Clone(r.EnabledSources),
		OnlyAdminAddSongs: r.OnlyAdminAddPlaylistItems,
		Public:            r.Public,
		PlaylistImport:    r.PlaylistImport,
	}
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

// RoomV2 is the playlist-item API contract; Room remains unchanged.
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

func (r Room) ToRoomV2() *RoomV2 {
	settings := r.Settings.ToRoomSettingsV2()

	return &RoomV2{
		ID:                                     r.ID,
		Name:                                   r.Name,
		Mode:                                   r.Mode,
		RoomType:                               RoomTypeMusic,
		HostID:                                 r.HostID,
		AdminPasswordHash:                      r.AdminPasswordHash,
		HasPassword:                            r.HasPassword,
		Settings:                               *settings,
		CreatedAt:                              r.CreatedAt,
		UserCount:                              r.UserCount,
		IsAdmin:                                r.IsAdmin,
		UserID:                                 r.UserID,
		ActiveSources:                          slices.Clone(r.ActiveSources),
		IsGenerating:                           r.IsGenerating,
		GenerationCount:                        r.GenerationCount,
		RoomGenerationMaxDailyCount:            r.RoomGenerationMaxDailyCount,
		RoomGenerationMaxExistingPlaylistItems: r.RoomGenerationMaxExistingSongs,
		GenerationError:                        r.GenerationError,
		StoredEnabledSources:                   slices.Clone(r.StoredEnabledSources),
	}
}

func (r RoomV2) ToRoom() *Room {
	settings := r.Settings.ToRoomSettings()

	return &Room{
		ID:                             r.ID,
		Name:                           r.Name,
		Mode:                           r.Mode,
		HostID:                         r.HostID,
		AdminPasswordHash:              r.AdminPasswordHash,
		HasPassword:                    r.HasPassword,
		Settings:                       *settings,
		CreatedAt:                      r.CreatedAt,
		UserCount:                      r.UserCount,
		IsAdmin:                        r.IsAdmin,
		UserID:                         r.UserID,
		ActiveSources:                  slices.Clone(r.ActiveSources),
		IsGenerating:                   r.IsGenerating,
		GenerationCount:                r.GenerationCount,
		RoomGenerationMaxDailyCount:    r.RoomGenerationMaxDailyCount,
		RoomGenerationMaxExistingSongs: r.RoomGenerationMaxExistingPlaylistItems,
		GenerationError:                r.GenerationError,
		StoredEnabledSources:           slices.Clone(r.StoredEnabledSources),
	}
}

// CreateRoomRequestV2 is the playlist-item API contract; CreateRoomRequest remains unchanged.
type CreateRoomRequestV2 struct {
	Name             string          `json:"name" minLength:"1" maxLength:"100"`
	RoomType         RoomType        `json:"roomType,omitempty" enums:"MUSIC,WATCH" default:"MUSIC"`
	Mode             string          `json:"mode,omitempty"`
	Password         string          `json:"password,omitempty"`
	ReservationToken string          `json:"reservationToken,omitempty"`
	Settings         *RoomSettingsV2 `json:"settings,omitempty"`
}

func (r CreateRoomRequest) ToCreateRoomRequestV2() *CreateRoomRequestV2 {
	var settings *RoomSettingsV2
	if r.Settings != nil {
		settings = r.Settings.ToRoomSettingsV2()
	}

	return &CreateRoomRequestV2{
		Name:             r.Name,
		Mode:             r.Mode,
		Password:         r.Password,
		ReservationToken: r.ReservationToken,
		Settings:         settings,
	}
}

func (r CreateRoomRequestV2) ToCreateRoomRequest() *CreateRoomRequest {
	var settings *RoomSettings
	if r.Settings != nil {
		settings = r.Settings.ToRoomSettings()
	}

	return &CreateRoomRequest{
		Name:             r.Name,
		Mode:             r.Mode,
		Password:         r.Password,
		ReservationToken: r.ReservationToken,
		Settings:         settings,
	}
}

// UpdateRoomRequestV2 is the playlist-item API contract; UpdateRoomRequest remains unchanged.
type UpdateRoomRequestV2 struct {
	Mode     string          `json:"mode,omitempty"`
	Settings *RoomSettingsV2 `json:"settings,omitempty"`
}

func (r UpdateRoomRequest) ToUpdateRoomRequestV2() *UpdateRoomRequestV2 {
	var settings *RoomSettingsV2
	if r.Settings != nil {
		settings = r.Settings.ToRoomSettingsV2()
	}

	return &UpdateRoomRequestV2{
		Mode:     r.Mode,
		Settings: settings,
	}
}

func (r UpdateRoomRequestV2) ToUpdateRoomRequest() *UpdateRoomRequest {
	var settings *RoomSettings
	if r.Settings != nil {
		settings = r.Settings.ToRoomSettings()
	}

	return &UpdateRoomRequest{
		Mode:     r.Mode,
		Settings: settings,
	}
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
