package vibe

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"
)

const MessageEvent = "message"

const SettingsActivityEvent = "settings_activity"

const MessageMaxLength = 500

const MessageKindChat = "chat"

const MessageKindAdded = "added"

const MessageKindVoted = "voted"

const MessageKindDeleted = "deleted"

const MessageKindSkipped = "skipped"

const MessageKindSkipVoted = "skipvoted"

const MessageKindRenamed = "renamed"

const MessageKindSettings = "settings"

type RoomMessage struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Name      string `json:"name"`
	IsAdmin   bool   `json:"isAdmin"`
	Kind      string `json:"kind"`
	Activity  bool   `json:"activity,omitempty"`
	Text      string `json:"text"`
	CreatedAt int64  `json:"createdAt"`
}

type CreateMessageRequest struct {
	Text string `json:"text" minLength:"1" maxLength:"500"`
}

func (r CreateMessageRequest) Validate() bool {
	length := utf8.RuneCountInString(strings.TrimSpace(r.Text))

	return length > 0 && length <= MessageMaxLength
}

type MessageAuthorFetcher interface {
	RoomFetcher
	SessionProfileFetcherCreator
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
	Messages  int       `json:"messages"`
}

type AdminMessageUsage struct {
	RoomID      string              `json:"roomId"`
	Total       int                 `json:"total"`
	Points      []MessageUsagePoint `json:"points"`
	GeneratedAt time.Time           `json:"generatedAt"`
}

type AdminMessageUsageLister interface {
	ListAdminMessageUsage(ctx context.Context, roomID string) (*AdminMessageUsage, error)
}
