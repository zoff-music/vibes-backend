# Backend Coding Rules

Non-negotiable conventions. Follow strictly.

## Critical Rules

- **No service/repository patterns** - no `service.*` or `repository.*` layers
- **No `New*` constructors** - prefer struct literals
- **No generic slice returns for workers** - worker methods must return single items or nil
- **Atomic DB Operations** - prefer single-action DB methods over list+loop in handlers
- **Control Flow Errors** - use `internalerror.ErrExpected{Err: internalerror.ErrNonRecoverable{...}}` for no-op
- **NO Error Constructors** - always use struct literals for errors
- **No inline error assignment** - never `if err := ...; err != nil {}`
- **Never return `nil, nil`** from `(*T, error)` functions
- **All errors wrapped** with context: `fmt.Errorf("error doing X: %w", err)`
- **All HTTP in clients** - packages under `client/`, consumed via interfaces
- **Domain types in `vibe/`** - ALL business logic types in separate files
- **Don't touch `monitoring/`**
- **Limit Return Values** - NEVER return more than 2 values. 3 or more is strictly illegal.
- **Strict Typing**: Use strong typing everywhere. Avoid `interface{}`/`any` unless absolutely necessary (e.g. strict reflection).
- **No `any` type**: Use explicit types, compose when needed.
- **Error Handling**: Return errors, don't panic. Use custom error types for specific failure modes.
- **NO Inlined Structs** - never use `struct{}{}` or anonymous structs.
- **NO Hardcoded JSON Slices** - never use `[]byte("{}")` or similar hacks.
- **Separate Const Declarations** - Do not group constants in a block (e.g. `const (...)`). Declare each on its own line: `const X = "x"`.

## File Layout

```
client/                    # External integrations
├── database/              # SQLite client
│   ├── database.go        # Client struct and Init
│   ├── rooms.go           # Room operations
│   ├── songs.go           # Queue operations
│   ├── playback.go        # Playback state operations
│   ├── participants.go    # User session operations
│   ├── skip.go            # Skip voting operations
│   ├── users.go           # User management
│   └── authorization.go   # OAuth token operations
├── youtube/               # YouTube API
│   ├── youtube.go         # Client struct and Init
│   ├── search.go          # Search method
│   ├── track.go           # GetTrack method
│   └── authorization.go   # OAuth flow
├── soundcloud/            # SoundCloud API
│   ├── soundcloud.go      # Client struct and Init
│   ├── search.go          # Search method
│   ├── track.go           # GetTrack method
│   └── authorization.go   # OAuth flow
├── spotify/               # Spotify API
│   ├── spotify.go         # Client struct, Init, token management
│   ├── search.go          # Search method
│   ├── track.go           # GetTrack method
│   ├── token.go           # Token refresh operations
│   └── authorization.go   # OAuth flow
└── internalpubsub/        # SSE broadcasting
    └── internalpubsub.go  # PubSub client

server/
├── server.go              # Client init, route wiring
├── router.go              # Route definitions
└── internal/handler/      # HTTP handlers
    ├── rooms.go           # Room CRUD operations
    ├── songs.go           # Queue management
    ├── playback.go        # Playback control
    ├── skip.go            # Skip voting
    ├── search.go          # Music search (all providers)
    ├── track.go           # Track details (all providers)
    ├── authorization.go   # OAuth flows
    ├── events.go          # SSE endpoint
    ├── config.go          # Provider configuration
    ├── healthz.go         # Health check
    └── error.go           # Error handling utilities

vibe/                      # ALL domain types & interfaces
├── rooms.go               # Room, RoomSettings types & interfaces
├── songs.go               # Song, AddSongRequest types & interfaces
├── playback.go            # PlaybackState, RoomAction types & interfaces
├── events.go              # SSE event types & interfaces
├── participants.go        # Participant, RoomUser types & interfaces
├── skip.go                # Skip voting interfaces
├── users.go               # User management interfaces
├── sessions.go            # Session types & interfaces
├── search.go              # Search result types & interfaces
├── token.go               # OAuth token types & interfaces
├── authorization.go       # OAuth types & interfaces
└── permissions.go         # Permission checking interfaces
```

## Handler Pattern

Handlers are functions returning `http.HandlerFunc`. Dependencies injected as interfaces.

**IMPORTANT:** Do NOT create helper functions for broadcasting multiple updates (e.g. `broadcastUpdates`). Instead, use `ips.NotifyRoomUpdates` which takes a slice of events.

