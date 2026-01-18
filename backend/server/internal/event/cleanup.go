package event

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/zoff-music/vibes/vibe"
)

// CleanupHandler cleans up inactive participants
type CleanupHandler struct {
	DB vibe.ParticipantStorage
}

// NewCleanupHandler creates a new CleanupHandler
func NewCleanupHandler(db vibe.ParticipantStorage) *CleanupHandler {
	return &CleanupHandler{DB: db}
}

// Handle deletes participants who haven't been seen in 1 hour
func (h *CleanupHandler) Handle(ctx context.Context, data []byte) error {
	deleted, err := h.DB.DeleteInactiveParticipants(ctx, 1*time.Hour)
	if err != nil {
		return err
	}

	if deleted > 0 {
		log.Infof("Cleaned up %d inactive participants", deleted)
	}

	return nil
}
