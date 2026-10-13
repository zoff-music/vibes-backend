package vibe

import (
	"context"
	"encoding/json"
	"fmt"
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
	ID      string `json:"id"`
	UserID  string `json:"userId"`
	Name    string `json:"name"`
	IsAdmin bool   `json:"isAdmin"`

	// IsModerator records verified platform-admin status when the message was sent.
	IsModerator bool   `json:"isModerator"`
	IsHost      bool   `json:"isHost"`
	Kind        string `json:"kind"`
	Activity    bool   `json:"activity"`
	Text        string `json:"text"`
	CreatedAt   int64  `json:"createdAt"`
}

type CreateMessageRequest struct {
	Text string `json:"text" minLength:"1" maxLength:"500"`
}

func (r CreateMessageRequest) Validate() bool {
	length := utf8.RuneCountInString(strings.TrimSpace(r.Text))

	return length > 0 && length <= MessageMaxLength
}

type MessageAuthorFetcher interface {
	RoomV2Fetcher
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
	RoomType  RoomType  `json:"roomType,omitempty"`
	RoomID    string    `json:"roomId"`
	Window    string    `json:"window"`
	Timestamp time.Time `json:"timestamp"`
	Messages  int       `json:"messages"`
}

type AdminMessageUsage struct {
	RoomPoints  []MessageUsagePoint `json:"roomPoints"`
	RoomID      string              `json:"roomId"`
	Total       int                 `json:"total"`
	Points      []MessageUsagePoint `json:"points"`
	GeneratedAt time.Time           `json:"generatedAt"`
}

// MarshalJSON preserves empty collections in the public JSON contract.
func (v AdminMessageUsage) MarshalJSON() ([]byte, error) {
	type payload AdminMessageUsage
	result := payload(v)

	if result.RoomPoints == nil {
		result.RoomPoints = make([]MessageUsagePoint, 0)
	}

	if result.Points == nil {
		result.Points = make([]MessageUsagePoint, 0)
	}

	body, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error marshaling AdminMessageUsage: %w", err)
	}

	return body, nil
}

type AdminMessageUsageLister interface {
	ListAdminMessageUsage(ctx context.Context, roomID string) (*AdminMessageUsage, error)
}