```go
func GetRoom(rf vibe.RoomFetcher) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        // NO tracing here - middleware handles it

        // Path params via gorilla/mux
        vars := mux.Vars(r)
        id := vars["id"]

        // Get session from middleware
        session, ok := helper.GetSessionFromContext(ctx)
        if !ok || session.UserID == "" {
            handleError(w, fmt.Errorf("unauthorized"), 
                http.StatusUnauthorized, false)
            return
        }

        room, err := rf.GetRoom(ctx, id, session.UserID)
        if err != nil {
            handleError(w, fmt.Errorf("error getting room: %w", err), 
                http.StatusInternalServerError, true)
            return
        }

        body, err := json.Marshal(room)
        if err != nil {
            handleError(w, fmt.Errorf("error marshaling room: %w", err), 
                http.StatusInternalServerError, true)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write(body)
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
    stmt, err := c.DB.Prepare(`
        SELECT r.id, r.name, r.mode, r.admin_password_hash, r.created_at,
               rs.skip_allowed, rs.democratic_skip, rs.skip_vote_threshold,
               rs.max_continuous_adds, rs.remove_on_play, rs.loop_queue, rs.allow_duplicates,
               COALESCE(COUNT(ru.id), 0) as user_count
        FROM rooms r
        LEFT JOIN room_settings rs ON r.id = rs.room_id
        LEFT JOIN room_users ru ON r.id = ru.room_id AND ru.last_seen_at > datetime('now', '-15 seconds')
        WHERE r.id = ?
        GROUP BY r.id
    `)
    if err != nil {
        return fmt.Errorf("error preparing GetRoomStatement: %w", err)
    }
    c.GetRoomStatement = stmt
    return nil
}

func (c *Client) GetRoom(ctx context.Context, id string, userID string) (*vibe.Room, error) {
    span, ctx := opentracing.StartSpanFromContext(ctx, "GetRoom")
    defer span.Finish()

    cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    row := c.GetRoomStatement.QueryRowContext(cctx, id)
    
    var room vibe.Room
    var adminPasswordHash sql.NullString
    // ... scan into room struct
    
    if err := row.Scan(&room.ID, &room.Name, /* ... */); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return &vibe.Room{}, nil // Return empty room, not nil
        }
        return nil, fmt.Errorf("error scanning room: %w", err)
    }

    // Check if user is admin
    room.IsAdmin = c.isUserAdmin(ctx, id, userID, adminPasswordHash.String)
    room.UserID = userID
    
    return &room, nil
}
```

**sql.ErrNoRows handling:** Return empty struct + nil, provide `IsEmpty()` method.

```go
func (r *Room) IsEmpty() bool {
    return r.ID == ""
}
```

## Session Management

Sessions are managed via middleware and stored in context:

```go
// Get session from context
session, ok := helper.GetSessionFromContext(ctx)
if !ok || session.UserID == "" {
    handleError(w, fmt.Errorf("unauthorized"), 
        http.StatusUnauthorized, false)
    return
}

userID := session.UserID
isAdmin := session.IsAdmin // If available
```

## Room Modes

### Server Mode (`vibe.RoomModeServer = "server"`)
- Server controls playback automatically
- Auto-plays first song when added to empty queue
- Continues to next song when current ends
- Skip settings apply to all users

### Host Mode (`vibe.RoomModeHost = "host"`)
- Only the host can control playback (play/pause/seek/skip)
- Other users can only add songs and vote
- Host is determined by `room.HostID`
- Democratic skip voting disabled (host decides)

## Skip Logic Implementation

```go
func (c *Client) SkipSong(ctx context.Context, roomID, userID string, isAdmin bool) (*vibe.PlaybackState, error) {
    // 1. Get room to check permissions and mode
    room, err := c.GetRoom(ctx, roomID, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch room: %w", err)
    }

    isHost := room.HostID == userID

    // 2. Check if skipping is allowed
    if !room.Settings.SkipAllowed && !isHost && !isAdmin {
        return nil, fmt.Errorf("skipping is disabled in this room")
    }

    // 3. Check host mode restrictions
    if room.Mode == vibe.RoomModeHost && !isHost && !isAdmin {
        return nil, fmt.Errorf("only hosts can skip in host mode")
    }

    // 4. Determine if this should be a forced skip
    shouldForce := isHost || isAdmin || !room.Settings.DemocraticSkip

    if shouldForce {
        return c.skipTrack(ctx, roomID)
    }

    // 5. Handle vote skip
    return c.handleVoteSkip(ctx, roomID, userID)
}
```

