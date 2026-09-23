package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/swaggo/swag"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
)

// Documentation serves the static UI, public specification, and Swagger assets.
// Never delegate arbitrary paths to http-swagger: its doc.json route serves the full spec.
func Documentation(directory string) http.HandlerFunc {
	assets := httpSwagger.Handler()

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Add("Vary", "Cookie")

		var filename string
		var contentType string

		switch r.URL.Path {
		case "/api/swagger/", "/api/swagger/index.html":
			filename = "index.html"
			contentType = "text/html; charset=utf-8"

		case "/api/swagger/theme.css":
			filename = "theme.css"
			contentType = "text/css; charset=utf-8"

		case "/api/swagger/theme.js":
			filename = "theme.js"
			contentType = "application/javascript"

		case "/api/swagger/doc.json":
			document, err := swag.ReadDoc()
			if err != nil {
				handleError(w, fmt.Errorf("error reading documentation in Documentation: %w", err), http.StatusInternalServerError, true)
				return
			}

			body, err := helper.PublicDocumentation(document)
			if err != nil {
				handleError(w, fmt.Errorf("error filtering documentation in Documentation: %w", err), http.StatusInternalServerError, true)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(body)
			return

		case "/api/swagger/swagger-ui.css", "/api/swagger/swagger-ui-bundle.js", "/api/swagger/swagger-ui-standalone-preset.js":
			// http-swagger parses RequestURI, including its query string. Only pass the
			// allowlisted path so a query cannot dispatch to its unfiltered doc.json.
			assetRequest := r.Clone(r.Context())
			assetRequest.RequestURI = r.URL.Path

			assets.ServeHTTP(w, assetRequest)
			return

		default:
			http.NotFound(w, r)
			return
		}

		body, err := os.ReadFile(filepath.Join(directory, filename))
		if err != nil {
			handleError(w, fmt.Errorf("error reading static file in Documentation: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(body)
	}
}

// AdminDocumentation is routed through the same session and admin middleware as admin APIs.
func AdminDocumentation() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Add("Vary", "Cookie")

		_, ok := helper.GetAdminUserFromContext(r.Context())
		if !ok {
			handleError(w, fmt.Errorf("error unauthorized documentation access"), http.StatusUnauthorized, false)
			return
		}

		document, err := swag.ReadDoc()
		if err != nil {
			handleError(w, fmt.Errorf("error reading documentation in AdminDocumentation: %w", err), http.StatusInternalServerError, true)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(document))
	}
}
