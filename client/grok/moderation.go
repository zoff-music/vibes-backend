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

func (c *Client) ReviewRoomNames(ctx context.Context, rooms []vibe.RoomNameDecision) ([]vibe.RoomNameDecision, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ReviewRoomNames")
	defer span.End()

	if !c.Enabled {
		return nil, fmt.Errorf("error reviewing room names: grok is not configured")
	}
	if len(rooms) == 0 {
		return nil, fmt.Errorf("error reviewing room names: no rooms supplied")
	}

	names, err := json.Marshal(rooms)
	if err != nil {
		return nil, fmt.Errorf("error encoding room names: %w", err)
	}

	body, err := json.Marshal(reviewRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{Role: "system", Content: vibe.RoomNameReviewInstruction},
			{Role: "user", Content: string(names)},
		},
		ReasoningEffort: "high",
		ResponseFormat: reviewResponseFormat{
			Type: "json_schema",
			JSONSchema: reviewJSONSchema{
				Name:   "room_name_review",
				Strict: true,
				Schema: reviewSchema{
					Type: "object",
					Properties: reviewProperties{
						FlaggedIDs: reviewIDsSchema{
							Type:  "array",
							Items: reviewIDSchema{Type: "string"},
						},
					},
					Required:             []string{"flaggedIds"},
					AdditionalProperties: false,
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error encoding review request: %w", err)
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
		return nil, fmt.Errorf("error requesting grok room review: %w", err)
	}

	var response chatCompletionResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error decoding grok room review: %w", err)
	}
	if len(response.Choices) != 1 || response.Choices[0].Message.Refusal != "" {
		return nil, fmt.Errorf("error reviewing room names: absent or refused completion")
	}

	content := strings.TrimSpace(response.Choices[0].Message.Content)
	lines := strings.SplitN(content, "\n", 2)
	if len(lines) == 2 && strings.HasSuffix(content, "```") {
		opening := strings.TrimSpace(lines[0])
		if opening == "```json" || opening == "```" {
			content = strings.TrimSpace(strings.TrimSuffix(lines[1], "```"))
		}
	}

	var verdict vibe.RoomNameVerdict
	err = json.Unmarshal([]byte(content), &verdict)
	if err != nil {
		return nil, fmt.Errorf("error decoding room name verdict: %w", err)
	}

	err = verdict.Validate(rooms)
	if err != nil {
		return nil, fmt.Errorf("error validating room name verdict: %w", err)
	}

	flagged := make(map[string]bool, len(verdict.FlaggedIDs))
	for _, id := range verdict.FlaggedIDs {
		flagged[id] = true
	}

	result := make([]vibe.RoomNameDecision, len(rooms))
	for i, room := range rooms {
		result[i] = vibe.RoomNameDecision{
			ID:      room.ID,
			Name:    room.Name,
			Flagged: flagged[room.ID],
		}
	}

	return result, nil
}

type reviewRequest struct {
	Model           string               `json:"model"`
	Messages        []chatMessage        `json:"messages"`
	ReasoningEffort string               `json:"reasoning_effort"`
	ResponseFormat  reviewResponseFormat `json:"response_format"`
}

type reviewResponseFormat struct {
	Type       string           `json:"type"`
	JSONSchema reviewJSONSchema `json:"json_schema"`
}

type reviewJSONSchema struct {
	Name   string       `json:"name"`
	Strict bool         `json:"strict"`
	Schema reviewSchema `json:"schema"`
}

type reviewSchema struct {
	Type                 string           `json:"type"`
	Properties           reviewProperties `json:"properties"`
	Required             []string         `json:"required"`
	AdditionalProperties bool             `json:"additionalProperties"`
}

type reviewProperties struct {
	FlaggedIDs reviewIDsSchema `json:"flaggedIds"`
}

type reviewIDsSchema struct {
	Type  string         `json:"type"`
	Items reviewIDSchema `json:"items"`
}

type reviewIDSchema struct {
	Type string `json:"type"`
}
