# Add Backend Handler

Create a new HTTP handler following project conventions.

## Requirements

Provide:
- Handler name (e.g., "GetRoom", "CreateSong")
- HTTP method and path
- Dependencies needed (interfaces from vibe/vibe.go)

## Template

File: `backend/server/internal/handler/{resource}.go`

```go
func {HandlerName}(
    {dep} vibe.{Interface},
) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // Parse path params (gorilla/mux)
        vars := mux.Vars(r)
        id := vars["id"]

        // Call dependency
        result, err := {dep}.{Method}(ctx, id)
        if err != nil {
            handleError(w, http.StatusInternalServerError,
                fmt.Errorf("error {action}: %w", err), true)
            return
        }

        // Respond
        response, err := json.Marshal(result)
        if err != nil {
            handleError(w, http.StatusInternalServerError,
                fmt.Errorf("error marshaling response: %w", err), true)
            return
        }

        w.Header().Add("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write(response)
    }
}
```

## Checklist

- [ ] Handler in `server/internal/handler/`
- [ ] Interface in `vibe/vibe.go` (if new)
- [ ] Database method in `client/database/` (if needed)
- [ ] Route added in `server/router.go`
- [ ] No tracing in handler (middleware handles it)
- [ ] Errors wrapped with context
