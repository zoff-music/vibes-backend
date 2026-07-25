package grok

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
			name: "initializes selected provider",
			config: config.Config{
				GrokAPIKey:   "grok-key",
				GrokEndpoint: "https://api.x.ai/v1/",
				AIModel:      "GROK:grok-4.5",
			},
			expectedEnabled:  true,
			expectedEndpoint: "https://api.x.ai/v1",
			expectedModel:    "grok-4.5",
			expectedHTTP:     true,
		},
		{
			name: "ignores configuration for another provider",
			config: config.Config{
				GrokAPIKey: "grok-key",
				AIModel:    "GEMINI:gemini-3.6-flash",
			},
			expectedEnabled: false,
		},
		{
			name: "rejects selected provider without API key",
			config: config.Config{
				GrokEndpoint: "https://api.x.ai/v1",
				AIModel:      "GROK:grok-4.5",
			},
			expectedError: "error grok API key is required",
		},
		{
			name: "rejects empty endpoint for selected provider",
			config: config.Config{
				GrokAPIKey: "grok-key",
				AIModel:    "GROK:grok-4.5",
			},
			expectedError: "error grok endpoint is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var grokClient Client
			err := grokClient.Init(context.Background(), &tt.config)
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
			if grokClient.Enabled != tt.expectedEnabled {
				t.Fatalf("expected enabled %t, got %t", tt.expectedEnabled, grokClient.Enabled)
			}
			if grokClient.Endpoint != tt.expectedEndpoint {
				t.Fatalf("expected endpoint %q, got %q", tt.expectedEndpoint, grokClient.Endpoint)
			}
			if grokClient.Model != tt.expectedModel {
				t.Fatalf("expected model %q, got %q", tt.expectedModel, grokClient.Model)
			}
			httpInitialized := grokClient.HTTPClient.Client != nil
			if httpInitialized != tt.expectedHTTP {
				t.Fatalf("expected HTTP initialized %t, got %t", tt.expectedHTTP, httpInitialized)
			}
		})
	}
}
