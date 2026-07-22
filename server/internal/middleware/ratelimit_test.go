package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

type rateLimitConsumerStub struct {
	request vibe.RateLimitRequest
	result  vibe.RateLimitResult
	err     error
	calls   int
}

type rateLimitMiddlewareTestCase struct {
	name                string
	method              string
	policies            map[string]vibe.RateLimitPolicy
	result              vibe.RateLimitResult
	expectedStatus      int
	expectedCalls       int
	expectedDeviceLimit int
	expectedIPLimit     int
	expectedRetryAfter  string
}

type rateLimitIdentityTestCase struct {
	name  string
	parts []string
}

func (s *rateLimitConsumerStub) ConsumeRateLimit(_ context.Context, request vibe.RateLimitRequest) (vibe.RateLimitResult, error) {
	s.request = request
	s.calls++
	return s.result, s.err
}

func TestRateLimitMiddleware(t *testing.T) {
	testCases := []rateLimitMiddlewareTestCase{
		{
			name:   "configured route is allowed",
			method: http.MethodGet,
			policies: map[string]vibe.RateLimitPolicy{
				"GetRoom": {Rate: time.Minute, Limit: 12},
			},
			result:              vibe.RateLimitResult{Allowed: true},
			expectedStatus:      http.StatusNoContent,
			expectedCalls:       1,
			expectedDeviceLimit: 12,
			expectedIPLimit:     120,
		},
		{
			name:                "unconfigured route uses default",
			method:              http.MethodGet,
			policies:            map[string]vibe.RateLimitPolicy{},
			result:              vibe.RateLimitResult{Allowed: true},
			expectedStatus:      http.StatusNoContent,
			expectedCalls:       1,
			expectedDeviceLimit: 60,
			expectedIPLimit:     600,
		},
		{
			name:   "exceeded route is rejected",
			method: http.MethodGet,
			policies: map[string]vibe.RateLimitPolicy{
				"GetRoom": {Rate: time.Minute, Limit: 12},
			},
			result:              vibe.RateLimitResult{Allowed: false, RetryAfter: 1500 * time.Millisecond},
			expectedStatus:      http.StatusTooManyRequests,
			expectedCalls:       1,
			expectedDeviceLimit: 12,
			expectedIPLimit:     120,
			expectedRetryAfter:  "2",
		},
		{
			name:                "options bypasses limiter",
			method:              http.MethodOptions,
			policies:            map[string]vibe.RateLimitPolicy{},
			result:              vibe.RateLimitResult{Allowed: true},
			expectedStatus:      http.StatusNoContent,
			expectedCalls:       0,
			expectedDeviceLimit: 0,
			expectedIPLimit:     0,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			consumer := &rateLimitConsumerStub{result: testCase.result}
			middleware := &RateLimitMiddleware{
				Client:   consumer,
				Policies: testCase.policies,
			}
			router := rateLimitTestRouter(middleware)
			request := httptest.NewRequest(testCase.method, "/rooms/electro", nil)
			request.RemoteAddr = "10.0.0.8:4242"
			request.Header.Set("X-Forwarded-For", "198.51.100.20, 203.0.113.7")
			session := helper.SessionPayload{UserID: "session-one", AuthType: "cookie"}
			request = request.WithContext(context.WithValue(request.Context(), helper.SessionKey, session))
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			if recorder.Code != testCase.expectedStatus {
				t.Fatalf("expected status %d, got %d", testCase.expectedStatus, recorder.Code)
			}
			if consumer.calls != testCase.expectedCalls {
				t.Fatalf("expected %d rate limit calls, got %d", testCase.expectedCalls, consumer.calls)
			}
			if consumer.request.DeviceLimit != testCase.expectedDeviceLimit {
				t.Fatalf("expected device limit %d, got %d", testCase.expectedDeviceLimit, consumer.request.DeviceLimit)
			}
			if consumer.request.IPLimit != testCase.expectedIPLimit {
				t.Fatalf("expected IP limit %d, got %d", testCase.expectedIPLimit, consumer.request.IPLimit)
			}
			if recorder.Header().Get("Retry-After") != testCase.expectedRetryAfter {
				t.Fatalf("expected Retry-After %q, got %q", testCase.expectedRetryAfter, recorder.Header().Get("Retry-After"))
			}
			if testCase.expectedCalls == 0 {
				return
			}

			if consumer.request.RouteName != "GetRoom" {
				t.Fatalf("expected GetRoom route, got %q", consumer.request.RouteName)
			}
			expectedDeviceHash := hashRateLimitIdentity("203.0.113.7", "session-one")
			if consumer.request.DeviceIdentityHash != expectedDeviceHash {
				t.Fatalf("expected device identity hash %q, got %q", expectedDeviceHash, consumer.request.DeviceIdentityHash)
			}
			expectedIPHash := hashRateLimitIdentity("203.0.113.7")
			if consumer.request.IPIdentityHash != expectedIPHash {
				t.Fatalf("expected IP identity hash %q, got %q", expectedIPHash, consumer.request.IPIdentityHash)
			}
		})
	}
}

func TestHashRateLimitIdentity(t *testing.T) {
	testCases := []rateLimitIdentityTestCase{
		{name: "IP and session", parts: []string{"203.0.113.7", "session-one"}},
		{name: "IP only", parts: []string{"203.0.113.7"}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			identity := hashRateLimitIdentity(testCase.parts...)
			if len(identity) != 64 {
				t.Fatalf("expected SHA-256 hash length 64, got %d", len(identity))
			}
			for _, part := range testCase.parts {
				if strings.Contains(identity, part) {
					t.Fatalf("expected identity to contain only a hash, got %q", identity)
				}
			}
		})
	}
}

func rateLimitTestRouter(middleware *RateLimitMiddleware) *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/rooms/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}).Methods(http.MethodGet, http.MethodOptions).Name("GetRoom")
	router.Use(middleware.Middleware)
	return router
}
