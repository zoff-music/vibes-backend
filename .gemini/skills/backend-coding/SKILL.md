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
- Inject dependencies as interfaces from `vibe/` directory
- Use `mux.Vars(r)` for path params
- Get session from middleware: `helper.GetSessionFromContext(ctx)`
- Always use `handleError(w, err, statusCode, shouldLog)` for errors
- NO tracing in handlers (middleware handles it)
- Include SSE updates for real-time features

```go
func GetRoom(rf vibe.RoomFetcher) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        vars := mux.Vars(r)
        roomID := vars["id"]
        
        // Get session from middleware
        session, ok := helper.GetSessionFromContext(ctx)
        if !ok || session.UserID == "" {
            handleError(w, fmt.Errorf("unauthorized"), 
                http.StatusUnauthorized, false)
            return
        }
        
        room, err := rf.GetRoom(ctx, roomID, session.UserID)
        if err != nil {
            handleError(w, fmt.Errorf("error getting room: %w", err), 
                http.StatusInternalServerError, true)
            return
        }
        
        response, err := json.Marshal(room)
        if err != nil {
            handleError(w, fmt.Errorf("error marshaling room: %w", err), 
                http.StatusInternalServerError, true)
            return
        }
        
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write(response)
    }
}
```

### 2. Database Pattern
- **NO Transactions** - Atomic queries only
- Use Prepared Statements (prepare in `prepareXStmt`, execute in `X`)
- 1:1 mapping between method `X` and statement `XStatement`
- Handle `sql.ErrNoRows` by returning empty struct + nil
- Include userID parameter for user-specific operations
- Context timeout of 5 seconds

```go
func (c *Client) GetRoom(ctx context.Context, roomID string, userID string) (*vibe.Room, error) {
    span, ctx := opentracing.StartSpanFromContext(ctx, "GetRoom")
    defer span.Finish()

    cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    row := c.GetRoomStatement.QueryRowContext(cctx, roomID)
    
    var room vibe.Room
    err := row.Scan(&room.ID, &room.Name, /* ... */)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return &vibe.Room{}, nil
        }
        return nil, fmt.Errorf("error scanning room: %w", err)
    }

    // Check if user is admin
    room.IsAdmin = c.isUserAdmin(ctx, roomID, userID, adminPasswordHash)
    room.UserID = userID
    
    return &room, nil
}
```

### 3. Error Handling
- **NO inline error assignment** (`if err := ...; err != nil {}`)
- Wrap ALL errors with `fmt.Errorf("error doing X: %w", err)`
- Error messages start with "error"
- Use `internalerror` package for control flow errors if needed
- Never return `nil, nil` from `(*T, error)` functions

### 4. File Structure
- `server/internal/handler/`: HTTP Handlers organized by resource
  - `rooms.go`: Room CRUD operations
  - `songs.go`: Queue management
  - `playback.go`: Playback control
  - `skip.go`: Skip voting
  - `search.go`: Music search
  - `authorization.go`: OAuth flows
  - `events.go`: SSE endpoint
- `client/`: All external integrations
  - `database/`: SQLite operations (organized by resource)
  - `youtube/`, `spotify/`, `soundcloud/`: Music provider APIs
  - `internalpubsub/`: SSE broadcasting
- `vibe/`: Pure domain types and interfaces (CRITICAL - ALL types here)
  - `rooms.go`: Room and RoomSettings types
  - `songs.go`: Song and queue types
  - `playback.go`: PlaybackState and actions
  - `events.go`: SSE event types
  - `participants.go`: User session types
- `config/`: Configuration management
- `monitoring/`: OpenTelemetry tracing and metrics

### 5. Room Modes
Handle different room modes in business logic:

```go
// Server mode: Server controls playback automatically
if room.Mode == vibe.RoomModeServer {
    // Auto-play logic, skip voting applies
}

// Host mode: Only host can control playback
if room.Mode == vibe.RoomModeHost {
    isHost := room.HostID == userID
    if !isHost && !isAdmin {
        return nil, fmt.Errorf("only hosts can control playback in host mode")
    }
}
```

### 6. SSE Event Broadcasting
Use `NotifyRoomUpdates` for multiple events:

```go
events := []vibe.RoomEvent{
    {
        Type:    vibe.PlaybackUpdate,
        Payload: playbackData,
        UserID:  userID,
    },
    {
        Type:    vibe.SongAdded,
        Payload: songData,
        UserID:  userID,
    },
}

err = ips.NotifyRoomUpdates(ctx, roomID, events)
if err != nil {
    log.Printf("failed to notify room updates: %v", err)
}
```

### 7. Skip Logic Implementation
Democratic skip voting with thresholds:

```go
// Force skip for admin/host or when democratic skip disabled
shouldForce := isHost || isAdmin || !room.Settings.DemocraticSkip

if shouldForce {
    return c.skipTrack(ctx, roomID)
}

// Vote skip with threshold calculation
activeParticipants, err := c.GetActiveParticipants(ctx, roomID, 15*time.Second)
threshold := int(math.Ceil(float64(len(activeParticipants)) * room.Settings.SkipVoteThreshold))
if threshold < 2 && len(activeParticipants) > 1 {
    threshold = 2 // Minimum 2 votes for groups
}
```

### 8. OAuth Flow Implementation
Handle OAuth state management:

```go
// Start OAuth flow
state := generateStateToken()
err := db.CreateAuthToken(ctx, userID, provider, "", state, time.Now().Add(10*time.Minute))

// Handle callback
authToken, err := db.GetAuthToken(ctx, userID, provider)
if err != nil || authToken.State != receivedState {
    return fmt.Errorf("invalid state")
}

// Exchange code for token
token, err := client.ExchangeCodeForToken(ctx, code)
err = db.UpsertAccessToken(ctx, userID, provider, token)
```

### 9. Tracing
Required for all methods taking `context.Context` **except HTTP handlers**:

```go
span, ctx := opentracing.StartSpanFromContext(ctx, "MethodName")
defer span.Finish()
```
