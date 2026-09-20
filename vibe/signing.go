package vibe

type AdminAuthPayload struct {
	UserID         string `json:"user_id"`
	AdminID        string `json:"admin_id"`
	SessionVersion int64  `json:"session_version"`
	IssuedAt       int64  `json:"issued_at"`
}

type CastTokenPayload struct {
	V      int    `json:"v"`
	Typ    string `json:"typ"`
	RoomID string `json:"roomId"`
	UserID string `json:"userId"`
	Iat    int64  `json:"iat"`
	Exp    int64  `json:"exp"`
}
