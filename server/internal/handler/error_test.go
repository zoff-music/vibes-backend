package handler

import (
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zoff-music/vibes-backend/client"
)

func TestHandleErrorDoesNotExposeInternalDetails(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		status          int
		expectedCode    string
		expectedMessage string
	}{
		{
			name:         "SQL error",
			err:          fmt.Errorf("error selecting secret_table: %w", sql.ErrNoRows),
			status:       500,
			expectedCode: "something went wrong",
		},
		{
			name:         "joined dependency errors",
			err:          errors.Join(sql.ErrNoRows, fmt.Errorf("error redis password=secret")),
			status:       500,
			expectedCode: "something went wrong",
		},
		{
			name:         "client failure also masks internal cause",
			err:          fmt.Errorf("error parsing secret input"),
			status:       400,
			expectedCode: "something went wrong",
		},
		{
			name: "wrapper without propagation stays private",
			err: client.ErrorCodeWrapper{
				ResponseBody: client.ErrorCodeResponseBody{
					Error:   "secret SQL error",
					Message: "secret database detail",
				},
				Err:        sql.ErrNoRows,
				StatusCode: 400,
			},
			status:       500,
			expectedCode: "something went wrong",
		},
		{
			name: "invalid public status fails closed",
			err: client.ErrorCodeWrapper{
				ResponseBody: client.ErrorCodeResponseBody{
					Error:     "song_vote_already_exists",
					Message:   "Your vote is already counted for this song.",
					Propagate: true,
				},
				Err:        sql.ErrNoRows,
				StatusCode: 200,
			},
			status:       500,
			expectedCode: "something went wrong",
		},
		{
			name: "wrapped public error preserves safe code and message",
			err: fmt.Errorf("error secret database context: %w", client.ErrorCodeWrapper{
				ResponseBody: client.ErrorCodeResponseBody{
					Error:     "song_vote_already_exists",
					Message:   "Your vote is already counted for this song.",
					Propagate: true,
				},
				Err:        sql.ErrNoRows,
				StatusCode: 409,
			}),
			status:          409,
			expectedCode:    "song_vote_already_exists",
			expectedMessage: "Your vote is already counted for this song.",
		},
		{
			name: "public numeric limit",
			err: client.ErrorCodeWrapper{
				ResponseBody: client.ErrorCodeResponseBody{
					Error:     "room_generation_song_limit",
					Message:   "Playlists can only be generated when the room has 20 songs or fewer.",
					Propagate: true,
				},
				Err:        sql.ErrNoRows,
				StatusCode: 409,
			},
			status:          409,
			expectedCode:    "room_generation_song_limit",
			expectedMessage: "Playlists can only be generated when the room has 20 songs or fewer.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			handleError(recorder, tt.err, tt.status, false)

			if recorder.Code != tt.status {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.status)
			}

			if recorder.Header().Get("Content-Type") != "application/json" {
				t.Fatal("missing JSON content type")
			}

			for _, internal := range []string{"secret", "sql:", "no rows", "password="} {
				if strings.Contains(recorder.Body.String(), internal) {
					t.Fatalf("exposed internal detail: %s", recorder.Body.String())
				}
			}

			var response client.ErrorCodeResponseBody
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			if err != nil {
				t.Fatal(err)
			}

			if response.Error != tt.expectedCode || response.Message != tt.expectedMessage {
				t.Fatalf("unexpected response: %+v", response)
			}

			if tt.expectedMessage != "" {
				if response.Namespace != "vibes-backend" || !response.Propagate || recorder.Header().Get("X-preserve-error") != "1" {
					t.Fatal("public error contract changed")
				}
				return
			}

			if recorder.Header().Get("X-preserve-error") != "" || response.Propagate {
				t.Fatal("internal error marked public")
			}
		})
	}
}
