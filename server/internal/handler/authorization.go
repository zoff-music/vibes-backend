package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// Authorize redirects the user to the provider for authentication
//
//	@Summary	Start provider authorization
//	@Tags		authorization
//	@Produce	json
//	@Success	307
//	@Failure	401	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/authorizations/spotify [get]
//	@Router		/api/v1/authorizations/soundcloud [get]
//	@Router		/api/v1/authorizations/youtube [get]
func Authorize(db vibe.PendingOAuthStateSaver, oa vibe.OAuthAuthorizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		state := generateRandomString(32)

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

		codeVerifier := generateRandomString(43) // length between 43 and 128 for PKCE

		err := db.SavePendingOAuthState(ctx, session.UserID, state, codeVerifier)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error saving pending oauth state: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		redirectURL := oa.GetOAuthURL(state, codeVerifier)
		http.Redirect(
			w,
			r,
			redirectURL,
			http.StatusTemporaryRedirect,
		)
	}
}

// OAuthCallback handles GET /api/v1/callbacks/{provider}
//
//	@Summary	Handle provider authorization callback
//	@Tags		authorization
//	@Param		code	query	string	true	"Authorization code"
//	@Param		state	query	string	true	"OAuth state"
//	@Success	307
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/callbacks/spotify [get]
//	@Router		/api/v1/callbacks/soundcloud [get]
//	@Router		/api/v1/callbacks/youtube [get]
func OAuthCallback(db vibe.CodeValidatorUpserter, oa vibe.OAuthExchanger, providerName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := r.URL.Query()
		code := query.Get("code")
		state := query.Get("state")

		pendingState, err := db.ValidateAndDeletePendingOAuthState(ctx, state)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error validating state: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if pendingState.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error invalid state parameter"),
				http.StatusBadRequest,
				true,
			)
			return
		}

		tokenResp, err := oa.ExchangeCode(ctx, code, pendingState.CodeVerifier)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error exchanging code for token: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		expiresAt := time.Now().UTC().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		refreshExpiresAt := time.Now().UTC().Add(30 * 24 * time.Hour) // Default 30 days specific for refresh token if not provided.

		authExpiresAt := time.Now().UTC().Add(24 * time.Hour)
		err = db.UpsertAuthToken(ctx, pendingState.UserID, providerName, code, state, authExpiresAt)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error storing auth token: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		err = db.UpsertAccessToken(ctx, pendingState.UserID, providerName, tokenResp.AccessToken, tokenResp.RefreshToken, expiresAt, refreshExpiresAt)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error storing access token: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		http.Redirect(
			w,
			r,
			fmt.Sprintf("/callback?status=success&provider=%s", providerName),
			http.StatusTemporaryRedirect,
		)
	}
}

// GetToken handles GET /api/v1/authorizations/{provider}/token
//
//	@Summary	Get a provider access token
//	@Tags		authorization
//	@Produce	json
//	@Success	200	{object}	vibe.ProviderTokenResponse
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	412	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/tokens/spotify [get]
//	@Router		/api/v1/tokens/soundcloud [get]
//	@Router		/api/v1/tokens/youtube [get]
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

			handleError(
				w,
				fmt.Errorf("error getting access token: %w", err),
				http.StatusForbidden,
				false,
			)
			return
		}

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

		if token.RefreshToken == "" || !token.RefreshExpiresAt.IsZero() && token.RefreshExpiresAt.Before(time.Now().UTC()) {
			handleError(
				w,
				fmt.Errorf("error refresh token expired"),
				http.StatusPreconditionFailed,
				true,
			)
			return
		}

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
