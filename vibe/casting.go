package vibe

// CreateCastingTokenRequest is the request payload for POST /api/v1/tokens/casting.
type CreateCastingTokenRequest struct {
	RoomID string `json:"roomId"`
}

// CastingTokenResponse is returned by POST /api/v1/tokens/casting.
type CastingTokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
	RoomID    string `json:"roomId"`
}
