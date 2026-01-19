package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes/server/internal/helper"
	"github.com/zoff-music/vibes/vibe"
)

// AuthorizeProvider redirects the user to the provider for authentication
func AuthorizeProvider(oa vibe.OAuthAuthorizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := generateRandomString(32)

		// Store state in a cookie for validation when provider redirects back
		helper.SetOAuthStateCookie(w, state)

		redirectURL := oa.GetOAuthURL(state)
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
	}
}

// GetAuthorizations handles GET /api/v1/authorizations
func GetAuthorizations(db vibe.ExternalAuthLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(w, fmt.Errorf("error user not authenticated"), http.StatusUnauthorized, true)
			return
		}

		auths, err := db.GetExternalAuths(ctx, session.UserID)
		if err != nil {
			handleError(w, fmt.Errorf("error getting authorizations: %w", err), http.StatusInternalServerError, true)
			return
		}

		body, err := json.Marshal(auths)
		if err != nil {
			handleError(w, fmt.Errorf("error marshaling authorizations: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}
}

// OAuthCallback handles GET /api/v1/callbacks/{provider}
func OAuthCallback(db vibe.ExternalAuthUpserter, frontendURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		provider := vars["provider"]

		query := r.URL.Query()
		code := query.Get("code")
		state := query.Get("state")

		// Validate state
		cookieState, err := helper.GetOAuthStateCookie(r)
		if err != nil || cookieState != state {
			handleError(w, fmt.Errorf("invalid state parameter"), http.StatusBadRequest, true)
			return
		}

		// Get user ID from session
		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(w, fmt.Errorf("error user not authenticated"), http.StatusUnauthorized, true)
			return
		}

		// Calculate expiration
		expiresAt := time.Now().Add(1 * time.Hour)

		err = db.UpsertExternalAuth(ctx, session.UserID, provider, code, state, expiresAt)
		if err != nil {
			handleError(w, fmt.Errorf("error storing external auth: %w", err), http.StatusInternalServerError, true)
			return
		}

		// Redirect to frontend callback page
		http.Redirect(w, r, fmt.Sprintf("%s/callback?status=success&provider=%s", frontendURL, provider), http.StatusTemporaryRedirect)
	}
}

// GetSpotifyToken handles GET /api/v1/authorizations/spotify/token
func GetSpotifyToken(db vibe.ExternalAuthGetter, te vibe.TokenExchanger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(w, fmt.Errorf("error user not authenticated"), http.StatusUnauthorized, true)
			return
		}

		auth, err := db.GetExternalAuth(ctx, session.UserID, "spotify")
		if err != nil {
			handleError(w, fmt.Errorf("authorization not found"), http.StatusNotFound, true)
			return
		}

		if auth.Code == "" {
			handleError(w, fmt.Errorf("error no auth code found"), http.StatusNotFound, true)
			return
		}

		// Exchange code for token
		tokenResp, err := te.ExchangeCode(ctx, auth.Code)
		if err != nil {
			handleError(w, fmt.Errorf("error exchanging code: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(tokenResp)
	}
}

func generateRandomString(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
