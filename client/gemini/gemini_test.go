package gemini

import (
	"context"
	"testing"

	"github.com/zoff-music/vibes-backend/config"
)

type initTest struct {
	name             string
	config           config.Config
	expectedEnabled  bool
	expectedEndpoint string
	expectedModel    string
	expectedHTTP     bool
	expectedError    string
}

func TestInit(t *testing.T) {
	tests := []initTest{
		{
			name: "initializes enabled client",
			config: config.Config{
				GeminiAPIKey:   "gemini-key",
				GeminiEndpoint: "https://generativelanguage.googleapis.com/v1beta/openai/",
				AIModel:        "GEMINI:gemini-3.6-flash",
			},
			expectedEnabled:  true,
			expectedEndpoint: "https://generativelanguage.googleapis.com/v1beta/openai",
			expectedModel:    "gemini-3.6-flash",
			expectedHTTP:     true,
		},
		{
			name: "initializes disabled client without api key",
			config: config.Config{
				GeminiEndpoint: "https://generativelanguage.googleapis.com/v1beta/openai",
				AIModel:        "GEMINI:gemini-3.6-flash",
			},
			expectedEnabled:  false,
			expectedEndpoint: "https://generativelanguage.googleapis.com/v1beta/openai",
			expectedModel:    "gemini-3.6-flash",
			expectedHTTP:     true,
		},
		{
			name: "ignores configuration for another provider",
			config: config.Config{
				GeminiAPIKey: "gemini-key",
				AIModel:      "GROK:grok-4.5",
			},
			expectedEnabled: false,
		},
		{
			name: "rejects empty endpoint",
			config: config.Config{
				AIModel: "GEMINI:gemini-3.6-flash",
			},
			expectedError: "error gemini endpoint is required",
		},
		{
			name: "rejects empty model",
			config: config.Config{
				GeminiEndpoint: "https://generativelanguage.googleapis.com/v1beta/openai",
				AIModel:        "GEMINI:",
			},
			expectedError: "error parsing configured AI model in Init: error parsing AI model \"GEMINI:\": model is required",
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
			if geminiClient.Endpoint != tt.expectedEndpoint {
				t.Fatalf("expected endpoint %q, got %q", tt.expectedEndpoint, geminiClient.Endpoint)
			}
			if geminiClient.Model != tt.expectedModel {
				t.Fatalf("expected model %q, got %q", tt.expectedModel, geminiClient.Model)
			}
			httpInitialized := geminiClient.HTTPClient.Client != nil
			if httpInitialized != tt.expectedHTTP {
				t.Fatalf("expected HTTP initialized %t, got %t", tt.expectedHTTP, httpInitialized)
			}
		})
	}
}
