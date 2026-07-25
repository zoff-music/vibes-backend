package sse

import (
	"net/http/httptest"
	"testing"

	"github.com/zoff-music/vibes-backend/vibe"
)

func TestClientWrite(t *testing.T) {
	tests := []struct {
		name     string
		event    vibe.RoomEvent
		expected string
	}{
		{
			name: "writes an event and its JSON payload",
			event: vibe.RoomEvent{
				Type:    "updated",
				Payload: []byte(`{"value":1}`),
			},
			expected: "event: updated\ndata: {\"value\":1}\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client Client
			recorder := httptest.NewRecorder()

			err := client.Write(recorder, recorder, tt.event)
			if err != nil {
				t.Fatalf("Write() error = %v", err)
			}
			if recorder.Body.String() != tt.expected {
				t.Fatalf("Write() body = %q, want %q", recorder.Body.String(), tt.expected)
			}
			if !recorder.Flushed {
				t.Fatal("Write() did not flush")
			}
		})
	}
}

func TestClientHeartbeat(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{
			name:     "writes a heartbeat comment",
			expected: ": heartbeat\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client Client
			recorder := httptest.NewRecorder()

			client.Heartbeat(recorder, recorder)

			if recorder.Body.String() != tt.expected {
				t.Fatalf("Heartbeat() body = %q, want %q", recorder.Body.String(), tt.expected)
			}
			if !recorder.Flushed {
				t.Fatal("Heartbeat() did not flush")
			}
		})
	}
}
