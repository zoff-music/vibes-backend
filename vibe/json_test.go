package vibe

import (
	"encoding/json/v2"
	"testing"
)

func TestJSONMarshalOmitsZeroRequestFields(t *testing.T) {
	tests := []struct {
		name     string
		request  RoomActionRequest
		expected string
	}{
		{
			name:     "omits a zero playback position",
			request:  RoomActionRequest{Action: RoomActionPlay},
			expected: `{"action":"play"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("expected request to marshal: %v", err)
			}
			if string(body) != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, body)
			}
		})
	}
}

func TestJSONUnmarshalRejectsDuplicateRequestFields(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "rejects duplicate actions",
			body: `{"action":"play","action":"pause"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var request RoomActionRequest
			err := json.Unmarshal([]byte(tt.body), &request)
			if err == nil {
				t.Fatal("expected duplicate request fields to fail")
			}
		})
	}
}
