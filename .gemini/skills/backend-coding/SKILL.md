---
name: Backend Coding
description: Guide for writing backend Go code (handlers, clients, DB)
---

# Backend Coding

You are working on the Go backend for Vibez.

**IMPORTANT: Read `backend/AGENTS.md` for the full, non-negotiable coding rules.**

## Quick Reference

### 1. Handler Pattern
- Return `http.HandlerFunc`
- Inject dependencies as interfaces
- Use `mux.Vars(r)` for path params
- Always use `handleError(w, code, err, log)` for errors
- NO tracing in handlers (middleware handles it)

```go
func GetRoom(rf vibe.RoomFetcher) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        vars := mux.Vars(r)
        id := vars["id"]
        
        room, err := rf.GetRoom(ctx, id)
        if err != nil {
            handleError(w, http.StatusInternalServerError,
                fmt.Errorf("error getting room: %w", err), true)
            return
        }
        // ...
    }
}
```

### 2. Database Pattern
- **NO Transactions** - Atomic queries only
- Use Prepared Statements (prepare in `prepareXStmt`, execute in `X`)
- 1:1 mapping between method `X` and statement `XStatement`
- Handle `sql.ErrNoRows` by returning empty struct + nil

### 3. Error Handling
- **NO inline error assignment** (`if err := ...; err != nil {}`)
- Wrap ALL errors with `fmt.Errorf("error doing X: %w", err)`
- Use `internalerror` package for control flow errors if needed
- Never return `nil, nil` from `(*T, error)` functions

### 4. File Structure
- `server/internal/handler/`: HTTP Handlers
- `client/`: All external integrations (DB, YouTube, Spotify, SoundCloud, PubSub)
- `vibe/`: Pure domain types and interfaces (CRITICAL - ALL types here)
- `config/`: Configuration management
- `monitoring/`: OpenTelemetry tracing and metrics

### 5. Tracing
Required for all methods taking `context.Context` **except HTTP handlers**:

```go
span, ctx := opentracing.StartSpanFromContext(ctx, "MethodName")
defer span.Finish()
```
