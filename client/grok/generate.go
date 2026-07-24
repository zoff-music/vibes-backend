package grok

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model           string         `json:"model"`
	Messages        []chatMessage  `json:"messages"`
	ReasoningEffort string         `json:"reasoning_effort"`
	Temperature     float64        `json:"temperature"`
	MaxTokens       int            `json:"max_tokens"`
	ResponseFormat  responseFormat `json:"response_format"`
}

type responseFormat struct {
	Type       string             `json:"type"`
	JSONSchema responseJSONSchema `json:"json_schema"`
}

type responseJSONSchema struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema playlistSchema `json:"schema"`
}

type playlistSchema struct {
	Type     string             `json:"type"`
	MinItems int                `json:"minItems"`
	MaxItems int                `json:"maxItems"`
	Items    playlistItemSchema `json:"items"`
}

type playlistItemSchema struct {
	Type                 string                 `json:"type"`
	Properties           playlistItemProperties `json:"properties"`
	Required             []string               `json:"required"`
	AdditionalProperties bool                   `json:"additionalProperties"`
}

type playlistItemProperties struct {
	Title     stringProperty `json:"title"`
	Artist    stringProperty `json:"artist"`
	YouTubeID stringProperty `json:"youtubeId"`
}

type stringProperty struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Pattern     string `json:"pattern,omitempty"`
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
		ReasoningEffort: "low",
		Temperature:     0.3,
		MaxTokens:       6000,
		ResponseFormat: responseFormat{
			Type: "json_schema",
			JSONSchema: responseJSONSchema{
				Name:   "generated_playlist",
				Strict: true,
				Schema: playlistSchema{
					Type:     "array",
					MinItems: 1,
					MaxItems: vibe.GeneratedPlaylistTrackCount,
					Items: playlistItemSchema{
						Type: "object",
						Properties: playlistItemProperties{
							Title: stringProperty{
								Type:        "string",
								Description: "Canonical song title",
							},
							Artist: stringProperty{
								Type:        "string",
								Description: "Canonical artist name",
							},
							YouTubeID: stringProperty{
								Type:        "string",
								Description: "Exact YouTube video ID when known with high confidence",
								Pattern:     "^[A-Za-z0-9_-]{11}$",
							},
						},
						Required: []string{
							"title",
							"artist",
						},
						AdditionalProperties: false,
					},
				},
			},
		},
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
	if len(playlist) == 0 {
		return nil, fmt.Errorf(
			"error generated playlist has no tracks",
		)
	}
	if len(playlist) > vibe.GeneratedPlaylistTrackCount {
		playlist = playlist[:vibe.GeneratedPlaylistTrackCount]
	}

	return &playlist, nil
}
