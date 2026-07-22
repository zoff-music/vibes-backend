package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

type RateLimitMiddleware struct {
	DB       vibe.RateLimitConsumer
	Policies map[string]vibe.RateLimitPolicy
}

func (m *RateLimitMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		routeName := mux.CurrentRoute(r).GetName()
		policy, ok := m.Policies[routeName]
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		if policy.Rate <= 0 || policy.Limit <= 0 {
			log.Printf("RateLimitMiddleware: invalid policy for route %s", routeName)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		clientIP := rateLimitClientIP(r)
		session, hasSession := helper.GetSessionFromContext(r.Context())
		deviceIdentity := strings.TrimSpace(r.UserAgent())
		if hasSession && session.UserID != "" {
			deviceIdentity = session.UserID
		}
		if deviceIdentity == "" {
			deviceIdentity = "unknown-device"
		}

		request := vibe.RateLimitRequest{
			RouteName:          routeName,
			DeviceIdentityHash: hashRateLimitIdentity(clientIP, deviceIdentity),
			IPIdentityHash:     hashRateLimitIdentity(clientIP),
			Rate:               policy.Rate,
			DeviceLimit:        policy.Limit,
			IPLimit:            policy.Limit * rateLimitIPMultiplier,
		}
		result, err := m.DB.ConsumeRateLimit(r.Context(), request)
		if err != nil {
			log.Printf("RateLimitMiddleware: error consuming limit for route %s: %v", routeName, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if !result.Allowed {
			retryAfter := max(result.RetryAfter, time.Second)
			w.Header().Set("Retry-After", strconv.FormatInt(int64(retryAfter/time.Second), 10))
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
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
			return parsedAddress.Unmap().String()
		}
	}

	realIP := strings.TrimSpace(r.Header.Get("X-Real-Ip"))
	parsedRealIP, err := netip.ParseAddr(realIP)
	if err == nil {
		return parsedRealIP.Unmap().String()
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		parsedHost, parseErr := netip.ParseAddr(host)
		if parseErr == nil {
			return parsedHost.Unmap().String()
		}
	}

	parsedRemoteAddress, err := netip.ParseAddr(r.RemoteAddr)
	if err == nil {
		return parsedRemoteAddress.Unmap().String()
	}

	return "unknown-ip"
}

func hashRateLimitIdentity(parts ...string) string {
	hash := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(hash[:])
}

const rateLimitIPMultiplier = 10
