package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/client"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

type RateLimitMiddleware struct {
	Checker  vibe.RateLimitChecker
	Policies map[string]vibe.RateLimitPolicy
}

func (m *RateLimitMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		if r.Header.Get(rateLimitRequestOriginHeader) != rateLimitExternalRequestOrigin {
			next.ServeHTTP(w, r)
			return
		}

		routeName := mux.CurrentRoute(r).GetName()
		policy, ok := m.Policies[routeName]
		if !ok {
			policy = vibe.RateLimitPolicy{
				Rate:  rateLimitDefaultRate,
				Limit: rateLimitDefaultLimit,
			}
		}

		bucket := policy.Bucket
		if bucket == "" {
			bucket = routeName
		}

		if policy.Rate <= 0 || policy.Limit <= 0 || policy.Rate/time.Duration(policy.Limit) < time.Microsecond {
			log.Printf("RateLimitMiddleware: invalid policy for route %s", routeName)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		ipLimit := policy.IPLimit
		if ipLimit == 0 {
			ipLimit = policy.Limit * rateLimitIPMultiplier
		}
		if ipLimit < 0 {
			log.Printf("RateLimitMiddleware: invalid IP policy for route %s", routeName)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if policy.GlobalLimit < 0 ||
			(policy.GlobalLimit > 0 && policy.GlobalRate <= 0) {
			log.Printf("RateLimitMiddleware: invalid global policy for route %s", routeName)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if policy.GlobalLimit > 0 {
			globalRequest := vibe.RateLimitRequest{
				RouteName:      bucket,
				IdentityHash:   rateLimitGlobalIdentity,
				IPIdentityHash: rateLimitGlobalIdentity,
				Rate:           policy.GlobalRate,
				Limit:          policy.GlobalLimit,
				IPLimit:        policy.GlobalLimit,
			}
			globalResult, err := m.Checker.CheckRateLimit(r.Context(), globalRequest)
			if err != nil {
				log.Printf("RateLimitMiddleware: error checking global limit for route %s: %v", routeName, err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			if !globalResult.Allowed {
				retryAfter := max(globalResult.RetryAfter, time.Second)
				retryAfterSeconds := (retryAfter + time.Second - 1) / time.Second
				body, err := json.Marshal(client.ErrorCodeResponseBody{
					Namespace: "vibes-backend",
					Error:     "rate_limit",
					Message:   "Too many requests. Please wait and try again.",
					Propagate: true,
				})
				if err != nil {
					log.Printf("RateLimitMiddleware: error marshalling rate limit response for route %s: %v", routeName, err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Retry-After", strconv.FormatInt(int64(retryAfterSeconds), 10))
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-preserve-error", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write(body)
				return
			}
		}

		session, hasSession := helper.GetSessionFromContext(r.Context())
		clientIP := rateLimitClientIP(r)
		deviceIdentity := strings.Join(
			[]string{clientIP, strings.TrimSpace(r.UserAgent())},
			"\x00",
		)
		if hasSession && session.UserID != "" {
			deviceIdentity = session.UserID
		}

		request := vibe.RateLimitRequest{
			RouteName:      bucket,
			IdentityHash:   hashRateLimitIdentity(deviceIdentity),
			IPIdentityHash: hashRateLimitIdentity(clientIP),
			Rate:           policy.Rate,
			Limit:          policy.Limit,
			IPLimit:        ipLimit,
		}
		result, err := m.Checker.CheckRateLimit(r.Context(), request)
		if err != nil {
			log.Printf("RateLimitMiddleware: error checking limit for route %s: %v", routeName, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if !result.Allowed {
			retryAfter := max(result.RetryAfter, time.Second)
			retryAfterSeconds := (retryAfter + time.Second - 1) / time.Second
			body, err := json.Marshal(client.ErrorCodeResponseBody{
				Namespace: "vibes-backend",
				Error:     "rate_limit",
				Message:   "Too many requests. Please wait and try again.",
				Propagate: true,
			})
			if err != nil {
				log.Printf("RateLimitMiddleware: error marshalling rate limit response for route %s: %v", routeName, err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Retry-After", strconv.FormatInt(int64(retryAfterSeconds), 10))
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-preserve-error", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write(body)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func rateLimitClientIP(r *http.Request) string {
	forwardedFor := r.Header.Get("X-Forwarded-For")
	forwardedAddresses := strings.Split(forwardedFor, ",")
	for i := len(forwardedAddresses) - 1; i >= 0; i-- {
		address := strings.TrimSpace(forwardedAddresses[i])
		parsedAddress, err := netip.ParseAddr(address)
		if err == nil {
			clientIP := parsedAddress.Unmap().String()

			return clientIP
		}
	}

	realIP := strings.TrimSpace(r.Header.Get("X-Real-Ip"))
	parsedRealIP, err := netip.ParseAddr(realIP)
	if err == nil {
		clientIP := parsedRealIP.Unmap().String()

		return clientIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		parsedHost, err := netip.ParseAddr(host)
		if err == nil {
			clientIP := parsedHost.Unmap().String()

			return clientIP
		}
	}

	parsedRemoteAddress, err := netip.ParseAddr(r.RemoteAddr)
	if err == nil {
		clientIP := parsedRemoteAddress.Unmap().String()

		return clientIP
	}

	return "unknown-ip"
}

func hashRateLimitIdentity(parts ...string) string {
	hash := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	identityHash := hex.EncodeToString(hash[:])

	return identityHash
}

const rateLimitIPMultiplier = 10

const rateLimitDefaultRate = time.Minute

const rateLimitDefaultLimit = 60

const rateLimitRequestOriginHeader = "X-Vibes-Request-Origin"

const rateLimitExternalRequestOrigin = "external"

const rateLimitGlobalIdentity = "ZOFF:GLOBAL"
