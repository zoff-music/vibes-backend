package database

import (
	"context"
	"fmt"

	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
)

func (c *Client) SearchAdminRooms(ctx context.Context, search vibe.AdminRoomSearch) (*vibe.AdminRoomResult, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "SearchAdminRooms")
	defer span.End()

	sortBy := vibe.AdminRoomSortV2(search.SortBy)
	if search.SortBy == vibe.AdminRoomSortSongs {
		sortBy = vibe.AdminRoomSortPlaylistItems
	}

	result, err := c.SearchAdminRoomsV2(ctx, vibe.AdminRoomSearchV2{Query: search.Query, SortBy: sortBy, Descending: search.Descending, From: search.From, To: search.To})
	if err != nil {
		return nil, fmt.Errorf("error searching legacy admin rooms: %w", err)
	}

	legacy := result.ToAdminRoomResult()
	return legacy, nil
}

func (c *Client) ListAdminRooms(ctx context.Context) ([]vibe.AdminRoomSummary, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "ListAdminRooms")
	defer span.End()

	rooms, err := c.ListAdminRoomsV2(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing legacy admin rooms: %w", err)
	}

	legacy := make([]vibe.AdminRoomSummary, len(rooms))
	for index, room := range rooms {
		legacy[index] = *room.ToAdminRoomSummary()
	}

	return legacy, nil
}
