package handler

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
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

		flagged := r.URL.Query().Get("flagged") == "true"

		result, err := db.SearchAdminRoomsV2(ctx, vibe.AdminRoomSearchV2{
			Flagged:    flagged,
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
	notifier vibe.RoomEventV3Notifier,
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
		err := jsonv2.UnmarshalRead(r.Body, &req)
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

		room, err := db.GetRoomV2(ctx, roomID, "")
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error fetching updated room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		roomBody, err := json.Marshal(room)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error encoding updated room: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		err = notifier.NotifyRoomUpdateV3(ctx, roomID, vibe.RoomEventV3{
			Type:    vibe.SettingsUpdate,
			Payload: roomBody,
		})
		if err != nil {
			log.Printf("error notifying updated room: %v", err)
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

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// AdminDeleteRoomV2 handles DELETE /api/v2/admin/rooms/{id}
//
//	@Summary		Delete a room
//	@Description	Deletes the identified room and notifies administrators of the change.
//	@Tags		admin
//	@Param		id	path		string	true	"Room ID"
//	@Success	204
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	404	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v2/admin/rooms/{id} [delete]
func AdminDeleteRoomV2(
	db vibe.AdminRoomDeleter,
	notifier vibe.AdminRoomChangeNotifier,
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

		if deleted {
			err = notifier.NotifyAdminRoomChange(ctx, roomID)
			if err != nil {
				log.Printf("error notifying deleted admin room: %v", err)
			}
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

		w.WriteHeader(http.StatusNoContent)
	}
}
