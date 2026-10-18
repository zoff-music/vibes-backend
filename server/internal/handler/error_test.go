package handler

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleErrorCancellation(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		logged bool
	}{
		{name: "canceled", err: context.Canceled, status: 499},
		{name: "wrapped cancellation", err: fmt.Errorf("error fetching stats: %w", context.Canceled), status: 499},
		{name: "deadline remains a failure", err: context.DeadlineExceeded, status: 500, logged: true},
		{name: "other failure remains visible", err: fmt.Errorf("error fetching stats"), status: 500, logged: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			previous := log.Writer()
			log.SetOutput(&output)
			t.Cleanup(func() { log.SetOutput(previous) })
			response := httptest.NewRecorder()

			handleError(response, tt.err, http.StatusInternalServerError, true)

			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d", response.Code, tt.status)
			}
			if (output.Len() > 0) != tt.logged {
				t.Fatalf("logged = %v, want %v", output.Len() > 0, tt.logged)
			}
			if !tt.logged && response.Body.Len() != 0 {
				t.Fatal("cancellation wrote an error body")
			}
		})
	}
}
