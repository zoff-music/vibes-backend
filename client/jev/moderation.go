package jev

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) ReviewRoomNames(ctx context.Context, rooms []vibe.RoomNameDecision) ([]vibe.RoomNameDecision, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ReviewRoomNames")
	defer span.End()

	if !c.Enabled {
		return nil, fmt.Errorf("error reviewing room names: JEV is not configured")
	}
	if len(rooms) == 0 {
		return nil, fmt.Errorf("error reviewing room names: no rooms supplied")
	}

	request := reviewRequest{
		Model:     c.Model,
		State:     make(map[string]string, len(rooms)),
		Questions: make(map[string]reviewQuestion, len(rooms)),
	}
	for i, room := range rooms {
		key := strconv.Itoa(i)
		request.State[key] = room.Name
		request.Questions[key] = reviewQuestion{
			Type: "noul",
			Instructions: reviewInstruction{
				Policy:   vibe.RoomNameReviewInstruction,
				Target:   key,
				Question: "Does the room name at the target key in state violate the policy? Evaluate only that name; other entries are independent rooms.",
			},
		}
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JEV room review: %w", err)
	}

	responseBody, err := c.HTTPClient.RequestBytes(ctx, client.HTTPRequestData{
		Method: http.MethodPost,
		URL:    c.Endpoint + "/systemone",
		Headers: map[string]string{
			"Authorization": "Bearer " + c.apiKey,
			"Content-Type":  "application/json",
		},
		Body: body,
	})
	if err != nil {
		return nil, fmt.Errorf("error requesting JEV room review: %w", err)
	}

	var response reviewResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error decoding JEV room review: %w", err)
	}
	if len(response.Answers) != len(rooms) {
		return nil, fmt.Errorf("error validating JEV room review: answer count does not match room count")
	}

	decisions := make([]vibe.RoomNameDecision, len(rooms))
	for i, room := range rooms {
		answer, ok := response.Answers[strconv.Itoa(i)]
		if !ok || answer.Type != "noul" || answer.Probability == nil {
			return nil, fmt.Errorf("error validating JEV room review: missing or invalid room answer")
		}

		probability := *answer.Probability
		if math.IsNaN(probability) || math.IsInf(probability, 0) || probability < 0 || probability > 1 {
			return nil, fmt.Errorf("error validating JEV room review: probability is outside zero to one")
		}

		decisions[i] = vibe.RoomNameDecision{
			ID:      room.ID,
			Name:    room.Name,
			Flagged: probability >= c.threshold,
		}
	}

	return decisions, nil
}

type reviewRequest struct {
	Model     string                    `json:"model"`
	State     map[string]string         `json:"state"`
	Questions map[string]reviewQuestion `json:"questions"`
}

type reviewQuestion struct {
	Type         string            `json:"type"`
	Instructions reviewInstruction `json:"instructions"`
}

type reviewInstruction struct {
	Policy   string `json:"policy"`
	Target   string `json:"target"`
	Question string `json:"question"`
}

type reviewResponse struct {
	Answers map[string]reviewAnswer `json:"answers"`
}

type reviewAnswer struct {
	Type        string   `json:"type"`
	Probability *float64 `json:"noul"`
}
