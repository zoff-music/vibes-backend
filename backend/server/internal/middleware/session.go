package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/zoff-music/vibes/server/internal/helper"
)

// SessionMiddleware extracts the session from the "session" cookie or creates a new one
func SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload helper.SessionPayload
		hasSession := false

		cookie, err := r.Cookie("session")
		if err == nil {
			decoded, err := base64.StdEncoding.DecodeString(cookie.Value)
			err = json.Unmarshal(decoded, &payload)
			if err == nil && payload.UserID != "" {
				hasSession = true
			}
		}

		if !hasSession {
			// No valid session, create a new one
			payload.UserID = uuid.New().String()
			log.Printf("SessionMiddleware: generated new userID=%s", payload.UserID)

			sessionJSON, _ := json.Marshal(payload)
			sessionEncoded := base64.StdEncoding.EncodeToString(sessionJSON)

			http.SetCookie(w, &http.Cookie{
				Name:     "session",
				Value:    sessionEncoded,
				Path:     "/",
				HttpOnly: true,
				Secure:   false,
				SameSite: http.SameSiteLaxMode,
			})
		}

		ctx := context.WithValue(r.Context(), helper.SessionKey, payload)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
