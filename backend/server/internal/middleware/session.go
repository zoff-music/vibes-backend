package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"

	"github.com/zoff-music/vibes/server/internal/helper"
)

// SessionMiddleware extracts the session from the "session" cookie and adds it to the context
func SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			// No cookie, proceed without session
			log.Println("SessionMiddleware: no session cookie found")
			next.ServeHTTP(w, r)
			return
		}

		log.Printf("SessionMiddleware: found session cookie: %s", cookie.Value)

		decoded, err := base64.StdEncoding.DecodeString(cookie.Value)
		if err != nil {
			// Invalid encoding, ignore
			log.Printf("SessionMiddleware: failed to decode cookie: %v", err)
			next.ServeHTTP(w, r)
			return
		}

		var payload helper.SessionPayload
		if err := json.Unmarshal(decoded, &payload); err != nil {
			// Invalid JSON, ignore
			log.Printf("SessionMiddleware: failed to unmarshal payload: %v", err)
			next.ServeHTTP(w, r)
			return
		}

		log.Printf("SessionMiddleware: parsed userID=%s", payload.UserID)
		ctx := context.WithValue(r.Context(), helper.SessionKey, payload)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
