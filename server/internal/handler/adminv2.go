package handler

import (
	"encoding/json/v2"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// AdminRoomsV2 handles GET /api/v2/admin/rooms
//
//	@Summary		Search rooms
//	@Description	Returns filtered, sorted, row-number-paginated room summaries with listener counts, queued playlist item counts, active sources, public visibility, and room password status.
//	@Tags		admin
//	@Produce	json
//	@Param		q		query		string	false	"Room name search"
//	@Param		sortBy	query		string	false	"Sort field: listeners or playlistItems"
//	@Param		order	query		string	false	"Sort order: asc or desc"
//	@Param		from	query		int		false	"Zero-based first row"
//	@Param		to		query		int		false	"Zero-based last row"
//	@Success	200	{object}	vibe.AdminRoomResultV2
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/admin/rooms [get]
func AdminRoomsV2(
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
			log.Printf("AdminRoomsV2: request missing admin_session cookie")
		}
		if hasAdminCookie {
			log.Printf("AdminRoomsV2: request has admin_session cookie")
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

		sortBy := vibe.AdminRoomSortV2(r.URL.Query().Get("sortBy"))
		if sortBy == "" {
			sortBy = vibe.AdminRoomSortListenersV2
		}
		if sortBy != vibe.AdminRoomSortListenersV2 &&
			sortBy != vibe.AdminRoomSortPlaylistItems {
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

		result, err := db.SearchAdminRoomsV2(ctx, vibe.AdminRoomSearchV2{
			Query:      query,
			SortBy:     sortBy,
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

		body, err := json.Marshal(result)
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

// AdminUpdateRoomV2 handles PATCH /api/v2/admin/rooms/{id}
//
//	@Summary		Update a room
//	@Description	Renames a room and/or clears its room admin password, then returns the refreshed room list.
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string						true	"Room ID"
//	@Param		request	body		vibe.AdminUpdateRoomRequest	true	"Room update payload"
//	@Success	200		{array}		vibe.AdminRoomSummaryV2
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	404		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Router		/api/v2/admin/rooms/{id} [patch]
func AdminUpdateRoomV2(
	db vibe.AdminRoomV2UpdaterLister,
	notifier vibe.AdminEventV2Notifier,
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

		body, err := json.Marshal(rooms)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling admin rooms: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		err = notifier.NotifyAdminUpdateV2(ctx, vibe.AdminEventV2{
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

// AdminDeleteRoomV2 handles DELETE /api/v2/admin/rooms/{id}
//
//	@Summary		Delete a room
//	@Description	Deletes the identified room and publishes the refreshed room list to administrators.
//	@Tags		admin
//	@Param		id	path		string	true	"Room ID"
//	@Success	204
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	404	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/admin/rooms/{id} [delete]
func AdminDeleteRoomV2(
	db vibe.AdminRoomV2DeleterLister,
	notifier vibe.AdminEventV2Notifier,
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

		body, err := json.Marshal(rooms)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling admin rooms: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		err = notifier.NotifyAdminUpdateV2(ctx, vibe.AdminEventV2{
			Type:    vibe.AdminRoomsUpdate,
			Payload: body,
		})
		if err != nil {
			log.Printf("error notifying admin rooms update: %v", err)
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// AdminEventsV2 handles GET /api/v2/admin/events (SSE)
//
//	@Summary		Subscribe to room administration events
//
// @Description Frames contain an `event` name and a JSON `data` payload, followed by a blank line. `connected` carries {"time":1700000000000}; `admin_rooms_update` carries an array of AdminRoomSummaryV2 objects. Heartbeats are SSE comments.
// @Description This is an ongoing text/event-stream response, not a single JSON document.
//
//	@Description	Streams the initial room list and subsequent room summary updates to authenticated administrators.
//	@Tags		admin
//	@Produce	text/event-stream
//	@Success	200	{string}	string "SSE frames with event-specific JSON data; see the stream description"
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/admin/events [get]
func AdminEventsV2(
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
			payload, err := json.Marshal(rooms)
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

				event, err := vibe.ParseAdminEvent(data)
				if err != nil {
					continue
				}

				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, event.Payload)
				flusher.Flush()
			}
		}
	}
}
