package gemini

import (
	"context"
	"testing"

	"github.com/zoff-music/vibes-backend/config"
)

type initTest struct {
	name            string
	config          config.Config
	expectedEnabled bool
	expectedError   string
}

func TestInit(t *testing.T) {
	tests := []initTest{
		{
			name: "initializes enabled client",
			config: config.Config{
				GeminiAPIKey:   "gemini-key",
				GeminiEndpoint: "https://generativelanguage.googleapis.com/v1beta/openai/",
				AIModel:        "gemini-3.6-flash",
			},
			expectedEnabled: true,
		},
		{
			name: "initializes disabled client without api key",
			config: config.Config{
				GeminiEndpoint: "https://generativelanguage.googleapis.com/v1beta/openai",
				AIModel:        "gemini-3.6-flash",
			},
			expectedEnabled: false,
		},
		{
			name: "rejects empty endpoint",
			config: config.Config{
				AIModel: "gemini-3.6-flash",
			},
			expectedError: "error gemini endpoint is required",
		},
		{
			name: "rejects empty model",
			config: config.Config{
				GeminiEndpoint: "https://generativelanguage.googleapis.com/v1beta/openai",
			},
			expectedError: "error gemini model is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var geminiClient Client
			err := geminiClient.Init(context.Background(), &tt.config)
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
			if geminiClient.Enabled != tt.expectedEnabled {
				t.Fatalf("expected enabled %t, got %t", tt.expectedEnabled, geminiClient.Enabled)
			}
			if geminiClient.Endpoint != "https://generativelanguage.googleapis.com/v1beta/openai" {
				t.Fatalf("unexpected endpoint %q", geminiClient.Endpoint)
			}
			if geminiClient.Model != "gemini-3.6-flash" {
				t.Fatalf("unexpected model %q", geminiClient.Model)
			}
			if geminiClient.HTTPClient.Client == nil {
				t.Fatal("expected HTTP client to be initialized")
			}
		})
	}
}
