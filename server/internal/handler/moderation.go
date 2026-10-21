package handler

import (
	"context"
	"fmt"

	"github.com/zoff-music/vibes-backend/vibe"
)

type ReviewRoomNames struct {
	DB vibe.RoomNameReviewClaimerUpdater
	AI vibe.RoomNameReviewer
}

func (h *ReviewRoomNames) Handle(ctx context.Context, _ []byte) error {
	review, err := h.DB.GetNextRoomNameReview(ctx)
	if err != nil {
		return fmt.Errorf("error claiming room name review: %w", err)
	}

	result, err := h.AI.ReviewRoomNames(ctx, review.Rooms)
	if err != nil {
		return fmt.Errorf("error reviewing room names: %w", err)
	}

	err = h.DB.CompleteRoomNameReview(ctx, &vibe.RoomNameReview{
		StartedAt: review.StartedAt,
		Rooms:     result.Rooms,
	})
	if err != nil {
		return fmt.Errorf("error completing room name review: %w", err)
	}

	return nil
}
