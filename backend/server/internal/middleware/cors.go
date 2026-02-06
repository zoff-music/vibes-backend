package middleware

import (
	"net/http"
	"os"
	"strings"
	"sync"
)

var (
	corsOnce           sync.Once
	corsAllowedOrigins map[string]struct{}
)

func loadCORSAllowedOrigins() {
	corsAllowedOrigins = map[string]struct{}{}
	raw := os.Getenv("CORS_ALLOWED_ORIGINS")
	for _, part := range strings.Split(raw, ",") {
		o := strings.TrimSpace(part)
		if o == "" {
			continue
		}
		corsAllowedOrigins[o] = struct{}{}
	}
}

func originAllowed(origin string) bool {
	corsOnce.Do(loadCORSAllowedOrigins)
	if origin == "" {
		return false
	}
	_, ok := corsAllowedOrigins[origin]
	return ok
}

// CORSMiddleware handles Cross-Origin Resource Sharing (CORS) headers.
// This middleware only allows credentialed CORS requests from an explicit
// allowlist (CORS_ALLOWED_ORIGINS).
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Always set these for consistency.
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-Session-Token, Cache-Control")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// If the request is a browser CORS request (Origin present), enforce allowlist.
		if origin != "" {
			if originAllowed(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Add("Vary", "Origin")
			} else {
				// Disallowed origin.
				if r.Method == http.MethodOptions {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
				// For non-preflight requests, omit CORS headers and continue.
			}
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