## SSE Event Broadcasting

Use `NotifyRoomUpdates` for multiple events:

```go
events := []vibe.RoomEvent{
    {
        Type:    vibe.PlaybackUpdate,
        Payload: playbackData,
        UserID:  userID,
    },
    {
        Type:    vibe.SongRemoved,
        Payload: songData,
        UserID:  userID,
    },
}

err = ips.NotifyRoomUpdates(ctx, roomID, events)
if err != nil {
    log.Printf("failed to notify room updates: %v", err)
}
```

## OAuth Flow Implementation

```go
// Start OAuth flow
func Authorize(db vibe.AuthTokenCreator, client vibe.OAuthClient) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Generate state token
        state := generateStateToken()
        
        // Store state in database
        err := db.CreateAuthToken(ctx, userID, provider, "", state, time.Now().Add(10*time.Minute))
        
        // Redirect to provider
        authURL := client.GetAuthURL(state)
        http.Redirect(w, r, authURL, http.StatusFound)
    }
}

// Handle OAuth callback
func OAuthCallback(db vibe.AccessTokenCreator, client vibe.OAuthClient, provider string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        code := r.URL.Query().Get("code")
        state := r.URL.Query().Get("state")
        
        // Verify state token
        authToken, err := db.GetAuthToken(ctx, userID, provider)
        if err != nil || authToken.State != state {
            handleError(w, fmt.Errorf("invalid state"), http.StatusBadRequest, false)
            return
        }
        
        // Exchange code for access token
        token, err := client.ExchangeCodeForToken(ctx, code)
        if err != nil {
            handleError(w, fmt.Errorf("error exchanging code: %w", err), 
                http.StatusInternalServerError, true)
            return
        }
        
        // Store access token
        err = db.UpsertAccessToken(ctx, userID, provider, token)
        if err != nil {
            handleError(w, fmt.Errorf("error storing token: %w", err), 
                http.StatusInternalServerError, true)
            return
        }
        
        // Return success page with postMessage
        w.Header().Set("Content-Type", "text/html")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`<script>window.parent.postMessage({type: 'auth_success', provider: '` + provider + `'}, '*');</script>`))
    }
}
```

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

- Action verbs: `Fetcher`, `Creator`, `Updater`, `Deleter`
- Combined interfaces: `RoomGetterUpdater`, `PlaybackFetcher`
- PubSub interfaces end with `Notifier`
- PubSub methods start with `Notify`
- Avoid name stuttering: in `vibe` package, use `Room` not `VibeRoom`

## API Conventions

- Endpoints plural: `/rooms`, `/songs`
- Proper REST verbs
- DELETE has no body
- Versioned routes: `/api/v1`
- All handlers use `handleError()`
- Path parameters via gorilla/mux: `{id}`, `{songId}`
- Query parameters for search: `?q=query`

## Common Interface Patterns

```go
// Single operations
type RoomFetcher interface {
    GetRoom(ctx context.Context, roomID string, userID string) (*Room, error)
}

type SongAdder interface {
    AddSong(ctx context.Context, roomID string, userID string, req AddSongRequest) (*Song, error)
}

// Combined operations
type RoomGetterUpdater interface {
    GetRoom(ctx context.Context, roomID string, userID string) (*Room, error)
    UpdateRoomSettings(ctx context.Context, roomID string, userID string, settings RoomSettings) error
}

// Background processing
type ExpiredPlaybackProcessor interface {
    ProcessNextExpiredPlayback(ctx context.Context) (*PlaybackState, error)
}
```

## Environment Variables

Backend uses these environment variables:

```bash
# Required
PORT=8080
DATABASE_PATH=./data/vibes.db
YOUTUBE_API_KEY=your-youtube-api-key

# Optional OAuth
SPOTIFY_CLIENT_ID=your-spotify-client-id
SPOTIFY_CLIENT_SECRET=your-spotify-client-secret
SOUNDCLOUD_CLIENT_ID=your-soundcloud-client-id
SOUNDCLOUD_CLIENT_SECRET=your-soundcloud-client-secret

# Optional
LOG_LEVEL=info
CORS_ORIGINS=https://localhost,https://yourdomain.com
```
