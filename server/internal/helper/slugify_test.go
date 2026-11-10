package helper

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "kanji", input: "東京 音楽", want: "u6771-u4eac-u20-u97f3-u697d"},
		{name: "mixed scripts", input: "Tokyo 東京 ２０２６", want: "u54-u6f-u6b-u79-u6f-u20-u6771-u4eac-u20-uff12-uff10-uff12-uff16"},
		{name: "kana", input: "かな カタカナ", want: "u304b-u306a-u20-u30ab-u30bf-u30ab-u30ca"},
		{name: "combining marks", input: "Café 音楽", want: "u43-u61-u66-u65-u301-u20-u97f3-u697d"},
		{name: "trimmed unicode", input: "  東京 音楽  ", want: "u6771-u4eac-u20-u97f3-u697d"},
		{name: "ascii compatibility", input: " --My  Room---42! ", want: "my-room-42"},
		{name: "unsafe URL characters", input: "東京/?#% 音楽", want: "u6771-u4eac-u2f-u3f-u23-u25-u20-u97f3-u697d"},
		{name: "emoji display name", input: "🎵", want: "u1f3b5"},
		{name: "only ASCII symbols", input: "/?#%", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Slugify(tt.input)
			if got != tt.want {
				t.Fatalf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
			if Slugify(got) != got {
				t.Errorf("Slugify(%q) is not stable", got)
			}
			if strings.ContainsAny(got, "/?#%") {
				t.Errorf("unsafe URL ID: %q", got)
			}
			if got == "" {
				return
			}
			router := mux.NewRouter()
			router.HandleFunc("/api/v2/rooms/{id}", func(w http.ResponseWriter, r *http.Request) {
				if mux.Vars(r)["id"] != tt.want {
					t.Errorf("room ID = %q, want %q", mux.Vars(r)["id"], tt.want)
				}
				w.WriteHeader(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodGet, "/api/v2/rooms/"+got, nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusNoContent {
				t.Errorf("status = %d, want %d", recorder.Code, http.StatusNoContent)
			}
		})
	}
}

func TestRoomIDEncodingSafety(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "path traversal", input: "東京/../../admin"},
		{name: "query and fragment", input: "東京?admin=true#settings"},
		{name: "header characters", input: "東京\r\nLocation: https://example.com"},
		{name: "encoded slash", input: "東京%2f%2e%2e"},
		{name: "HTML", input: "東京<script>alert(1)</script>"},
		{name: "maximum length", input: strings.Repeat("\U0010ffff", 100)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := Slugify(tt.input)
			for _, character := range id {
				if !(character >= 'a' && character <= 'z') && !(character >= '0' && character <= '9') && character != '-' {
					t.Fatalf("unsafe ID character: %q", character)
				}
			}
			if len(id) > 799 {
				t.Fatalf("encoded ID length = %d, exceeds bound", len(id))
			}
		})
	}
}
