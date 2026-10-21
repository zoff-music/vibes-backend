package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) ReviewRoomNames(ctx context.Context, rooms []vibe.RoomNameDecision) ([]vibe.RoomNameDecision, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ReviewRoomNames")
	defer span.End()

	if !c.Enabled {
		return nil, fmt.Errorf("error reviewing room names: gemini is not configured")
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
		return nil, fmt.Errorf("error requesting gemini room review: %w", err)
	}

	var response chatCompletionResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error decoding gemini room review: %w", err)
	}
	if len(response.Choices) != 1 || response.Choices[0].Message.Refusal != "" {
		return nil, fmt.Errorf("error reviewing room names: absent or refused completion")
	}

	var verdict vibe.RoomNameVerdict
	err = json.Unmarshal([]byte(response.Choices[0].Message.Content), &verdict)
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
	Model           string        `json:"model"`
	Messages        []chatMessage `json:"messages"`
	ReasoningEffort string        `json:"reasoning_effort"`
}
