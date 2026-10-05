package database

import (
	"context"
	"fmt"
	"github.com/zoff-music/vibes-backend/monitoring/tracing"
	"github.com/zoff-music/vibes-backend/vibe"
	"time"
)

func (c *Client) CreateRemoteControl(ctx context.Context, remoteID, ownerUserID, pairingTokenHash, pairingCodeHash, roomID string, pairingExpiresAt time.Time) (*vibe.RemoteControl, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "CreateRemoteControl")
	defer span.End()

	remote, err := c.CreateRemoteControlV2(ctx, remoteID, ownerUserID, pairingTokenHash, pairingCodeHash, roomID, pairingExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy remote control in CreateRemoteControl: %w", err)
	}

	legacy := remote.ToRemoteControl()
	return legacy, nil
}

func (c *Client) GetRemoteControlByOwner(ctx context.Context, ownerUserID string) (*vibe.RemoteControl, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetRemoteControlByOwner")
	defer span.End()

	remote, err := c.GetRemoteControlByOwnerV2(ctx, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy remote control in GetRemoteControlByOwner: %w", err)
	}

	legacy := remote.ToRemoteControl()
	return legacy, nil
}

func (c *Client) GetRemoteControl(ctx context.Context, remoteID string) (*vibe.RemoteControl, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "GetRemoteControl")
	defer span.End()

	remote, err := c.GetRemoteControlV2(ctx, remoteID)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy remote control in GetRemoteControl: %w", err)
	}

	legacy := remote.ToRemoteControl()
	return legacy, nil
}

func (c *Client) PairRemoteControl(ctx context.Context, remoteID, pairingTokenHash, pairingCodeHash, controllerTokenHash string) (*vibe.RemoteControl, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "PairRemoteControl")
	defer span.End()

	remote, err := c.PairRemoteControlV2(ctx, remoteID, pairingTokenHash, pairingCodeHash, controllerTokenHash)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy remote control in PairRemoteControl: %w", err)
	}

	legacy := remote.ToRemoteControl()
	return legacy, nil
}

func (c *Client) AuthenticateRemoteControl(ctx context.Context, remoteID, controllerTokenHash string) (*vibe.RemoteControl, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "AuthenticateRemoteControl")
	defer span.End()

	remote, err := c.AuthenticateRemoteControlV2(ctx, remoteID, controllerTokenHash)
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy remote control in AuthenticateRemoteControl: %w", err)
	}

	legacy := remote.ToRemoteControl()
	return legacy, nil
}

func (c *Client) UpdateOwnedRemoteControl(ctx context.Context, remoteID, ownerUserID string, request vibe.RemoteUpdateRequest) (*vibe.RemoteControl, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpdateOwnedRemoteControl")
	defer span.End()

	remote, err := c.UpdateOwnedRemoteControlV2(ctx, remoteID, ownerUserID, *request.ToRemoteUpdateRequestV2())
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy remote control in UpdateOwnedRemoteControl: %w", err)
	}

	legacy := remote.ToRemoteControl()
	return legacy, nil
}

func (c *Client) UpdatePairedRemoteControl(ctx context.Context, remoteID string, request vibe.RemoteUpdateRequest) (*vibe.RemoteControl, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "UpdatePairedRemoteControl")
	defer span.End()

	remote, err := c.UpdatePairedRemoteControlV2(ctx, remoteID, *request.ToRemoteUpdateRequestV2())
	if err != nil {
		return nil, fmt.Errorf("error mapping legacy remote control in UpdatePairedRemoteControl: %w", err)
	}

	legacy := remote.ToRemoteControl()
	return legacy, nil
}
