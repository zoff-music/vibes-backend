package youtube

import "testing"

func TestVideoItemIsLiveVideo(t *testing.T) {
	tests := []struct {
		name     string
		item     videoItem
		expected bool
	}{
		{
			name: "live broadcast",
			item: videoItem{
				Snippet: videoSnippet{LiveBroadcastContent: youtubeLiveBroadcastContent},
			},
			expected: true,
		},
		{
			name: "upcoming broadcast",
			item: videoItem{
				Snippet: videoSnippet{LiveBroadcastContent: youtubeUpcomingBroadcastContent},
			},
			expected: true,
		},
		{
			name: "zero duration",
			item: videoItem{
				ContentDetails: videoContentDetails{Duration: youtubeZeroDuration},
			},
			expected: true,
		},
		{
			name: "recorded video",
			item: videoItem{
				Snippet:        videoSnippet{LiveBroadcastContent: "none"},
				ContentDetails: videoContentDetails{Duration: "PT3M"},
			},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.item.isLiveVideo()
			if actual != test.expected {
				t.Fatalf("expected %t, got %t", test.expected, actual)
			}
		})
	}
}
