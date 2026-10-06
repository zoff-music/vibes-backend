# Skip the Current Playlist Item

Host mode only permits the host to skip. Server mode may either skip immediately
or collect democratic skip votes according to room settings. This authority
model applies to both MUSIC and WATCH rooms.

```mermaid
sequenceDiagram
    actor Listener
    participant Platform
    participant API as Vibes backend
    participant DB as PostgreSQL
    participant Events as Room event stream

    Listener->>Platform: Skip current item
    Platform->>API: POST /api/v2/rooms/{id}/skips
    API->>DB: Evaluate mode, permissions and skip settings

    alt Skip is forbidden
        DB-->>API: Host-only or disabled error
        API-->>Platform: 403 Forbidden
    else More democratic votes are required
        DB-->>API: Current and required vote counts
        API->>Events: Publish skip_vote
        API-->>Platform: Vote result
        Events-->>Platform: Updated skip count
    else Item is skipped
        DB->>DB: Select next playlist item and update playback
        DB-->>API: New playback state
        API->>DB: Load updated playlist
        API->>Events: Publish playlist_items_update and playback_update
        API-->>Platform: Skip result
        Events-->>Platform: New item and playlist
    end
```
