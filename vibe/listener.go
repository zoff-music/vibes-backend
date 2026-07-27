package vibe

import (
	"context"
	"time"
)

type ListenerUsagePoint struct {
	Window    string    `json:"window"`
	Timestamp time.Time `json:"timestamp"`
	Listeners int64     `json:"listeners"`
}

type AdminListenerUsage struct {
	Points      []ListenerUsagePoint `json:"points"`
	GeneratedAt time.Time            `json:"generatedAt"`
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
