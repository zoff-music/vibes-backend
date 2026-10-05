package vibe

import "context"

// PermissionProvider defines the data access requirements for authentication/permissions
type PermissionProvider interface {
	RoomV2Fetcher
	GetUser(ctx context.Context, roomID, userID string) (*User, error)
}
