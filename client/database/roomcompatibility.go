package database

import (
	"context"
	"fmt"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) SearchPublicRooms(ctx context.Context, search vibe.PublicRoomSearch) (*vibe.PublicRoomResult, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "SearchPublicRooms")
	defer span.End()

	result, err := c.SearchPublicRoomsV3(ctx, search)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy public rooms: %w", err)
	}

	legacy := result.ToPublicRoomResult()
	return legacy, nil
}

func (r *publicRoomRow) toPublicRoomV3() (*vibe.PublicRoomV3, error) {
	return &vibe.PublicRoomV3{
		ID:                r.ID.String,
		Name:              r.Name.String,
		ListenerCount:     int(r.ListenerCount.Int64),
		PlaylistItemCount: int(r.PlaylistItemCount.Int64),
	}, nil
}

// GetRoom preserves the legacy room contract at the database boundary.
func (c *Client) GetRoom(ctx context.Context, roomID string, userID string) (*vibe.Room, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetRoom")
	defer span.End()

	result, err := c.GetRoomV2(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy room in GetRoom: %w", err)
	}

	legacy := result.ToRoom()

	return legacy, nil
}

// GetRoomByName preserves the legacy room contract at the database boundary.
func (c *Client) GetRoomByName(ctx context.Context, roomID string, userID string) (*vibe.Room, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetRoomByName")
	defer span.End()

	result, err := c.GetRoomByNameV2(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy room in GetRoomByName: %w", err)
	}

	legacy := result.ToRoom()

	return legacy, nil
}

// CreateRoom preserves the legacy room contract at the database boundary.
func (c *Client) CreateRoom(ctx context.Context, room *vibe.Room, reservationToken string) (*vibe.Room, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreateRoom")
	defer span.End()

	canonical := room.ToRoomV2()

	result, err := c.CreateRoomV2(ctx, canonical, reservationToken)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy room in CreateRoom: %w", err)
	}

	legacy := result.ToRoom()

	return legacy, nil
}

// UpdateRoom preserves the legacy room contract at the database boundary.
func (c *Client) UpdateRoom(ctx context.Context, room *vibe.Room) (*vibe.Room, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpdateRoom")
	defer span.End()

	canonical := room.ToRoomV2()

	result, err := c.UpdateRoomV2(ctx, canonical)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy room in UpdateRoom: %w", err)
	}

	legacy := result.ToRoom()

	return legacy, nil
}
