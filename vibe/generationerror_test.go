package vibe

import (
	"testing"
)

func TestPublicGenerationError(t *testing.T) {
	tests := []struct {
		name   string
		reason string
		want   string
	}{
		{
			name:   "no failure",
			reason: "",
			want:   "",
		},
		{
			name:   "generic failure",
			reason: RoomGenerationFailure,
			want:   RoomGenerationFailure,
		},
		{
			name:   "quota failure",
			reason: RoomGenerationYouTubeQuotaFailure,
			want:   RoomGenerationYouTubeQuotaFailure,
		},
		{
			name:   "persisted SQL detail",
			reason: "sql: no rows in result set",
			want:   RoomGenerationFailure,
		},
		{
			name:   "persisted provider detail",
			reason: "upstream request failed: token=secret",
			want:   RoomGenerationFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PublicGenerationError(tt.reason)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
