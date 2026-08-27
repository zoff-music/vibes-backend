package vibe

import "testing"

func TestIsLiveVideo(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		duration int
		expected bool
	}{
		{
			name:     "youtube zero duration",
			source:   SourceTypeYouTube,
			duration: 0,
			expected: true,
		},
		{
			name:     "youtube negative duration",
			source:   SourceTypeYouTube,
			duration: -1,
			expected: true,
		},
		{
			name:     "youtube recorded video",
			source:   SourceTypeYouTube,
			duration: 180,
			expected: false,
		},
		{
			name:     "soundcloud zero duration",
			source:   SourceTypeSoundCloud,
			duration: 0,
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := IsLiveVideo(test.source, test.duration)
			if actual != test.expected {
				t.Fatalf("expected %t, got %t", test.expected, actual)
			}
		})
	}
}
