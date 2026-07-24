package grok

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type playlistJSONSchema struct {
	Type                 string             `json:"type"`
	Properties           playlistProperties `json:"properties"`
	Required             []string           `json:"required"`
	AdditionalProperties bool               `json:"additionalProperties"`
}

type playlistProperties struct {
	Tracks playlistTracksSchema `json:"tracks"`
}

type playlistTracksSchema struct {
	Type     string          `json:"type"`
	Items    trackJSONSchema `json:"items"`
	MinItems int             `json:"minItems"`
	MaxItems int             `json:"maxItems"`
}

type trackJSONSchema struct {
	Type                 string          `json:"type"`
	Properties           trackProperties `json:"properties"`
	Required             []string        `json:"required"`
	AdditionalProperties bool            `json:"additionalProperties"`
}

type trackProperties struct {
	Artist         stringJSONSchema `json:"artist"`
	Title          stringJSONSchema `json:"title"`
	YouTubeVideoID stringJSONSchema `json:"youtubeVideoId"`
}

type stringJSONSchema struct {
	Type      string `json:"type"`
	MinLength int    `json:"minLength"`
	MaxLength int    `json:"maxLength,omitempty"`
	Pattern   string `json:"pattern,omitempty"`
}

type responseJSONSchema struct {
	Name   string             `json:"name"`
	Schema playlistJSONSchema `json:"schema"`
	Strict bool               `json:"strict"`
}

type responseFormat struct {
	Type       string             `json:"type"`
	JSONSchema responseJSONSchema `json:"json_schema"`
}

type chatCompletionRequest struct {
	Model           string         `json:"model"`
	Messages        []chatMessage  `json:"messages"`
	ResponseFormat  responseFormat `json:"response_format"`
	ReasoningEffort string         `json:"reasoning_effort"`
	Temperature     float64        `json:"temperature"`
	MaxTokens       int            `json:"max_tokens"`
}

type chatCompletionMessage struct {
	Content string `json:"content"`
	Refusal string `json:"refusal"`
}

type chatCompletionChoice struct {
	Message chatCompletionMessage `json:"message"`
}

type chatCompletionResponse struct {
	Choices []chatCompletionChoice `json:"choices"`
}

func (c *Client) GeneratePlaylist(ctx context.Context, prompt string) (*vibe.GeneratedPlaylist, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GeneratePlaylist")
	defer span.End()

	if !c.Enabled {
		return nil, fmt.Errorf("error grok client is not configured")
	}

	request := chatCompletionRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{
				Role:    "system",
				Content: vibe.GeneratedPlaylistSystemInstruction,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		ResponseFormat:  playlistResponseFormat(),
		ReasoningEffort: "none",
		Temperature:     0.7,
		MaxTokens:       3000,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling grok playlist request: %w", err)
	}

	responseBody, err := c.HTTPClient.RequestBytes(ctx, client.HTTPRequestData{
		Method: http.MethodPost,
		URL:    c.Endpoint + "/chat/completions",
		Headers: map[string]string{
			"Authorization": "Bearer " + c.apiKey,
			"Content-Type":  "application/json",
		},
		Body: body,
	})
	if err != nil {
		return nil, fmt.Errorf("error requesting grok playlist: %w", err)
	}

	var response chatCompletionResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling grok playlist response: %w", err)
	}
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("error grok playlist response has no choices")
	}
	if response.Choices[0].Message.Refusal != "" {
		return nil, fmt.Errorf("error grok refused playlist request")
	}

	var playlist vibe.GeneratedPlaylist
	err = json.Unmarshal([]byte(response.Choices[0].Message.Content), &playlist)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling generated playlist: %w", err)
	}

	err = validatePlaylist(&playlist)
	if err != nil {
		return nil, fmt.Errorf("error validating generated playlist: %w", err)
	}

	return &playlist, nil
}

func playlistResponseFormat() responseFormat {
	stringSchema := stringJSONSchema{
		Type:      "string",
		MinLength: 1,
	}

	return responseFormat{
		Type: "json_schema",
		JSONSchema: responseJSONSchema{
			Name:   "generated_playlist",
			Strict: true,
			Schema: playlistJSONSchema{
				Type: "object",
				Properties: playlistProperties{
					Tracks: playlistTracksSchema{
						Type:     "array",
						MinItems: vibe.GeneratedPlaylistTrackCount,
						MaxItems: vibe.GeneratedPlaylistTrackCount,
						Items: trackJSONSchema{
							Type: "object",
							Properties: trackProperties{
								Artist: stringSchema,
								Title:  stringSchema,
								YouTubeVideoID: stringJSONSchema{
									Type:      "string",
									MinLength: 11,
									MaxLength: 11,
									Pattern:   "^[A-Za-z0-9_-]{11}$",
								},
							},
							Required:             []string{"artist", "title", "youtubeVideoId"},
							AdditionalProperties: false,
						},
					},
				},
				Required:             []string{"tracks"},
				AdditionalProperties: false,
			},
		},
	}
}

func validatePlaylist(playlist *vibe.GeneratedPlaylist) error {
	if len(playlist.Tracks) != vibe.GeneratedPlaylistTrackCount {
		return fmt.Errorf("error expected %d tracks, got %d", vibe.GeneratedPlaylistTrackCount, len(playlist.Tracks))
	}

	seen := make(map[string]bool, vibe.GeneratedPlaylistTrackCount)
	seenVideoIDs := make(map[string]bool, vibe.GeneratedPlaylistTrackCount)
	for index := range playlist.Tracks {
		playlist.Tracks[index].Artist = strings.TrimSpace(playlist.Tracks[index].Artist)
		playlist.Tracks[index].Title = strings.TrimSpace(playlist.Tracks[index].Title)
		if playlist.Tracks[index].Artist == "" || playlist.Tracks[index].Title == "" {
			return fmt.Errorf("error track %d has an empty artist or title", index+1)
		}

		key := strings.ToLower(playlist.Tracks[index].Artist + "\x00" + playlist.Tracks[index].Title)
		if seen[key] {
			return fmt.Errorf("error track %d is a duplicate", index+1)
		}
		seen[key] = true

		videoID := playlist.Tracks[index].YouTubeVideoID
		if len(videoID) != youtubeVideoIDLength {
			return fmt.Errorf("error track %d has an invalid youtube video ID", index+1)
		}
		if seenVideoIDs[videoID] {
			return fmt.Errorf("error track %d has a duplicate youtube video ID", index+1)
		}
		seenVideoIDs[videoID] = true
	}

	return nil
}

const youtubeVideoIDLength = 11
