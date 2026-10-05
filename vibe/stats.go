package vibe

// Stats contains public, service-wide usage statistics.
//
// Deprecated: Use StatsV2 for new code. Retained for legacy API compatibility.
type Stats struct {
	TotalListeners int `json:"totalListeners"`
	TotalSongs     int `json:"totalSongs"`
	TotalRooms     int `json:"totalRooms"`
}
