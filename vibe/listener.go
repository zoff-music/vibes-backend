package vibe

import (
	"context"
	"time"
)

type ListenerUsagePoint struct {
	Window    string    `json:"window"`
	Timestamp time.Time `json:"timestamp"`
	Listeners int       `json:"listeners"`
}

type AdminListenerUsage struct {
	RoomPoints  []RoomListenerUsagePoint `json:"roomPoints"`
	Points      []ListenerUsagePoint     `json:"points"`
	GeneratedAt time.Time                `json:"generatedAt"`
}

type RoomListenerUsagePoint struct {
	ListenerUsagePoint
	RoomType RoomType `json:"roomType,omitempty"`
	RoomID   string   `json:"roomId"`
}

type CachedAdminListenerUsage struct {
	Usage AdminListenerUsage
	Found bool
}

func (u *CachedAdminListenerUsage) IsEmpty() bool {
	return !u.Found
}

type ListenerUsageCreator interface {
	CreateListenerUsage(ctx context.Context) error
}

type AdminListenerUsageLister interface {
	ListAdminRoomListenerUsage(ctx context.Context) ([]RoomListenerUsagePoint, error)
	ListAdminListenerUsage(ctx context.Context) ([]ListenerUsagePoint, error)
}

type CachedAdminListenerUsageFetcher interface {
	GetCachedAdminListenerUsage(ctx context.Context) (*CachedAdminListenerUsage, error)
}

type CachedAdminListenerUsageCreator interface {
	CacheAdminListenerUsage(ctx context.Context, usage AdminListenerUsage) error
}

type CachedAdminListenerUsageFetcherCreator interface {
	CachedAdminListenerUsageFetcher
	CachedAdminListenerUsageCreator
}
