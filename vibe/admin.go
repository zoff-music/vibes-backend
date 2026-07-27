package vibe

import (
	"context"
	"time"
)

type AdminRoomSummary struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	UserCount        int      `json:"userCount"`
	SongCount        int      `json:"songCount"`
	ActiveSources    []string `json:"activeSources"`
	HasAdminPassword bool     `json:"hasAdminPassword"`
}

type AdminRoomSearch struct {
	Query      string
	SortBy     AdminRoomSort
	Descending bool
	From       int
	To         int
}

type AdminRoomResult struct {
	Rooms []AdminRoomSummary `json:"rooms"`
	From  int                `json:"from"`
	To    int                `json:"to"`
	Total int                `json:"total"`
	Count int                `json:"count"`
}

type AdminRoomSort string

type AdminUpdateRoomRequest struct {
	Name               *string `json:"name,omitempty"`
	ClearAdminPassword *bool   `json:"clearAdminPassword,omitempty"`
}

type AdminLoginRequest struct {
	Password string `json:"password"`
}

type AdminSessionResponse struct {
	Authorized bool `json:"authorized"`
}

type AdminSearchUsageSummary struct {
	Window   string `json:"window"`
	Provider string `json:"provider"`
	Total    int64  `json:"total"`
	Unique   int64  `json:"unique"`
	Cached   int64  `json:"cached"`
	Live     int64  `json:"live"`
}

type AdminSearchUsage struct {
	Summaries   []AdminSearchUsageSummary `json:"summaries"`
	GeneratedAt time.Time                 `json:"generatedAt"`
}

type CachedAdminSearchUsage struct {
	Usage AdminSearchUsage
	Found bool
}

func (u *CachedAdminSearchUsage) IsEmpty() bool {
	return !u.Found
}

type AdminEvent struct {
	Type    string `json:"type"`
	Payload []byte `json:"payload"`
}

type AdminRoomLister interface {
	ListAdminRooms(ctx context.Context) ([]AdminRoomSummary, error)
}

type AdminRoomSearcher interface {
	SearchAdminRooms(ctx context.Context, search AdminRoomSearch) (*AdminRoomResult, error)
}

type AdminSearchUsageLister interface {
	ListAdminSearchUsage(ctx context.Context) ([]AdminSearchUsageSummary, error)
}

type CachedAdminSearchUsageFetcher interface {
	GetCachedAdminSearchUsage(ctx context.Context) (*CachedAdminSearchUsage, error)
}

type CachedAdminSearchUsageCreator interface {
	CacheAdminSearchUsage(ctx context.Context, usage AdminSearchUsage) error
}

type CachedAdminSearchUsageFetcherCreator interface {
	CachedAdminSearchUsageFetcher
	CachedAdminSearchUsageCreator
}

type AdminRoomUpdater interface {
	UpdateAdminRoom(ctx context.Context, roomID string, request AdminUpdateRoomRequest) (bool, error)
}

type AdminRoomDeleter interface {
	DeleteAdminRoom(ctx context.Context, roomID string) (bool, error)
}

// AdminRoomUpdaterLister updates a room and lists the resulting room state.
type AdminRoomUpdaterLister interface {
	AdminRoomLister
	AdminRoomUpdater
}

// AdminRoomDeleterLister deletes a room and lists the resulting room state.
type AdminRoomDeleterLister interface {
	AdminRoomLister
	AdminRoomDeleter
}

type AdminEventNotifier interface {
	NotifyAdminUpdate(ctx context.Context, event AdminEvent) error
}

const AdminRoomsUpdate = "admin_rooms_update"

const AdminRoomSortListeners AdminRoomSort = "listeners"

const AdminRoomSortSongs AdminRoomSort = "songs"
