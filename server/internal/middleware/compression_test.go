package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompressionMiddleware(t *testing.T) {
	tests := []struct {
		name            string
		acceptEncoding  string
		contentEncoding string
	}{
		{
			name:            "compresses when gzip is accepted",
			acceptEncoding:  "gzip",
			contentEncoding: "gzip",
		},
		{
			name:            "leaves response uncompressed without gzip",
			acceptEncoding:  "identity",
			contentEncoding: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const responseBody = "zoff response compression"

			supportsFlushing := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, supportsFlushing = w.(http.Flusher)
				_, err := w.Write([]byte(responseBody))
				if err != nil {
					t.Fatalf("error writing response: %v", err)
				}
			})
			handler := CompressionMiddleware(next)
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set("Accept-Encoding", tt.acceptEncoding)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if !supportsFlushing {
				t.Fatal("expected response writer to preserve streaming flush support")
			}
			if recorder.Header().Get("Content-Encoding") != tt.contentEncoding {
				t.Fatalf("expected Content-Encoding %q, got %q", tt.contentEncoding, recorder.Header().Get("Content-Encoding"))
			}

			var bodyReader io.Reader = recorder.Body
			if tt.contentEncoding == "gzip" {
				gzipReader, err := gzip.NewReader(recorder.Body)
				if err != nil {
					t.Fatalf("error creating gzip reader: %v", err)
				}
				defer gzipReader.Close()
				bodyReader = gzipReader
			}

			body, err := io.ReadAll(bodyReader)
			if err != nil {
				t.Fatalf("error reading response body: %v", err)
			}
			if strings.TrimSpace(string(body)) != responseBody {
				t.Fatalf("expected response body %q, got %q", responseBody, string(body))
			}
		})
	}
}
