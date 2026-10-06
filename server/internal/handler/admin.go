package handler

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// AdminLogin handles POST /api/v1/admin/sessions
//
//	@Summary		Sign in to room administration
//	@Description	Validates an admin username and password and creates an authenticated admin session for the current user.
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		request	body		vibe.AdminLoginRequest	true	"Admin login payload"
//	@Success	200		{object}	vibe.AdminSessionResponse
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	401		{object}	vibe.ErrorResponse
//	@Failure	403		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Router		/api/v1/admin/sessions [post]
func AdminLogin(
	fetcher vibe.AdminUserByUsernameFetcher,
	passwordPepper string,
	cookieSecret string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req vibe.AdminLoginRequest
		err := json.UnmarshalRead(r.Body, &req)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding request body: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		username, err := vibe.NormalizeAdminUsername(req.Username)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error validating admin username: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}

		err = vibe.ValidateAdminPassword(req.Password)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error validating admin password: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		admin, err := fetcher.GetAdminUserByUsername(ctx, username)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error getting admin user in AdminLogin handler: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if admin.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error invalid admin credentials"),
				http.StatusForbidden,
				false,
			)
			return
		}

		passwordMatches, err := helper.VerifyAdminPassword(
			req.Password,
			passwordPepper,
			admin.PasswordHash,
		)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error verifying admin password in AdminLogin handler: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if !passwordMatches {
			handleError(
				w,
				fmt.Errorf("error invalid admin credentials"),
				http.StatusForbidden,
				false,
			)
			return
		}

		payload := vibe.AdminAuthPayload{
			UserID:         session.UserID,
			AdminID:        admin.ID,
			SessionVersion: admin.SessionVersion,
			IssuedAt:       time.Now().Unix(),
		}

		signed, err := helper.SignAdminAuthPayload(payload, cookieSecret)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error signing admin session: %w", err),
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
			User:       admin,
		}

		body, err := json.Marshal(resp)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshal response: %w", err),
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

