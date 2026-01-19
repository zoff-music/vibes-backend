package event

import (
	"context"
	"log"
	"time"

	"github.com/zoff-music/vibes/vibe"
)

// CleanupHandler cleans up inactive participants
type CleanupHandler struct {
	DB vibe.ParticipantStorage
}

// Handle deletes participants who haven't been seen in 1 hour
func (h *CleanupHandler) Handle(ctx context.Context, data []byte) error {
	deleted, err := h.DB.DeleteInactiveParticipants(ctx, 1*time.Hour)
	if err != nil {
		return err
	}

	if deleted > 0 {
		log.Printf("Cleaned up %d inactive participants", deleted)
	}

	return nil
}
