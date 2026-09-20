package middleware

import "net/http"

type RequestBodyMiddleware struct {
	MaxBytes int
}

func (m RequestBodyMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > int64(m.MaxBytes) {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, int64(m.MaxBytes))
		next.ServeHTTP(w, r)
	})
}
