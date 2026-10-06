package middleware

import (
	"net/http"

	"github.com/gorilla/handlers"
)

// CompressionMiddleware compresses supported responses when the client accepts gzip.
func CompressionMiddleware(next http.Handler) http.Handler {
	handler := handlers.CompressHandler(next)

	return handler
}
