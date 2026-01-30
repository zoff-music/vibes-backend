package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/zoff-music/vibes/server/internal/helper"
)

type SessionMiddleware struct {
	Secret string
}

// Middleware extracts the session from the "session" cookie or creates a new one
func (m *SessionMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, ok := m.extractSession(r)
		if !ok {
			// Check if this is a Cast Receiver request
			isCast := r.Header.Get("X-Cast-Receiver") == "1"
			casterID := r.Header.Get("X-Cast-Caster-Id")

			if isCast && casterID != "" {
				payload = helper.SessionPayload{UserID: casterID}
				ok = true
			} else {
				payload = m.createNewSession(w, r)
			}
		}

		ctx := context.WithValue(r.Context(), helper.SessionKey, payload)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *SessionMiddleware) extractSession(r *http.Request) (helper.SessionPayload, bool) {
	cookie, err := r.Cookie("session")
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

	return payload, true
}

func (m *SessionMiddleware) createNewSession(w http.ResponseWriter, _ *http.Request) helper.SessionPayload {
	userID := uuid.New().String()
	payload := helper.SessionPayload{UserID: userID}

	sessionJSON, _ := json.Marshal(payload)
	sessionEncoded := base64.StdEncoding.EncodeToString(sessionJSON)
	signed := m.sign(sessionEncoded)

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    signed,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	return payload
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
