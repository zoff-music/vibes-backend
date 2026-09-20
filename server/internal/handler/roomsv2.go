package handler

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/zoff-music/vibes-backend/vibe"
)

const publicRoomPageSize = 10

const publicRoomMaximumPageSize = 100

const publicRoomMaximumQueryLength = 100

// GetPublicRoomsV2 handles GET /api/v2/rooms/public.
//
//	@Summary		Browse public rooms
//	@Description	Returns public rooms ordered by listener count, song count, then ID, all descending. Ranges are zero-based and inclusive. Only rooms with protected admin controls are listed.
//	@Tags			rooms
//	@Produce		json
//	@Param			q		query		string	false	"Case-insensitive room name search (literal substring, up to 100 characters)"
//	@Param			live	query		boolean	false	"Only rooms with active listeners" default(false)
//	@Param			from	query		int		false	"First row, zero-based" minimum(0) default(0)
//	@Param			to		query		int		false	"Last row, inclusive. Defaults to from + 9; at most 100 rooms per page" minimum(0)
//	@Success		200		{object}	vibe.PublicRoomResult
//	@Failure		400		{object}	vibe.ErrorResponse
//	@Failure		500		{object}	vibe.ErrorResponse
//	@Router			/api/v2/rooms/public [get]
func GetPublicRoomsV2(db vibe.PublicRoomsSearcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if !utf8.ValidString(query) || strings.ContainsRune(query, '\x00') {
			handleError(w, fmt.Errorf("error invalid room search text"), http.StatusBadRequest, false)
			return
		}

		if utf8.RuneCountInString(query) > publicRoomMaximumQueryLength {
			handleError(w, fmt.Errorf("error room search is too long"), http.StatusBadRequest, false)
			return
		}

		live := r.URL.Query().Get("live")
		if live != "" && live != "true" && live != "false" {
			handleError(w, fmt.Errorf("error invalid live room filter"), http.StatusBadRequest, false)
			return
		}

		var from int64
		var err error
		fromValue := r.URL.Query().Get("from")
		if fromValue != "" {
			from, err = strconv.ParseInt(fromValue, 10, 32)
			if err != nil {
				handleError(w, fmt.Errorf("error parsing first room row: %w", err), http.StatusBadRequest, false)
				return
			}

			if from < 0 {
				handleError(w, fmt.Errorf("error invalid first room row"), http.StatusBadRequest, false)
				return
			}
		}

		to := from + publicRoomPageSize - 1
		toValue := r.URL.Query().Get("to")
		if toValue != "" {
			to, err = strconv.ParseInt(toValue, 10, 32)
			if err != nil {
				handleError(w, fmt.Errorf("error parsing last room row: %w", err), http.StatusBadRequest, false)
				return
			}

			if to < from {
				handleError(w, fmt.Errorf("error invalid last room row"), http.StatusBadRequest, false)
				return
			}
		}

		if to-from+1 > publicRoomMaximumPageSize {
			handleError(w, fmt.Errorf("error room page is too large"), http.StatusBadRequest, false)
			return
		}

		result, err := db.SearchPublicRooms(ctx, vibe.PublicRoomSearch{
			Query: query,
			Live:  live == "true",
			From:  int(from),
			To:    int(to),
		})
		if err != nil {
			handleError(w, fmt.Errorf("error searching public rooms: %w", err), http.StatusInternalServerError, true)
			return
		}

		body, err := json.Marshal(result)
		if err != nil {
			handleError(w, fmt.Errorf("error marshaling public rooms: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}
