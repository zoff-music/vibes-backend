package middleware

import (
	"net/http"
	"strings"
	"sync"
)

type CORSMiddleware struct {
	AllowedOriginsCSV string

	allowedOrigins map[string]struct{}
	initOnce       sync.Once
}

func (m *CORSMiddleware) init() {
	m.allowedOrigins = map[string]struct{}{}
	for _, part := range strings.Split(m.AllowedOriginsCSV, ",") {
		o := strings.TrimSpace(part)
		if o == "" {
			continue
		}
		m.allowedOrigins[o] = struct{}{}
	}
}

func (m *CORSMiddleware) originAllowed(origin string) bool {
	if origin == "" {
		return false
	}
	m.initOnce.Do(m.init)
	_, ok := m.allowedOrigins[origin]
	return ok
}

// Middleware handles Cross-Origin Resource Sharing (CORS) headers.
// This middleware only allows credentialed CORS requests from an explicit
// allowlist (CORS_ALLOWED_ORIGINS).
func (m *CORSMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.initOnce.Do(m.init)

		origin := r.Header.Get("Origin")

		// Always set these for consistency.
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-Session-Token, Cache-Control")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// If the request is a browser CORS request (Origin present), enforce allowlist.
		if origin != "" {
			if m.originAllowed(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Add("Vary", "Origin")
			} else if r.Method == http.MethodOptions {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			// For non-preflight requests, omit CORS headers and continue.
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
