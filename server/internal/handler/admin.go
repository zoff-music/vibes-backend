package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/zoff-music/vibes-backend/monitoring/opentracing"
	"log"
	"net/http"
	"time"

	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

const adminSessionDuration = 24 * time.Hour

// AdminLogin handles POST /api/v1/admin/sessions
func AdminLogin(
	adminPassword *string,
	cookieSecret string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req vibe.AdminLoginRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("invalid request body: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		if req.Password == "" {
			handleError(
				w,
				fmt.Errorf("password required"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		password := ""
		if adminPassword != nil {
			password = *adminPassword
		}

		if password == "" || req.Password != password {
			handleError(
				w,
				fmt.Errorf("incorrect password"),
				http.StatusForbidden,
				false,
			)
			return
		}

		payload := helper.AdminAuthPayload{
			UserID:       session.UserID,
			PasswordHash: helper.HashAdminPassword(password),
			IssuedAt:     time.Now().Unix(),
		}

		signed, err := helper.SignAdminAuthPayload(payload, cookieSecret)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("failed to sign admin session: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     helper.AdminAuthCookieName,
			Value:    signed,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(adminSessionDuration),
		})

		resp := vibe.AdminSessionResponse{
			Authorized: true,
		}

		body, err := json.Marshal(resp)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("marshal response: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// AdminLogout handles DELETE /api/v1/admin/sessions
func AdminLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     helper.AdminAuthCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})

		resp := vibe.AdminSessionResponse{
			Authorized: false,
		}

		body, err := json.Marshal(resp)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("marshal response: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// AdminRooms handles GET /api/v1/admin/rooms
func AdminRooms(
	db vibe.AdminRoomLister,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		hasAdminCookie := false
		for _, cookie := range r.Cookies() {
			if cookie.Name == helper.AdminAuthCookieName {
				hasAdminCookie = true
				break
			}
		}
		if !hasAdminCookie {
			log.Printf("AdminRooms: request missing admin_session cookie")
		}
		if hasAdminCookie {
			log.Printf("AdminRooms: request has admin_session cookie")
		}

		rooms, err := db.ListAdminRooms(ctx)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("failed to fetch admin rooms: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(rooms)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("marshal response: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// AdminEvents handles GET /api/v1/admin/events (SSE)
func AdminEvents(
	ips vibe.AdminSubscriberPublisher,
	db vibe.AdminRoomLister,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		container, err := ips.Subscribe("admin")
		if err != nil {
			handleError(
				w,
				fmt.Errorf("failed to subscribe to admin events: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		defer container.Subscription.Destroy()

		flusher, ok := w.(http.Flusher)
		if !ok {
			handleError(
				w,
				fmt.Errorf("streaming not supported"),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		fmt.Fprintf(w, "event: connected\ndata: {\"time\": %d}\n\n", time.Now().UnixMilli())
		flusher.Flush()

		rooms, err := db.ListAdminRooms(ctx)
		if err == nil {
			payload, marshalErr := json.Marshal(rooms)
			if marshalErr == nil {
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", vibe.AdminRoomsUpdate, payload)
				flusher.Flush()
			}
		}

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		messages := container.Subscription.Listen()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fmt.Fprintf(w, ": heartbeat\n\n")
				flusher.Flush()
			case data, ok := <-messages:
				if !ok {
					return
				}

				var event vibe.AdminEvent
				err := json.Unmarshal(data, &event)
				if err != nil {
					continue
				}

				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, event.Payload)
				flusher.Flush()
			}
		}
	}
}

// ReviewAdminRooms handles scheduled admin room updates
type ReviewAdminRooms struct {
	DB  vibe.AdminRoomLister
	IPS vibe.AdminEventNotifier
}

// Handle fetches admin rooms and broadcasts the update
func (h *ReviewAdminRooms) Handle(ctx context.Context, data []byte) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Handle")
	defer span.Finish()

	rooms, err := h.DB.ListAdminRooms(ctx)
	if err != nil {
		return fmt.Errorf("error listing admin rooms: %w", err)
	}

	payload, err := json.Marshal(rooms)
	if err != nil {
		return fmt.Errorf("error marshaling admin rooms: %w", err)
	}

	err = h.IPS.NotifyAdminUpdate(ctx, vibe.AdminEvent{
		Type:    vibe.AdminRoomsUpdate,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("error notifying admin rooms update: %w", err)
	}

	return nil
}
