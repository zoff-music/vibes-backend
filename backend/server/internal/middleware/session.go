package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
)

type key int

const (
	SessionKey key = iota
)

type SessionPayload struct {
	UserID string `json:"user_id"`
}

// SessionMiddleware extracts the session from the "session" cookie and adds it to the context
func SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			// No cookie, proceed without session
			next.ServeHTTP(w, r)
			return
		}

		decoded, err := base64.StdEncoding.DecodeString(cookie.Value)
		if err != nil {
			// Invalid encoding, ignore
			next.ServeHTTP(w, r)
			return
		}

		var payload SessionPayload
		if err := json.Unmarshal(decoded, &payload); err != nil {
			// Invalid JSON, ignore
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), SessionKey, payload)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
