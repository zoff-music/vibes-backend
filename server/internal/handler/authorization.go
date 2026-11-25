package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// Authorize redirects the user to the provider for authentication
//
//	@Param playlist query string false "Playlist ID or SoundCloud URL for a single-use import"
//	@Param channel query string false "Same-origin browser callback channel UUID"
//	@Summary	Start provider authorization
//	@Tags		authorization
//	@Produce	json
//	@Success	307	"Redirect; follow the Location response header"
//	@Header	307	{string}	Location	"Redirect destination"
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/authorizations/soundcloud [get]
//	@Router		/api/v1/authorizations/youtube [get]
func Authorize(db vibe.PendingOAuthStateSaver, oa vibe.OAuthAuthorizer, providerName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		state, err := generateRandomString(32)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error generating OAuth state: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

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

		codeVerifier, err := generateRandomString(43) // length between 43 and 128 for PKCE
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error generating PKCE code verifier: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		source := r.URL.Query().Get("playlist")
		if source != "" {
			if providerName == vibe.SourceTypeSoundCloud {
				source, err = vibe.ResolveSoundCloudPlaylistURL(source)
				if err != nil {
					handleError(
						w,
						fmt.Errorf("error validating playlist: %w", err),
						http.StatusBadRequest,
						false,
					)
					return
				}
			}
			channel := r.URL.Query().Get("channel")
			if channel != "" {
				parsed, parseErr := uuid.Parse(channel)
				if parseErr != nil {
					handleError(
						w,
						fmt.Errorf("error validating callback channel: %w", parseErr),
						http.StatusBadRequest,
						false,
					)
					return
				}
				channel = parsed.String()
			}
			state = "playlist." + playlistHash(providerName+"."+source) + "." + channel + "." + state
		}
		state = providerName + "." + state
		err = db.SavePendingOAuthState(ctx, session.UserID, state, codeVerifier)
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
		// #nosec G710 -- each provider constructs a fixed HTTPS origin and URL-encodes state as a query parameter.
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
//	@Success 200 {string} string "Single-use playlist import code page"
//	@Summary	Handle provider authorization callback
//	@Tags		authorization
//	@Param		code	query	string	true	"Authorization code"
//	@Param		state	query	string	true	"OAuth state"
//	@Success	307	"Redirect; follow the Location response header"
//	@Header	307	{string}	Location	"Redirect destination"
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Router		/api/v1/callbacks/soundcloud [get]
//	@Router		/api/v1/callbacks/youtube [get]
func OAuthCallback(db vibe.CodeValidatorUpserter, oa vibe.OAuthExchanger, providerName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := r.URL.Query()
		code := query.Get("code")
		state := query.Get("state")

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" || session.AuthType != "cookie" {
			handleError(
				w,
				fmt.Errorf("error OAuth callback requires the initiating browser session"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		if code == "" || !strings.HasPrefix(state, providerName+".") {
			handleError(
				w,
				fmt.Errorf("error invalid OAuth callback parameters"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		pendingState, err := db.ConsumePendingOAuthState(ctx, session.UserID, state)
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
		if strings.HasPrefix(state, providerName+".playlist.") {
			parts := strings.Split(state, ".")
			if len(parts) != 5 {
				handleError(
					w,
					fmt.Errorf("error invalid playlist state"),
					http.StatusBadRequest,
					false,
				)
				return
			}
			receipt, err := generateRandomString(18)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error creating import code: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}
			deadline := time.Now().Add(5 * time.Minute)
			if expiresAt.Before(deadline) {
				deadline = expiresAt
			}
			err = db.SavePlaylistAuthorization(ctx, playlistHash(receipt), parts[2], tokenResp.AccessToken, deadline)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error saving import authorization: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			scriptDigest := sha256.Sum256([]byte(playlistCallbackScript))
			w.Header().Set("Content-Security-Policy", "script-src 'sha256-"+base64.StdEncoding.EncodeToString(scriptDigest[:])+"'; default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
			_, err = fmt.Fprintf(w, playlistCallbackPage, template.HTMLEscapeString(parts[3]), template.HTMLEscapeString(receipt), playlistCallbackScript)
			if err != nil {
				log.Printf("error writing import code page: %v", err)
			}
			return
		}

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
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	403	{object}	vibe.ErrorResponse
//	@Failure	412	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
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
			handleError(
				w,
				fmt.Errorf("error getting access token: %w", err),
				http.StatusForbidden,
				false,
			)
			return
		}

		if token.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error getting access token: access token not found"),
				http.StatusPreconditionFailed,
				false,
			)
			return
		}

		if token.ExpiresAt.After(time.Now().UTC().Add(1 * time.Minute)) {
			resp := vibe.ProviderTokenResponse{
				AccessToken: token.AccessToken,
				ExpiresAt:   token.ExpiresAt,
			}

			// #nosec G117 -- the authorized client requires its provider token for SDK playback.
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

		// #nosec G117 -- the authorized client requires its provider token for SDK playback.
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

func generateRandomString(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("error reading cryptographic random bytes: %w", err)
	}

	value := base64.RawURLEncoding.EncodeToString(b)
	return value, nil
}

// CleanupExpiredTokens cleans up expired external auth records
type CleanupExpiredTokens struct {
	DB vibe.AuthTokenCleaner
}

// Handle deletes expired external auth tokens
func (h *CleanupExpiredTokens) Handle(ctx context.Context, _ []byte) error {
	deletedAuth, err := h.DB.DeleteExpiredAuthTokens(ctx)
	if err != nil {
		return fmt.Errorf("error deleting expired auth tokens in CleanupExpiredTokens.Handle: %w", err)
	}

	deletedAccess, err := h.DB.DeleteExpiredAccessTokens(ctx)
	if err != nil {
		return fmt.Errorf("error deleting expired access tokens in CleanupExpiredTokens.Handle: %w", err)
	}

	totalDeleted := deletedAuth + deletedAccess
	if totalDeleted > 0 {
		log.Printf("Cleaned up %d expired auth tokens and %d expired access tokens", deletedAuth, deletedAccess)
	}

	return nil
}

// RefreshYouTubeTokens refreshes expired YouTube access tokens
type RefreshYouTubeTokens struct {
	DB       vibe.ExpiredTokenClaimUpdater
	Provider vibe.TokenRefresher
}

// Handle refreshes the next expired YouTube token
func (h *RefreshYouTubeTokens) Handle(ctx context.Context, _ []byte) error {
	token, err := h.DB.ClaimAndGetExpiredTokenForRefresh(ctx, "youtube")
	if err != nil {
		return fmt.Errorf("error claiming expired token for refresh in youtube handler: %w", err)
	}

	newToken, err := h.Provider.RefreshToken(ctx, token.RefreshToken)
	if err != nil {
		return fmt.Errorf("error refreshing YouTube token in RefreshYouTubeTokens.Handle: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(newToken.ExpiresIn) * time.Second)
	refreshToken := newToken.RefreshToken
	if refreshToken == "" {
		refreshToken = token.RefreshToken
	}

	err = h.DB.UpsertAccessToken(ctx, token.UserID, "youtube", newToken.AccessToken, refreshToken, expiresAt, token.RefreshExpiresAt)
	if err != nil {
		return fmt.Errorf("error upserting access token in RefreshYouTubeTokens.Handle: %w", err)
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
	deleted, err := h.DB.DeleteExpiredPendingOAuthStates(ctx)
	if err != nil {
		return fmt.Errorf("error deleting expired pending OAuth states in CleanupExpiredPendingOAuthStates.Handle: %w", err)
	}

	if deleted > 0 {
		log.Printf("Cleaned up %d expired pending OAuth states", deleted)
	}

	return nil
}

func playlistHash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

const playlistCallbackScript = `
const code = document.querySelector('code');
if (code.dataset.channel && typeof BroadcastChannel !== 'undefined') {
    const channel = new BroadcastChannel(code.dataset.channel);
    channel.onmessage = ({data}) => {
        if (data === 'received') {
            channel.close();
            window.close();
        }
    };
    channel.postMessage(code.textContent);
}
`

const playlistCallbackPage = `<!doctype html>
<html lang="en">
<head>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Playlist authorized</title>
</head>
<body>
    <h1>Playlist authorized</h1>
    <p>Return to Zoff. If prompted, paste this single-use import code within five minutes.</p>
    <code data-channel="%s">%s</code>
    <script>%s</script>
</body>
</html>`
