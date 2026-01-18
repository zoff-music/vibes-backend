package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

const SessionKey = "session"

type SessionPayload struct {
	UserID string `json:"user_id"`
}

// SessionMiddleware extracts the session from the "session" cookie and adds it to the context
func SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			// No cookie, proceed without session
			log.Debug("SessionMiddleware: no session cookie found")
			next.ServeHTTP(w, r)
			return
		}

		log.Debugf("SessionMiddleware: found session cookie: %s", cookie.Value)

		decoded, err := base64.StdEncoding.DecodeString(cookie.Value)
		if err != nil {
			// Invalid encoding, ignore
			log.Warnf("SessionMiddleware: failed to decode cookie: %v", err)
			next.ServeHTTP(w, r)
			return
		}

		var payload SessionPayload
		if err := json.Unmarshal(decoded, &payload); err != nil {
			// Invalid JSON, ignore
			log.Warnf("SessionMiddleware: failed to unmarshal payload: %v", err)
			next.ServeHTTP(w, r)
			return
		}

		log.Debugf("SessionMiddleware: parsed userID=%s", payload.UserID)
		ctx := context.WithValue(r.Context(), SessionKey, payload)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
