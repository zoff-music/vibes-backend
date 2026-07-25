package event

import (
	"testing"

	"github.com/zoff-music/vibes-backend/vibe"
)

func TestRoomSSEConnectionShouldSend(t *testing.T) {
	tests := []struct {
		name       string
		connection roomSSEConnection
		event      vibe.RoomEvent
		expected   bool
	}{
		{
			name: "sends events without an originating user",
			connection: roomSSEConnection{
				UserID: "user-1",
			},
			event: vibe.RoomEvent{
				UserID: "",
			},
			expected: true,
		},
		{
			name: "filters events from the connected user",
			connection: roomSSEConnection{
				UserID: "user-1",
			},
			event: vibe.RoomEvent{
				UserID: "user-1",
			},
			expected: false,
		},
		{
			name: "sends events from another user",
			connection: roomSSEConnection{
				UserID: "user-1",
			},
			event: vibe.RoomEvent{
				UserID: "user-2",
			},
			expected: true,
		},
		{
			name: "filters cast events using the owner identity",
			connection: roomSSEConnection{
				UserID:         "cast:room-1:user-1",
				CastOwnerID:    "user-1",
				IsCastReceiver: true,
			},
			event: vibe.RoomEvent{
				UserID: "user-1",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.connection.ShouldSend(tt.event)
			if actual != tt.expected {
				t.Fatalf("ShouldSend() = %t, want %t", actual, tt.expected)
			}
		})
	}
}