// AdminSession handles GET /api/v1/admin/sessions
//
//	@Summary	Get the current admin session
//	@Tags		admin
//	@Produce	json
//	@Success	200	{object}	vibe.AdminSessionResponse
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/admin/sessions [get]
func AdminSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := helper.GetAdminUserFromContext(r.Context())
		if !ok || admin.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		response := vibe.AdminSessionResponse{
			Authorized: true,
			User:       admin,
		}
		body, err := json.Marshal(response)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling admin session response: %w", err),
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
//
//	@Summary		Sign out of room administration
//	@Description	Clears the current user's admin session.
//	@Tags		admin
//	@Produce	json
//	@Success	200	{object}	vibe.AdminSessionResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/admin/sessions [delete]
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
				fmt.Errorf("error marshal response: %w", err),
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
//
//	@Summary		Search rooms
//	@Description	Returns filtered, sorted, row-number-paginated room summaries with listener counts, queued song counts, active sources, and room password status.
//	@Tags		admin
//	@Produce	json
//	@Param		q		query		string	false	"Room name search"
//	@Param		sortBy	query		string	false	"Sort field: listeners or songs"
//	@Param		order	query		string	false	"Sort order: asc or desc"
//	@Param		from	query		int		false	"Zero-based first row"
//	@Param		to		query		int		false	"Zero-based last row"
//	@Success	200	{object}	vibe.AdminRoomResult
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/admin/rooms [get]
//
// Deprecated: Use GET /api/v2/admin/rooms. Retained for existing clients.
//
// @Deprecated
// @Description Deprecated: Use GET /api/v2/admin/rooms for the playlist-item contract. This endpoint retains its existing payloads.
func AdminRooms(
	db vibe.AdminRoomV2Searcher,
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

		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if len(query) > adminRoomQueryMaximumLength {
			handleError(
				w,
				fmt.Errorf("error room search query too long"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		sortBy := vibe.AdminRoomSort(r.URL.Query().Get("sortBy"))
		if sortBy == "" {
			sortBy = vibe.AdminRoomSortListeners
		}
		if sortBy != vibe.AdminRoomSortListeners &&
			sortBy != vibe.AdminRoomSortSongs {
			handleError(
				w,
				fmt.Errorf("error invalid room sort"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		order := r.URL.Query().Get("order")
		if order == "" {
			order = adminRoomOrderDescending
		}
		if order != adminRoomOrderAscending &&
			order != adminRoomOrderDescending {
			handleError(
				w,
				fmt.Errorf("error invalid room sort order"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		from := 0
		var err error
		fromValue := r.URL.Query().Get("from")
		if fromValue != "" {
			from, err = strconv.Atoi(fromValue)
			if err != nil || from < 0 {
				handleError(
					w,
					fmt.Errorf("error invalid first room row"),
					http.StatusBadRequest,
					false,
				)
				return
			}
		}

		to := from + adminRoomPageSize - 1
		toValue := r.URL.Query().Get("to")
		if toValue != "" {
			to, err = strconv.Atoi(toValue)
			if err != nil || to < from {
				handleError(
					w,
					fmt.Errorf("error invalid last room row"),
					http.StatusBadRequest,
					false,
				)
				return
			}
		}
		if to-from+1 > adminRoomMaximumPageSize {
			handleError(
				w,
				fmt.Errorf("error room page too large"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		canonicalSort := vibe.AdminRoomSortV2(sortBy)
		if sortBy == vibe.AdminRoomSortSongs {
			canonicalSort = vibe.AdminRoomSortPlaylistItems
		}

		result, err := db.SearchAdminRoomsV2(ctx, vibe.AdminRoomSearchV2{
			Query:      query,
			SortBy:     canonicalSort,
			Descending: order == adminRoomOrderDescending,
			From:       from,
			To:         to,
		})
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching admin rooms: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		legacy := result.ToAdminRoomResult()

		body, err := json.Marshal(legacy)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshal response: %w", err),
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

// AdminSearchUsage handles GET /api/v1/admin/searches/usage
//
//	@Summary	List search usage activity
//	@Tags		admin
//	@Produce	json
//	@Success	200	{object}	vibe.AdminSearchUsage
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/admin/searches/usage [get]
func AdminSearchUsage(
	db vibe.AdminSearchUsageLister,
	cache vibe.CachedAdminSearchUsageFetcherCreator,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		cachedUsage, err := cache.GetCachedAdminSearchUsage(ctx)
		if err != nil {
			log.Printf("error getting cached admin search usage: %v", err)
			cachedUsage = &vibe.CachedAdminSearchUsage{}
		}

		usage := &cachedUsage.Usage
		if cachedUsage.IsEmpty() {
			points, err := db.ListAdminSearchUsage(ctx)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error fetching admin search usage: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			roomPoints, err := db.ListAdminRoomSearchUsage(ctx)
			if err != nil {
				handleError(w, fmt.Errorf("error fetching room search usage: %w", err), http.StatusInternalServerError, true)
				return
			}

			usage = &vibe.AdminSearchUsage{
				RoomPoints:  roomPoints,
				Points:      points,
				GeneratedAt: time.Now().UTC(),
			}
			err = cache.CacheAdminSearchUsage(ctx, *usage)
			if err != nil {
				log.Printf("error caching admin search usage: %v", err)
			}
		}

		body, err := json.Marshal(usage)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling admin search usage: %w", err),
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

// AdminUpdateRoom handles PATCH /api/v1/admin/rooms/{id}
//
//	@Summary		Update a room
//	@Description	Renames a room and/or clears its room admin password, then returns the refreshed room list.
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string						true	"Room ID"
//	@Param		request	body		vibe.AdminUpdateRoomRequest	true	"Room update payload"
//	@Success	200		{array}		vibe.AdminRoomSummary
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	404		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Router		/api/v1/admin/rooms/{id} [patch]
//
// Deprecated: Use PATCH /api/v2/admin/rooms/{id}. Retained for existing clients.
//
// @Deprecated
// @Description Deprecated: Use PATCH /api/v2/admin/rooms/{id} for the playlist-item contract. This endpoint retains its existing payloads.
func AdminUpdateRoom(
	db vibe.AdminRoomV2UpdaterLister,
	notifier vibe.AdminEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]
		if roomID == "" {
			handleError(
				w,
				fmt.Errorf("error room id required"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		var req vibe.AdminUpdateRoomRequest
		err := json.UnmarshalRead(r.Body, &req)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding request body: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		if req.Name != nil {
			name := strings.TrimSpace(*req.Name)
			if name == "" {
				handleError(
					w,
					fmt.Errorf("error room name required"),
					http.StatusBadRequest,
					false,
				)
				return
			}
			req.Name = &name
		}

		updated, err := db.UpdateAdminRoom(ctx, roomID, req)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error updating admin room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if !updated {
			handleError(
				w,
				fmt.Errorf("error room not found"),
				http.StatusNotFound,
				false,
			)
			return
		}

		rooms, err := db.ListAdminRoomsV2(ctx)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching admin rooms: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		legacy := make([]vibe.AdminRoomSummary, len(rooms))
		for index, room := range rooms {
			legacy[index] = *room.ToAdminRoomSummary()
		}

		body, err := json.Marshal(legacy)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling admin rooms: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		err = notifier.NotifyAdminUpdate(ctx, vibe.AdminEvent{
			Type:    vibe.AdminRoomsUpdate,
			Payload: body,
		})
		if err != nil {
			log.Printf("error notifying admin rooms update: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// AdminDeleteRoom handles DELETE /api/v1/admin/rooms/{id}
//
//	@Summary		Delete a room
//	@Description	Deletes the identified room and publishes the refreshed room list to administrators.
//	@Tags		admin
//	@Param		id	path		string	true	"Room ID"
//	@Success	204
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	404	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/admin/rooms/{id} [delete]
//
// Deprecated: Use DELETE /api/v2/admin/rooms/{id}. Retained for existing clients.
//
// @Deprecated
// @Description Deprecated: Use DELETE /api/v2/admin/rooms/{id} for the playlist-item contract. This endpoint retains its existing payloads.
func AdminDeleteRoom(
	db vibe.AdminRoomV2DeleterLister,
	notifier vibe.AdminEventNotifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		roomID := vars["id"]
		if roomID == "" {
			handleError(
				w,
				fmt.Errorf("error room id required"),
				http.StatusBadRequest,
				false,
			)
			return
		}

		deleted, err := db.DeleteAdminRoom(ctx, roomID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error deleting admin room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		if !deleted {
			handleError(
				w,
				fmt.Errorf("error room not found"),
				http.StatusNotFound,
				false,
			)
			return
		}

		rooms, err := db.ListAdminRoomsV2(ctx)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching admin rooms: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		legacy := make([]vibe.AdminRoomSummary, len(rooms))
		for index, room := range rooms {
			legacy[index] = *room.ToAdminRoomSummary()
		}

		body, err := json.Marshal(legacy)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling admin rooms: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		err = notifier.NotifyAdminUpdate(ctx, vibe.AdminEvent{
			Type:    vibe.AdminRoomsUpdate,
			Payload: body,
		})
		if err != nil {
			log.Printf("error notifying admin rooms update: %v", err)
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// AdminEvents handles GET /api/v1/admin/events (SSE)
//
//	@Summary		Subscribe to room administration events
//
// @Description Frames contain an `event` name and a JSON `data` payload, followed by a blank line. `connected` carries {"time":1700000000000}; `admin_rooms_update` carries an array of AdminRoomSummary objects. Heartbeats are SSE comments.
// @Description This is an ongoing text/event-stream response, not a single JSON document.
//
//	@Description	Streams the initial room list and subsequent room summary updates to authenticated administrators.
//	@Tags		admin
//	@Produce	text/event-stream
//	@Success	200	{string}	string "SSE frames with event-specific JSON data; see the stream description"
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/admin/events [get]
//
// Deprecated: Use GET /api/v2/admin/events. Retained for existing clients.
//
// @Deprecated
// @Description Deprecated: Use GET /api/v2/admin/events for the playlist-item contract. This endpoint retains its existing payloads.
func AdminEvents(
	subscriber vibe.Subscriber,
	db vibe.AdminRoomV2Lister,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		container, err := subscriber.Subscribe(ctx, "admin")
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error subscribing to admin events: %w", err),
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
				fmt.Errorf("error streaming not supported"),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		fmt.Fprintf(w, "event: connected\ndata: {\"time\": %d}\n\n", time.Now().UnixMilli())
		flusher.Flush()

		rooms, err := db.ListAdminRoomsV2(ctx)
		if err == nil {
			legacy := make([]vibe.AdminRoomSummary, len(rooms))
			for index, room := range rooms {
				legacy[index] = *room.ToAdminRoomSummary()
			}

			payload, err := json.Marshal(legacy)
			if err == nil {
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

				canonical, err := vibe.ParseAdminEvent(data)
				if err != nil {
					continue
				}

				event, err := canonical.ToAdminEvent()
				if err != nil {
					continue
				}

				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, event.Payload)
				flusher.Flush()
			}
		}
	}
}

const adminSessionDuration = 24 * time.Hour

const adminRoomPageSize = 10

const adminRoomMaximumPageSize = 50

const adminRoomQueryMaximumLength = 100

const adminRoomOrderAscending = "asc"

const adminRoomOrderDescending = "desc"

// AdminUsers handles GET /api/v1/admin/users
//
//	@Summary	List admin users
//	@Tags		admin
//	@Produce	json
//	@Success	200	{array}		vibe.AdminUser
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/admin/users [get]
func AdminUsers(lister vibe.AdminUserLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := lister.ListAdminUsers(r.Context())
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error listing admin users in AdminUsers handler: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(users)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling admin users response: %w", err),
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

// AdminCreateUser handles POST /api/v1/admin/users
//
//	@Summary	Create an admin user
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		request	body		vibe.AdminCreateUserRequest	true	"Admin user"
//	@Success	201		{object}	vibe.AdminUser
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	401		{object}	vibe.ErrorResponse
//	@Failure	409		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Router		/api/v1/admin/users [post]
func AdminCreateUser(
	creator vibe.AdminUserCreator,
	passwordPepper string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request vibe.AdminCreateUserRequest
		err := json.UnmarshalRead(r.Body, &request)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding admin user request: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		username, err := vibe.NormalizeAdminUsername(request.Username)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error validating admin username: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}

		err = vibe.ValidateAdminPassword(request.Password)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error validating admin password: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}

		passwordHash, err := helper.GenerateAdminPasswordHash(
			request.Password,
			passwordPepper,
		)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error hashing admin password: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		user, err := creator.CreateAdminUser(r.Context(), vibe.AdminUser{
			ID:           uuid.NewString(),
			Username:     username,
			PasswordHash: passwordHash,
		})
		if err != nil {
			var usernameError internalerror.ErrAdminUsernameUnavailable
			if errors.As(err, &usernameError) {
				handleError(
					w,
					fmt.Errorf("error admin username unavailable: %w", err),
					http.StatusConflict,
					false,
				)
				return
			}

			handleError(
				w,
				fmt.Errorf("error creating admin user: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(user)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling admin user response: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(body)
	}
}

// AdminUpdateUser handles PATCH /api/v1/admin/users/{id}
//
//	@Summary	Reset another admin user's password
//	@Tags		admin
//	@Accept		json
//	@Param		id		path	string						true	"Admin user ID"
//	@Param		request	body	vibe.AdminUpdateUserRequest	true	"New password"
//	@Success	204
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	403	{object}	vibe.ErrorResponse
//	@Failure	404	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/admin/users/{id} [patch]
func AdminUpdateUser(
	updater vibe.AdminUserPasswordUpdater,
	passwordPepper string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := helper.GetAdminUserFromContext(r.Context())
		if !ok || admin.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		adminID := mux.Vars(r)["id"]
		if adminID == "" {
			handleError(
				w,
				fmt.Errorf("error admin user id required"),
				http.StatusBadRequest,
				false,
			)
			return
		}
		if adminID == admin.ID {
			handleError(
				w,
				fmt.Errorf("error cannot reset your own password"),
				http.StatusForbidden,
				false,
			)
			return
		}

		var request vibe.AdminUpdateUserRequest
		err := json.UnmarshalRead(r.Body, &request)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding admin password request: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		err = vibe.ValidateAdminPassword(request.Password)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error validating admin password: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}

		passwordHash, err := helper.GenerateAdminPasswordHash(
			request.Password,
			passwordPepper,
		)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error hashing admin password: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		updated, err := updater.UpdateAdminUserPassword(
			r.Context(),
			adminID,
			passwordHash,
		)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error updating admin password: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if !updated {
			handleError(
				w,
				fmt.Errorf("error admin user not found"),
				http.StatusNotFound,
				false,
			)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// AdminDeleteUser handles DELETE /api/v1/admin/users/{id}
//
//	@Summary	Delete another admin user
//	@Tags		admin
//	@Param		id	path	string	true	"Admin user ID"
//	@Success	204
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	403	{object}	vibe.ErrorResponse
//	@Failure	404	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/admin/users/{id} [delete]
func AdminDeleteUser(deleter vibe.AdminUserDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := helper.GetAdminUserFromContext(r.Context())
		if !ok || admin.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		adminID := mux.Vars(r)["id"]
		if adminID == "" {
			handleError(
				w,
				fmt.Errorf("error admin user id required"),
				http.StatusBadRequest,
				false,
			)
			return
		}
		if adminID == admin.ID {
			handleError(
				w,
				fmt.Errorf("error cannot delete your own admin user"),
				http.StatusForbidden,
				false,
			)
			return
		}

		deleted, err := deleter.DeleteAdminUser(r.Context(), adminID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error deleting admin user: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if !deleted {
			handleError(
				w,
				fmt.Errorf("error admin user not found"),
				http.StatusNotFound,
				false,
			)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// ReviewAdminRooms handles scheduled admin room updates.
type ReviewAdminRooms struct {
	DB     vibe.AdminRoomV2Lister
	Events vibe.AdminEventV2Notifier
}

// Handle fetches admin rooms and broadcasts the update.
func (h *ReviewAdminRooms) Handle(ctx context.Context, _ []byte) error {
	rooms, err := h.DB.ListAdminRoomsV2(ctx)
	if err != nil {
		return fmt.Errorf("error listing admin rooms: %w", err)
	}

	payload, err := json.Marshal(rooms)
	if err != nil {
		return fmt.Errorf("error marshaling admin rooms: %w", err)
	}

	err = h.Events.NotifyAdminUpdateV2(ctx, vibe.AdminEventV2{
		Type:    vibe.AdminRoomsUpdate,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("error notifying admin rooms update: %w", err)
	}

	return nil
}
