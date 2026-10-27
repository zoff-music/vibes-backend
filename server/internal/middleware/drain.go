package middleware

import (
	"net/http"
	"sync"
)

// DrainMiddleware tracks handlers until their deferred cleanup has finished.
type DrainMiddleware struct {
	mutex    sync.Mutex
	running  sync.WaitGroup
	draining bool
}

func (m *DrainMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mutex.Lock()
		if m.draining {
			m.mutex.Unlock()
			http.Error(w, "server shutting down", http.StatusServiceUnavailable)
			return
		}

		m.running.Add(1)
		m.mutex.Unlock()
		defer m.running.Done()

		next.ServeHTTP(w, r)
	})
}

func (m *DrainMiddleware) Wait() {
	// Close admission before waiting so late requests cannot race WaitGroup.Add.
	m.mutex.Lock()
	m.draining = true
	m.mutex.Unlock()

	m.running.Wait()
}
