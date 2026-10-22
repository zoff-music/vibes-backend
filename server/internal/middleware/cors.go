package middleware

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type CORSMiddleware struct {
	AllowedOriginsCSV string

	allowedOrigins map[string]bool
	initOnce       sync.Once
}

func (m *CORSMiddleware) init() {
	m.allowedOrigins = map[string]bool{}
	for _, part := range strings.Split(m.AllowedOriginsCSV, ",") {
		o := strings.TrimSpace(part)
		if o == "" {
			continue
		}
		m.allowedOrigins[o] = true
	}
}

func (m *CORSMiddleware) originAllowed(origin string) bool {
	if origin == "" || origin == "null" {
		return false
	}
	m.initOnce.Do(m.init)
	return m.allowedOrigins[origin]
}

// Middleware handles CORS and rejects untrusted browser mutations before they
// reach handlers. Omitting CORS response headers alone does not prevent CSRF.
func (m *CORSMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.initOnce.Do(m.init)

		origin := r.Header.Get("Origin")
		safe := r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions
		allowed := len(r.Header.Values("Origin")) == 1 && m.originAllowed(origin)

		w.Header().Add("Vary", "Origin")

		if !safe {
			w.Header().Add("Vary", "Sec-Fetch-Site")
			w.Header().Add("Vary", "Referer")

			if len(r.Header.Values("Origin")) > 0 && !allowed {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			if len(r.Header.Values("Origin")) == 0 {
				// Older browsers may send only Referer. Native and server clients
				// without browser provenance remain supported.
				referer := r.Header.Get("Referer")
				if referer != "" {
					parsed, err := url.Parse(referer)
					if err != nil || parsed.User != nil || !m.originAllowed(parsed.Scheme+"://"+parsed.Host) {
						http.Error(w, "forbidden", http.StatusForbidden)
						return
					}
				}

				site := r.Header.Get("Sec-Fetch-Site")
				if site != "" && site != "same-origin" && site != "none" {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
			}
		}

		// Always set these for consistency.
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-Session-Token, X-Zoff-Embed, X-Zoff-Remote-ID, X-Zoff-Remote-Token, Cache-Control")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, Retry-After")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// If the request is a browser CORS request (Origin present), enforce allowlist.
		if origin != "" {
			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			if !allowed && r.Method == http.MethodOptions {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
