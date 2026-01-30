package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/zoff-music/vibes/internalerror"
	"github.com/zoff-music/vibes/server/internal/helper"
	"github.com/zoff-music/vibes/vibe"
)

// Authorize redirects the user to the provider for authentication
func Authorize(db vibe.PendingOAuthStateSaver, oa vibe.OAuthAuthorizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		state := generateRandomString(32)

		// Get user ID from session
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error user not authenticated"),
				http.StatusUnauthorized,
				true,
			)
			return
		}

		// Store state in database
		err := db.SavePendingOAuthState(ctx, session.UserID, state)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error saving pending oauth state: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		fmt.Printf("DEBUG: Authorize - UserID: %s, State: %s\n", session.UserID, state)

		redirectURL := oa.GetOAuthURL(state)
		http.Redirect(
			w,
			r,
			redirectURL,
			http.StatusTemporaryRedirect,
		)
	}
}

// OAuthCallback handles GET /api/v1/callbacks/{provider}
func OAuthCallback(db vibe.CodeValidatorUpserter, oa vibe.OAuthExchanger, providerName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := r.URL.Query()
		code := query.Get("code")
		state := query.Get("state")

		// Validate state from database (stateless check to handle cross-domain cookies)
		userID, err := db.ValidateAndDeletePendingOAuthState(ctx, state)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error validating state: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if userID == "" {
			handleError(
				w,
				fmt.Errorf("invalid state parameter"),
				http.StatusBadRequest,
				true,
			)
			return
		}

		// Exchange code for tokens
		tokenResp, err := oa.ExchangeCode(ctx, code)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error exchanging code for token: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		// Calculate expiration
		expiresAt := time.Now().UTC().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		refreshExpiresAt := time.Now().UTC().Add(30 * 24 * time.Hour) // Default 30 days specific for refresh token if not provided.

		// Store initial auth token info
		authExpiresAt := time.Now().UTC().Add(24 * time.Hour)
		err = db.UpsertAuthToken(ctx, userID, providerName, code, state, authExpiresAt)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error storing auth token: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		// Store access tokens
		err = db.UpsertAccessToken(ctx, userID, providerName, tokenResp.AccessToken, tokenResp.RefreshToken, expiresAt, refreshExpiresAt)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error storing access token: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		// Redirect to frontend callback page
		http.Redirect(
			w,
			r,
			fmt.Sprintf("/callback?status=success&provider=%s", providerName),
			http.StatusTemporaryRedirect,
		)
	}
}

// GetToken handles GET /api/v1/authorizations/{provider}/token
func GetToken(db vibe.AccessTokenUpserterGetter, oa vibe.TokenRefresher, providerName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error user not authenticated"),
				http.StatusUnauthorized,
				true,
			)
			return
		}

		// Get current access token
		token, err := db.GetAccessToken(ctx, session.UserID, providerName)
		if err != nil {
			var notFoundErr internalerror.ErrAccessTokenNotFound
			if errors.As(err, &notFoundErr) {
				handleError(
					w,
					fmt.Errorf("error getting access token: %w", err),
					http.StatusPreconditionFailed,
					false,
				)
				return
			}

			// If not found or error, return 403
			handleError(
				w,
				fmt.Errorf("error getting access token: %w", err),
				http.StatusForbidden,
				false,
			)
			return
		}

		// Check if access token is valid
		if token.ExpiresAt.After(time.Now().UTC().Add(1 * time.Minute)) {
			resp := vibe.ProviderTokenResponse{
				AccessToken: token.AccessToken,
				ExpiresAt:   token.ExpiresAt,
			}

			body, err := json.Marshal(resp)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error marshalling response: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
			return
		}

		// Check Refresh Token expiration if we track it (optional check, provider will fail anyway)
		if token.RefreshToken == "" || !token.RefreshExpiresAt.IsZero() && token.RefreshExpiresAt.Before(time.Now().UTC()) {
			handleError(
				w,
				fmt.Errorf("refresh token expired"),
				http.StatusPreconditionFailed,
				true,
			)
			return
		}

		// Refresh token
		tokenResp, err := oa.RefreshToken(ctx, token.RefreshToken)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error refreshing token: %w", err),
				http.StatusForbidden,
				true,
			)
			return
		}

		// Update DB
		newExpiresAt := time.Now().UTC().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		newRefreshToken := tokenResp.RefreshToken
		if newRefreshToken == "" {
			newRefreshToken = token.RefreshToken // Keep old refresh token if not rotated
		}
		newRefreshExpiresAt := time.Now().UTC().Add(30 * 24 * time.Hour) // Reset refresh expiration

		err = db.UpsertAccessToken(ctx, token.UserID, providerName, tokenResp.AccessToken, newRefreshToken, newExpiresAt, newRefreshExpiresAt)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error updating access token: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		resp := vibe.ProviderTokenResponse{
			AccessToken: tokenResp.AccessToken,
			ExpiresAt:   newExpiresAt,
		}

		body, err := json.Marshal(resp)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshalling response: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

func generateRandomString(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
