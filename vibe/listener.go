package vibe

import (
	"context"
	"encoding/json"
	"fmt"
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

// MarshalJSON preserves empty collections in the public JSON contract.
func (v AdminListenerUsage) MarshalJSON() ([]byte, error) {
	type payload AdminListenerUsage
	result := payload(v)

	if result.RoomPoints == nil {
		result.RoomPoints = make([]RoomListenerUsagePoint, 0)
	}

	if result.Points == nil {
		result.Points = make([]ListenerUsagePoint, 0)
	}

	body, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error marshaling AdminListenerUsage: %w", err)
	}

	return body, nil
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
