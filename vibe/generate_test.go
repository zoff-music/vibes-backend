package vibe

import "testing"

type parseAIModelTest struct {
	name             string
	value            string
	expectedProvider string
	expectedModel    string
	expectedError    string
}

func TestParseAIModel(t *testing.T) {
	tests := []parseAIModelTest{
		{
			name:             "parses Grok model",
			value:            "GROK:grok-4.5",
			expectedProvider: AIProviderGrok,
			expectedModel:    "grok-4.5",
		},
		{
			name:             "parses Gemini model case insensitively",
			value:            "gemini:gemini-3.6-flash",
			expectedProvider: AIProviderGemini,
			expectedModel:    "gemini-3.6-flash",
		},
		{
			name:             "parses legacy unprefixed Grok model",
			value:            "grok-4.5",
			expectedProvider: AIProviderGrok,
			expectedModel:    "grok-4.5",
		},
		{
			name:             "parses legacy unprefixed Gemini model",
			value:            "gemini-3.6-flash",
			expectedProvider: AIProviderGemini,
			expectedModel:    "gemini-3.6-flash",
		},
		{
			name:          "rejects missing provider separator",
			value:         "unknown-model",
			expectedError: "error parsing AI model \"unknown-model\": expected PROVIDER:model",
		},
		{
			name:          "rejects unsupported provider",
			value:         "OPENAI:gpt-5",
			expectedError: "error parsing AI model \"OPENAI:gpt-5\": provider must be GROK or GEMINI",
		},
		{
			name:          "rejects empty model",
			value:         "GEMINI:",
			expectedError: "error parsing AI model \"GEMINI:\": model is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := ParseAIModel(tt.value)
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
			if model.Provider != tt.expectedProvider {
				t.Fatalf("expected provider %q, got %q", tt.expectedProvider, model.Provider)
			}
			if model.Name != tt.expectedModel {
				t.Fatalf("expected model %q, got %q", tt.expectedModel, model.Name)
			}
		})
	}
}
