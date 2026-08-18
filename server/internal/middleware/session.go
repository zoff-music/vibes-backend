package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

type SessionMiddleware struct {
	Secret                     string
	CastTokenSecret            string
	EmbedBasePath              string
	RemoteControlAuthenticator vibe.RemoteControlAuthenticator
}

// Middleware extracts the appropriate session cookie or creates a new one.
func (m *SessionMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1) If caller provides a Bearer token, it must be valid (no silent fallback).
		authz := r.Header.Get("Authorization")
		if authz != "" {
			if !strings.HasPrefix(authz, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			token := strings.TrimSpace(strings.TrimPrefix(authz, "Bearer "))
			if token == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			castPayload, err := helper.VerifyCastToken(m.CastTokenSecret, token, time.Now())
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			payload := helper.SessionPayload{
				UserID:     castPayload.UserID,
				AuthType:   "cast",
				CastRoomID: castPayload.RoomID,
			}

			// Prevent a cast token for room A from being used against room B endpoints.
			vars := mux.Vars(r)
			if vars != nil {
				roomID, ok := vars["id"]
				if ok && roomID != "" && roomID != payload.CastRoomID {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
			}

			ctx := context.WithValue(r.Context(), helper.SessionKey, payload)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		remoteID := strings.TrimSpace(r.Header.Get(remoteRequestHeader))
		if remoteID != "" {
			currentRoute := mux.CurrentRoute(r)
			if currentRoute == nil {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			routeName := currentRoute.GetName()
			if !remoteRouteNames[routeName] {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			controllerToken := strings.TrimSpace(r.Header.Get(remoteSessionHeader))
			if controllerToken == "" {
				cookie, err := r.Cookie(remoteSessionCookieName)
				if err != nil || cookie.Value == "" {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				controllerToken = cookie.Value
			}

			controllerTokenHash := helper.HashRemoteCredential(m.Secret, controllerToken)
			remote, err := m.RemoteControlAuthenticator.AuthenticateRemoteControl(
				r.Context(),
				remoteID,
				controllerTokenHash,
			)
			if err != nil {
				log.Printf("SessionMiddleware: error authenticating remote control: %v", err)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if remote.IsEmpty() {
				http.Error(w, "remote machine is unavailable", http.StatusUnauthorized)
				return
			}

			if remoteRoomRouteNames[routeName] {
				vars := mux.Vars(r)
				roomID := vars["id"]
				if roomID == "" || roomID != remote.CurrentRoomID {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
			}

			payload := helper.SessionPayload{
				UserID:       remote.OwnerUserID,
				AuthType:     "remote",
				RemoteID:     remote.ID,
				RemoteRoomID: remote.CurrentRoomID,
				EventOrigin:  vibe.RoomEventOriginRemote,
			}
			ctx := context.WithValue(r.Context(), helper.SessionKey, payload)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// 2) Otherwise, use the isolated signed cookie session (or create a new one).
		embedRequest := r.Header.Get(embedRequestHeader) == embedRequestHeaderValue
		if !embedRequest {
			embedBasePath := "/" + strings.Trim(m.EmbedBasePath, "/")
			referer := r.Referer()
			if referer != "" {
				refererURL, err := url.Parse(referer)
				if err != nil {
					log.Printf("SessionMiddleware: error parsing referer in Middleware: %v", err)
				}
				if err == nil && (refererURL.Path == embedBasePath || strings.HasPrefix(refererURL.Path, embedBasePath+"/")) {
					embedRequest = true
				}
			}
		}

		cookieName := sessionCookieName
		sameSite := http.SameSiteLaxMode
		if embedRequest {
			cookieName = embedSessionCookieName
			sameSite = http.SameSiteNoneMode
		}

		payload, ok := m.extractSession(r, cookieName)
		if !ok {
			createdPayload, err := m.createNewSession(w, cookieName, sameSite)
			if err != nil {
				log.Printf("SessionMiddleware: %v", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			payload = *createdPayload
		}

		ctx := context.WithValue(r.Context(), helper.SessionKey, payload)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *SessionMiddleware) extractSession(r *http.Request, cookieName string) (helper.SessionPayload, bool) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return helper.SessionPayload{}, false
	}

	raw, ok := m.unsign(cookie.Value)
	if !ok {
		log.Printf("SessionMiddleware: cookie signature invalid")
		return helper.SessionPayload{}, false
	}

	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		log.Printf("SessionMiddleware: invalid base64: %v", err)
		return helper.SessionPayload{}, false
	}

	var payload helper.SessionPayload
	err = json.Unmarshal(decoded, &payload)
	if err != nil {
		log.Printf("SessionMiddleware: invalid json: %v", err)
		return helper.SessionPayload{}, false
	}

	if payload.UserID == "" {
		return helper.SessionPayload{}, false
	}

	// Backwards compatibility: old cookies may not have AuthType.
	if payload.AuthType == "" {
		payload.AuthType = "cookie"
	}

	return payload, true
}

func (m *SessionMiddleware) createNewSession(w http.ResponseWriter, cookieName string, sameSite http.SameSite) (*helper.SessionPayload, error) {
	userID := uuid.New().String()
	payload := helper.SessionPayload{
		UserID:   userID,
		IsNew:    true,
		AuthType: "cookie",
	}

	sessionJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling session in createNewSession: %w", err)
	}
	sessionEncoded := base64.StdEncoding.EncodeToString(sessionJSON)
	signed := m.sign(sessionEncoded)

	// #nosec G124 -- all session cookies are Secure and HttpOnly; SameSite is restricted to Lax or None above.
	http.SetCookie(w, &http.Cookie{
		Name:        cookieName,
		Value:       signed,
		Path:        "/",
		HttpOnly:    true,
		Secure:      true,
		SameSite:    sameSite,
		Partitioned: sameSite == http.SameSiteNoneMode,
	})

	return &payload, nil
}

func (m *SessionMiddleware) sign(value string) string {
	if m.Secret == "" {
		return value
	}
	mac := hmac.New(sha256.New, []byte(m.Secret))
	mac.Write([]byte(value))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return value + "." + signature
}

func (m *SessionMiddleware) unsign(value string) (string, bool) {
	if m.Secret == "" {
		return value, true
	}

	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return "", false
	}

	payload, signature := parts[0], parts[1]
	mac := hmac.New(sha256.New, []byte(m.Secret))
	mac.Write([]byte(payload))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return payload, hmac.Equal([]byte(signature), []byte(expected))
}

const sessionCookieName = "session"
const embedSessionCookieName = "embed_session"
const embedRequestHeader = "X-Zoff-Embed"
const embedRequestHeaderValue = "true"

const remoteRequestHeader = "X-Zoff-Remote-ID"

const remoteSessionHeader = "X-Zoff-Remote-Token"

const remoteSessionCookieName = "remote_session"

var remoteRouteNames = map[string]bool{
	"GetRemoteControl":          true,
	"UpdateRemoteControl":       true,
	"RemoteEvents":              true,
	"GetRoom":                   true,
	"UpdateRoomSettings":        true,
	"SkipSong":                  true,
	"GetPlaybackState":          true,
	"UpdatePlaybackState":       true,
	"CreateSession":             true,
	"GetSongs":                  true,
	"AddSong":                   true,
	"AddPlaylist":               true,
	"RemoveSong":                true,
	"VoteSong":                  true,
	"RoomEvents":                true,
	"SearchMusic":               true,
	"GetMusicTrack":             true,
	"GetYouTubePlaylist":        true,
	"SearchSoundCloud":          true,
	"ResolveSoundCloudTrack":    true,
	"GetSoundCloudTrack":        true,
	"ResolveSoundCloudPlaylist": true,
	"GetProviders":              true,
}

var remoteRoomRouteNames = map[string]bool{
	"GetRoom":             true,
	"UpdateRoomSettings":  true,
	"SkipSong":            true,
	"GetPlaybackState":    true,
	"UpdatePlaybackState": true,
	"CreateSession":       true,
	"GetSongs":            true,
	"AddSong":             true,
	"AddPlaylist":         true,
	"RemoveSong":          true,
	"VoteSong":            true,
	"RoomEvents":          true,
}
