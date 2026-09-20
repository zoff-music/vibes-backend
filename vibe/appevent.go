package vibe

import "context"

type AppEventHandler interface {
	Handle(ctx context.Context, data []byte) error
}
