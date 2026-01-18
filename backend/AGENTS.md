# Backend Coding Rules

Non-negotiable conventions. Follow strictly.

## Critical Rules

- **No service/repository patterns** - no `service.*` or `repository.*` layers
- **No `New*` constructors** - prefer struct literals
- **No inline error assignment** - never `if err := ...; err != nil {}`
- **Never return `nil, nil`** from `(*T, error)` functions
- **All errors wrapped** with context: `fmt.Errorf("error doing X: %w", err)`
- **All HTTP in clients** - packages under `client/`, consumed via interfaces
- **Domain types in `vibe/vibe.go`** - not in handlers
- **Don't touch `monitoring/`**

## File Layout

```
client/                    # External integrations
├── database/database.go   # SQLite client
├── youtube/youtube.go     # YouTube API
└── internalpubsub/        # SSE broadcasting

server/
├── server.go              # Client init, route wiring
├── router.go              # Route definitions
└── internal/handler/      # HTTP handlers

vibe/vibe.go               # ALL domain types & interfaces
```

## Handler Pattern

Handlers are functions returning `http.HandlerFunc`. Dependencies injected as interfaces.

```go
func GetRoom(rf vibe.RoomFetcher) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        // NO tracing here - middleware handles it

        // Path params via gorilla/mux
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

## Database Pattern

**No transactions** - each query must be atomic (single statement).

Prepared statements with 1:1 naming:
- `prepareXStmt()` → prepares `XStatement`
- `X()` → executes `XStatement`

```go
func (c *Client) prepareGetRoomStmt() error {
    stmt, err := c.DB.Prepare(`SELECT ... FROM rooms WHERE id = ?`)
    if err != nil {
        return fmt.Errorf("error preparing GetRoomStatement: %w", err)
    }
    c.GetRoomStatement = stmt
    return nil
}

func (c *Client) GetRoom(ctx context.Context, id string) (*vibe.Room, error) {
    span, ctx := opentracing.StartSpanFromContext(ctx, "GetRoom")
    defer span.Finish()

    cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    row := c.GetRoomStatement.QueryRowContext(cctx, id)
    // ...
}
```

**sql.ErrNoRows handling:** Return empty struct + nil, provide `IsEmpty()` method.

## Tracing

Required for all methods taking `context.Context` **except HTTP handlers** (middleware handles those):

```go
span, ctx := opentracing.StartSpanFromContext(ctx, "MethodName")
defer span.Finish()
```

## Error Wrapping

All errors must be wrapped. Error messages start with "error ".

```go
// Good
return nil, fmt.Errorf("error fetching room: %w", err)

// Bad - no wrapping
return nil, err

// Bad - inline assignment
if err := doThing(); err != nil { }
```

## Interface Naming

- Action verbs: `Fetcher`, `Creator`, `Updater`
- PubSub interfaces end with `Notifier`
- PubSub methods start with `Notify`
- Avoid name stuttering: in `vibe` package, use `Room` not `VibeRoom`

## API Conventions

- Endpoints plural: `/rooms`, `/songs`
- Proper REST verbs
- DELETE has no body
- Versioned routes: `/api/v1`
- All handlers use `handleError()`
