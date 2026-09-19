package vibe

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"
)

const MessageEvent = "message"

type RoomMessage struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Name      string `json:"name"`
	IsAdmin   bool   `json:"isAdmin"`
	Kind      string `json:"kind"`
	Text      string `json:"text"`
	CreatedAt int64  `json:"createdAt"`
}

type CreateMessageRequest struct {
	Text string `json:"text"`
}

func (r CreateMessageRequest) Validate() bool {
	length := utf8.RuneCountInString(strings.TrimSpace(r.Text))
	return length > 0 && length <= 500
}

type MessageAuthorFetcher interface {
	RoomFetcher
	SessionProfileFetcherCreator
}

type SessionProfileRoomUpdater interface {
	SessionProfileUpdater
	SessionProfileFetcherCreator
	RoomFetcher
	GetSessionRooms(ctx context.Context, userID string) ([]string, error)
}

type MessageUsageCreator interface {
	CreateMessageUsage(ctx context.Context, roomID string, sentAt time.Time) error
}

type MessageAuthorFetcherUsageCreator interface {
	MessageAuthorFetcher
	MessageUsageCreator
}

type MessageUsagePoint struct {
	Window    string    `json:"window"`
	Timestamp time.Time `json:"timestamp"`
	Messages  int64     `json:"messages"`
}

type AdminMessageUsage struct {
	RoomID      string              `json:"roomId"`
	Total       int64               `json:"total"`
	Points      []MessageUsagePoint `json:"points"`
	GeneratedAt time.Time           `json:"generatedAt"`
}

type AdminMessageUsageLister interface {
	ListAdminMessageUsage(ctx context.Context, roomID string) (*AdminMessageUsage, error)
}
