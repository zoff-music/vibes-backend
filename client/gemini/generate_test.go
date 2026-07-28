package gemini

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/vibe"
)

type generatePlaylistTest struct {
	name          string
	enabled       bool
	playlist      vibe.GeneratedPlaylist
	expectedError string
}

func TestGeneratePlaylist(t *testing.T) {
	tests := []generatePlaylistTest{
		{
			name:    "generates playlist from system instruction",
			enabled: true,
			playlist: vibe.GeneratedPlaylist{
				{
					Title:     "Midnight City",
					Artist:    "M83",
					YouTubeID: "dX3k_QDnzHE",
				},
			},
		},
		{
			name:          "rejects disabled client",
			expectedError: "error validating gemini client in GeneratePlaylist: client is not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedRequest chatCompletionRequest
			var receivedAuthorization string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedAuthorization = r.Header.Get("Authorization")
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("expected request body, got error %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				err = json.Unmarshal(body, &receivedRequest)
				if err != nil {
					t.Errorf("expected JSON request, got error %v", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				playlistBody, err := json.Marshal(tt.playlist)
				if err != nil {
					t.Errorf("expected playlist marshal to succeed, got error %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				response := chatCompletionResponse{
					Choices: []chatCompletionChoice{
						{
							Message: chatCompletionMessage{
								Content: string(playlistBody),
							},
						},
					},
				}
				err = json.NewEncoder(w).Encode(response)
				if err != nil {
					t.Errorf("expected response encode to succeed, got error %v", err)
				}
			}))
			defer server.Close()

			geminiClient := Client{
				Enabled:           tt.enabled,
				Endpoint:          server.URL,
				Model:             "gemini-3.6-flash",
				apiKey:            "gemini-key",
				trackCount:        30,
				systemInstruction: vibe.GeneratedPlaylistSystemInstruction(30),
				HTTPClient: client.HTTPClient{
					Client: server.Client(),
				},
			}

			playlist, err := geminiClient.GeneratePlaylist(context.Background(), "songs for a night drive")
			if tt.expectedError != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.expectedError)
				}
				if err.Error() != tt.expectedError {
					t.Fatalf("expected error %q, got %q", tt.expectedError, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if len(*playlist) != 1 {
				t.Fatalf("expected one generated track, got %d", len(*playlist))
			}
			if (*playlist)[0].Title != "Midnight City" {
				t.Fatalf("unexpected generated track title %q", (*playlist)[0].Title)
			}
			if receivedAuthorization != "Bearer gemini-key" {
				t.Fatalf("unexpected authorization header %q", receivedAuthorization)
			}
			if receivedRequest.Model != "gemini-3.6-flash" {
				t.Fatalf("unexpected model %q", receivedRequest.Model)
			}
			if receivedRequest.ReasoningEffort != "low" {
				t.Fatalf("unexpected reasoning effort %q", receivedRequest.ReasoningEffort)
			}
			if len(receivedRequest.Messages) != 2 {
				t.Fatalf("expected two messages, got %d", len(receivedRequest.Messages))
			}
			if receivedRequest.Messages[0].Role != "system" {
				t.Fatalf("unexpected first message role %q", receivedRequest.Messages[0].Role)
			}
			if receivedRequest.Messages[0].Content != vibe.GeneratedPlaylistSystemInstruction(30) {
				t.Fatal("expected generated playlist system instruction")
			}
			if receivedRequest.Messages[1].Content != "songs for a night drive" {
				t.Fatalf("unexpected user prompt %q", receivedRequest.Messages[1].Content)
			}
		})
	}
}
