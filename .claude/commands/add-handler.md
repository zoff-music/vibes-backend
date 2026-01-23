# Add Backend Handler

Create a new HTTP handler following project conventions.

## Requirements

Provide:
- Handler name (e.g., "GetRoom", "CreateSong")
- HTTP method and path
- Dependencies needed (interfaces from `vibe/` directory)
- Which handler file (rooms.go, songs.go, playback.go, etc.)

## Template

File: `backend/server/internal/handler/{resource}.go`

```go
func {HandlerName}(
    {dep} vibe.{Interface},
    ips vibe.RoomEventNotifier, // If SSE updates needed
) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // Parse path params (gorilla/mux)
        vars := mux.Vars(r)
        roomID := vars["id"]

        // Get session from middleware
        session, ok := helper.GetSessionFromContext(ctx)
        if !ok || session.UserID == "" {
            handleError(w, fmt.Errorf("unauthorized"), 
                http.StatusUnauthorized, false)
            return
        }
        userID := session.UserID

        // Parse request body (for POST/PUT/PATCH)
        var req vibe.{RequestType}
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            handleError(w, fmt.Errorf("error decoding request: %w", err), 
                http.StatusBadRequest, false)
            return
        }

        // Call dependency
        result, err := {dep}.{Method}(ctx, roomID, userID, req)
        if err != nil {
            handleError(w, fmt.Errorf("error {action}: %w", err), 
                http.StatusInternalServerError, true)
            return
        }

        // Broadcast SSE update (if needed)
        if ips != nil {
            payload, _ := json.Marshal(result)
            err = ips.NotifyRoomUpdate(ctx, roomID, vibe.RoomEvent{
                Type:    vibe.{EventType},
                Payload: payload,
                UserID:  userID,
            })
            if err != nil {
                log.Printf("failed to notify room update: %v", err)
            }
        }

        // Respond
        response, err := json.Marshal(result)
        if err != nil {
            handleError(w, fmt.Errorf("error marshaling response: %w", err), 
                http.StatusInternalServerError, true)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write(response)
    }
}
```

## Handler Files

- `rooms.go` - Room CRUD operations (CreateRoom, GetRoom, UpdateRoomSettings)
- `songs.go` - Queue management (GetSongs, AddSong, RemoveSong, ReorderSongs, VoteSong)
- `playback.go` - Playback control (GetPlaybackState, UpdatePlaybackState)
- `skip.go` - Skip voting (SkipSong)
- `search.go` - Music search (SearchMusic, SearchSpotify, SearchSoundCloud)
- `track.go` - Track details (GetMusicTrack, GetSpotifyTrack, GetSoundCloudTrack)
- `authorization.go` - OAuth flows (Authorize, OAuthCallback)
- `events.go` - SSE endpoint (RoomEvents)
- `config.go` - Provider configuration (GetProviders)
- `healthz.go` - Health check

## SSE Event Types

```go
const (
    PlaybackUpdate = "playback_update"
    SongAdded      = "song_added"
    SongRemoved    = "song_removed"
    QueueReordered = "songs_update"
    NewHost        = "new_host"
    UserJoined     = "user_joined"
    UserLeft       = "user_left"
    UsersUpdate    = "users_update"
    SettingsUpdate = "settings_update"
)
```

## Route Registration

Add to `backend/server/router.go`:

```go
api.HandleFunc("/rooms/{id}/endpoint", handler.HandlerName(s.DB, s.InternalPubSub)).Methods(http.MethodPost, http.MethodOptions).Name("HandlerName")
```

## Checklist

- [ ] Handler in `server/internal/handler/{resource}.go`
- [ ] Interface in `vibe/{resource}.go` (if new)
- [ ] Database method in `client/database/{resource}.go` (if needed)
- [ ] Route added in `server/router.go`
- [ ] No tracing in handler (middleware handles it)
- [ ] Errors wrapped with context
- [ ] Session validation for protected endpoints
- [ ] SSE updates for real-time features
- [ ] Proper HTTP status codes
- [ ] JSON content type header
