package handler

import (
	"context"
	"fmt"
	"github.com/zoff-music/vibes-backend/monitoring/opentracing"
	"log"
	"time"

	"github.com/zoff-music/vibes-backend/vibe"
)

// CleanupInactiveParticipants cleans up inactive participants
type CleanupInactiveParticipants struct {
	DB vibe.ParticipantStorage
}

// Handle deletes participants who haven't been seen in 1 hour
func (h *CleanupInactiveParticipants) Handle(ctx context.Context, _ []byte) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Handle")
	defer span.Finish()

	deleted, err := h.DB.DeleteInactiveParticipants(ctx, 1*time.Hour)
	if err != nil {
		return err
	}

	if deleted > 0 {
		log.Printf("Cleaned up %d inactive participants", deleted)
	}

	return nil
}

// CleanupExpiredTokens cleans up expired external auth records
type CleanupExpiredTokens struct {
	DB vibe.AuthTokenCleaner
}

// Handle deletes expired external auth tokens
func (h *CleanupExpiredTokens) Handle(ctx context.Context, _ []byte) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Handle")
	defer span.Finish()

	deletedAuth, err := h.DB.DeleteExpiredAuthTokens(ctx)
	if err != nil {
		return err
	}

	deletedAccess, err := h.DB.DeleteExpiredAccessTokens(ctx)
	if err != nil {
		return err
	}

	totalDeleted := deletedAuth + deletedAccess
	if totalDeleted > 0 {
		log.Printf("Cleaned up %d expired auth tokens and %d expired access tokens", deletedAuth, deletedAccess)
	}

	return nil
}

// RefreshSpotifyTokens refreshes expired Spotify access tokens
type RefreshSpotifyTokens struct {
	DB       vibe.ExpiredTokenClaimUpdater
	Provider vibe.TokenRefresher
}

// Handle refreshes the next expired Spotify token
func (h *RefreshSpotifyTokens) Handle(ctx context.Context, _ []byte) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Handle")
	defer span.Finish()

	token, err := h.DB.ClaimAndGetExpiredTokenForRefresh(ctx, "spotify")
	if err != nil {
		return fmt.Errorf("error claiming expired token for refresh in spotify handler: %w", err)
	}

	newToken, err := h.Provider.RefreshToken(ctx, token.RefreshToken)
	if err != nil {
		log.Printf("Failed to refresh Spotify token for user %s: %v", token.UserID, err)
		return nil // Don't return error to continue processing other tokens
	}

	expiresAt := time.Now().Add(time.Duration(newToken.ExpiresIn) * time.Second)
	// Spotify refresh tokens don't expire, keep existing refresh_expires_at
	refreshToken := newToken.RefreshToken
	if refreshToken == "" {
		refreshToken = token.RefreshToken // Keep existing if not provided
	}

	err = h.DB.UpsertAccessToken(ctx, token.UserID, "spotify", newToken.AccessToken, refreshToken, expiresAt, token.RefreshExpiresAt)
	if err != nil {
		return err
	}

	log.Printf("Refreshed Spotify token for user %s", token.UserID)
	return nil
}

// RefreshYouTubeTokens refreshes expired YouTube access tokens
type RefreshYouTubeTokens struct {
	DB       vibe.ExpiredTokenClaimUpdater
	Provider vibe.TokenRefresher
}

// Handle refreshes the next expired YouTube token
func (h *RefreshYouTubeTokens) Handle(ctx context.Context, _ []byte) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Handle")
	defer span.Finish()

	token, err := h.DB.ClaimAndGetExpiredTokenForRefresh(ctx, "youtube")
	if err != nil {
		return fmt.Errorf("error claiming expired token for refresh in youtube handler: %w", err)
	}

	newToken, err := h.Provider.RefreshToken(ctx, token.RefreshToken)
	if err != nil {
		log.Printf("Failed to refresh YouTube token for user %s: %v", token.UserID, err)
		return nil
	}

	expiresAt := time.Now().Add(time.Duration(newToken.ExpiresIn) * time.Second)
	refreshToken := newToken.RefreshToken
	if refreshToken == "" {
		refreshToken = token.RefreshToken
	}

	err = h.DB.UpsertAccessToken(ctx, token.UserID, "youtube", newToken.AccessToken, refreshToken, expiresAt, token.RefreshExpiresAt)
	if err != nil {
		return fmt.Errorf("error upserting access-token for refreh in youtube handler")
	}

	log.Printf("Refreshed YouTube token for user %s", token.UserID)
	return nil
}

// CleanupExpiredPendingOAuthStates cleans up expired pending OAuth states
type CleanupExpiredPendingOAuthStates struct {
	DB vibe.ExpiredPendingOAuthStateCleaner
}

// Handle deletes expired pending OAuth states
func (h *CleanupExpiredPendingOAuthStates) Handle(ctx context.Context, _ []byte) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Handle")
	defer span.Finish()

	deleted, err := h.DB.DeleteExpiredPendingOAuthStates(ctx)
	if err != nil {
		return err
	}

	if deleted > 0 {
		log.Printf("Cleaned up %d expired pending OAuth states", deleted)
	}

	return nil
}
