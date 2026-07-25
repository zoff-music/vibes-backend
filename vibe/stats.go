package vibe

import "context"

// Stats contains public, service-wide usage statistics.
type Stats struct {
	TotalListeners int `json:"totalListeners"`
}

// StatsFetcher fetches public, service-wide usage statistics.
type StatsFetcher interface {
	GetStats(ctx context.Context) (*Stats, error)
}
