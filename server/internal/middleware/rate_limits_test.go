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

func (s *rateLimitConsumerStub) ConsumeRateLimit(_ context.Context, request vibe.RateLimitRequest) (vibe.RateLimitResult, error) {
	s.request = request
	s.calls++
	return s.result, s.err
}

func TestRateLimitMiddlewareAllowsConfiguredRoute(t *testing.T) {
	consumer := &rateLimitConsumerStub{
		result: vibe.RateLimitResult{Allowed: true},
	}
	middleware := &RateLimitMiddleware{
		DB: consumer,
		Policies: map[string]vibe.RateLimitPolicy{
			"GetRoom": {Rate: time.Minute, Limit: 12},
		},
	}
	router := rateLimitTestRouter(middleware)
	request := httptest.NewRequest(http.MethodGet, "/rooms/electro", nil)
	request.RemoteAddr = "10.0.0.8:4242"
	request.Header.Set("X-Forwarded-For", "198.51.100.20, 203.0.113.7")
	session := helper.SessionPayload{UserID: "session-one", AuthType: "cookie"}
	request = request.WithContext(context.WithValue(request.Context(), helper.SessionKey, session))
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if consumer.calls != 1 {
		t.Fatalf("expected one rate limit call, got %d", consumer.calls)
	}
	if consumer.request.RouteName != "GetRoom" {
		t.Fatalf("expected GetRoom route, got %q", consumer.request.RouteName)
	}
	if consumer.request.DeviceLimit != 12 {
		t.Fatalf("expected device limit 12, got %d", consumer.request.DeviceLimit)
	}
	if consumer.request.IPLimit != 120 {
		t.Fatalf("expected IP limit 120, got %d", consumer.request.IPLimit)
	}
	expectedDeviceHash := hashRateLimitIdentity("203.0.113.7", "session-one")
	if consumer.request.DeviceIdentityHash != expectedDeviceHash {
		t.Fatalf("expected device identity hash %q, got %q", expectedDeviceHash, consumer.request.DeviceIdentityHash)
	}
	expectedIPHash := hashRateLimitIdentity("203.0.113.7")
	if consumer.request.IPIdentityHash != expectedIPHash {
		t.Fatalf("expected IP identity hash %q, got %q", expectedIPHash, consumer.request.IPIdentityHash)
	}
}

func TestRateLimitMiddlewareRejectsExceededRoute(t *testing.T) {
	consumer := &rateLimitConsumerStub{
		result: vibe.RateLimitResult{Allowed: false, RetryAfter: 3 * time.Second},
	}
	middleware := &RateLimitMiddleware{
		DB: consumer,
		Policies: map[string]vibe.RateLimitPolicy{
			"GetRoom": {Rate: time.Minute, Limit: 12},
		},
	}
	router := rateLimitTestRouter(middleware)
	request := httptest.NewRequest(http.MethodGet, "/rooms/electro", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, recorder.Code)
	}
	if recorder.Header().Get("Retry-After") != "3" {
		t.Fatalf("expected Retry-After 3, got %q", recorder.Header().Get("Retry-After"))
	}
}

func TestRateLimitMiddlewareSkipsUnconfiguredRoutesAndOptions(t *testing.T) {
	consumer := &rateLimitConsumerStub{
		result: vibe.RateLimitResult{Allowed: true},
	}
	middleware := &RateLimitMiddleware{
		DB:       consumer,
		Policies: map[string]vibe.RateLimitPolicy{},
	}
	router := rateLimitTestRouter(middleware)
	request := httptest.NewRequest(http.MethodGet, "/rooms/electro", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if consumer.calls != 0 {
		t.Fatalf("expected no calls for unconfigured route, got %d", consumer.calls)
	}

	middleware.Policies["GetRoom"] = vibe.RateLimitPolicy{Rate: time.Minute, Limit: 12}
	optionsRequest := httptest.NewRequest(http.MethodOptions, "/rooms/electro", nil)
	optionsRecorder := httptest.NewRecorder()
	router.ServeHTTP(optionsRecorder, optionsRequest)

	if optionsRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected OPTIONS status %d, got %d", http.StatusNoContent, optionsRecorder.Code)
	}
	if consumer.calls != 0 {
		t.Fatalf("expected no calls for OPTIONS, got %d", consumer.calls)
	}
}

func TestHashRateLimitIdentityDoesNotExposeInput(t *testing.T) {
	identity := hashRateLimitIdentity("203.0.113.7", "session-one")
	if len(identity) != 64 {
		t.Fatalf("expected SHA-256 hash length 64, got %d", len(identity))
	}
	if strings.Contains(identity, "session-one") || strings.Contains(identity, "203.0.113.7") {
		t.Fatalf("expected identity to contain only a hash, got %q", identity)
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
