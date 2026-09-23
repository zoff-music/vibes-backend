package handler

import (
	"fmt"
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/swaggo/swag"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
)

// Documentation serves the branded UI, public specification, and an explicit asset allowlist.
// Never delegate arbitrary paths to http-swagger: its doc.json route serves the full spec.
func Documentation() http.HandlerFunc {
	assets := httpSwagger.Handler()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Add("Vary", "Cookie")
		switch r.URL.Path {
		case "/api/swagger/", "/api/swagger/index.html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(documentationHTML))
		case "/api/swagger/theme.css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
			_, _ = w.Write([]byte(documentationCSS))
		case "/api/swagger/theme.js":
			w.Header().Set("Content-Type", "application/javascript")
			_, _ = w.Write([]byte(documentationJS))
		case "/api/swagger/doc.json":
			document, err := swag.ReadDoc()
			if err != nil {
				handleError(w, fmt.Errorf("error reading documentation: %w", err), http.StatusInternalServerError, true)
				return
			}
			body, err := helper.PublicDocumentation(document)
			if err != nil {
				handleError(w, fmt.Errorf("error filtering documentation: %w", err), http.StatusInternalServerError, true)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(body)
		case "/api/swagger/swagger-ui.css", "/api/swagger/swagger-ui-bundle.js", "/api/swagger/swagger-ui-standalone-preset.js":
			// http-swagger parses RequestURI, including its query string. Only pass the
			// allowlisted path so a query cannot dispatch to its unfiltered doc.json.
			assetRequest := r.Clone(r.Context())
			assetRequest.RequestURI = r.URL.Path
			assets.ServeHTTP(w, assetRequest)
		default:
			http.NotFound(w, r)
		}
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
			handleError(w, fmt.Errorf("error reading admin documentation: %w", err), http.StatusInternalServerError, true)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(document))
	}
}
