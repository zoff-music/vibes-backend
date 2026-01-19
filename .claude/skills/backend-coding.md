# Backend Coding

You are working on the Go backend for Vibes.

**IMPORTANT: Read `backend/AGENTS.md` for the full, non-negotiable coding rules.**

## Quick Reference

### 1. Handler Pattern
- Return `http.HandlerFunc`.
- Inject dependencies as interfaces.
- Use `mux.Vars(r)` for path params.
- Always use `handleError(w, code, err, log)` for errors.

```go
func GetRoom(rf vibe.RoomFetcher) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // ...
    }
}
```

### 2. Database Pattern
- **NO Transactions**. Atomic queries only.
- Use Prepared Statements (prepare in `prepareXStmt`, execute in `X`).
- 1:1 mapping between method `X` and statement `XStatement`.

### 3. Error Handling
- **NO inline error assignment**.
- Wrap ALL errors with `fmt.Errorf("doing X: %w", err)`.
- Use `internalerror` package for control flow errors if needed.

### 4. File Structure
- `server/internal/handler/`: HTTP Handlers.
- `client/`: All external integrations (DB, Music APIs).
- `vibe/`: Pure domain types and interfaces.
