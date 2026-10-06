package handler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/zoff-music/vibes-backend/vibe"
)

// CleanupInactiveParticipants cleans up inactive participants
type CleanupInactiveParticipants struct {
	DB vibe.InactiveParticipantCleaner
}

// Handle cleans participants who haven't been seen in 1 hour.
func (h *CleanupInactiveParticipants) Handle(ctx context.Context, _ []byte) error {
	cleaned, err := h.DB.CleanInactiveParticipants(ctx, 1*time.Hour)
	if err != nil {
		return fmt.Errorf("error cleaning inactive participants in CleanupInactiveParticipants.Handle: %w", err)
	}

	if cleaned > 0 {
		log.Printf("Cleaned up %d inactive participants", cleaned)
	}

	return nil
}
