package vibe

import "context"

// PermissionProvider defines the data access requirements for authentication/permissions
type PermissionProvider interface {
	RoomFetcher
	GetUser(ctx context.Context, roomID, userID string) (*User, error)
}
