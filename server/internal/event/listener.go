package event

import (
	"context"
	"fmt"

	"github.com/zoff-music/vibes-backend/vibe"
)

type TrackListenerUsage struct {
	DB vibe.ListenerUsageCreator
}

func (h *TrackListenerUsage) Handle(ctx context.Context, _ []byte) error {
	err := h.DB.CreateListenerUsage(ctx)
	if err != nil {
		return fmt.Errorf(
			"error creating listener usage in TrackListenerUsage.Handle: %w",
			err,
		)
	}

	return nil
}
