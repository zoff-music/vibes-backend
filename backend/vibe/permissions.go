package vibe

// PermissionProvider defines the data access requirements for authentication/permissions
type PermissionProvider interface {
	RoomFetcher
	UserFetcher
}
