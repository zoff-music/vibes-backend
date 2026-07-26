# Playlist Song Operations

## Add a song

The frontend submits normalized metadata returned by an enabled provider. The
backend validates the room and source before inserting the song.

```mermaid
sequenceDiagram
    actor Listener
    participant Platform
    participant API as Vibes backend
    participant DB as PostgreSQL
    participant Events as Room event stream

    Listener->>Platform: Select Add song
    Platform->>API: POST /rooms/{id}/songs
    API->>DB: Load room and permissions
    DB-->>API: Room settings
    API->>API: Validate session, enabled source and provider URL
    API->>DB: Add song atomically
    DB-->>API: Added or existing song result
    API->>DB: Load ordered playlist
    API->>Events: Publish queue_reordered

    opt First song in the room
        API->>DB: Initialize playing state
        API->>Events: Publish playback_update
    end

    API-->>Platform: Song result
    Events-->>Platform: Updated playlist and playback
```

## Remove a song

Removing a song is restricted to a room administrator.

```mermaid
sequenceDiagram
    actor Admin as Room administrator
    participant Platform
    participant API as Vibes backend
    participant DB as PostgreSQL
    participant Events as Room event stream

    Admin->>Platform: Remove song
    Platform->>API: DELETE /rooms/{id}/songs/{songId}
    API->>DB: Load room and administrator status
    DB-->>API: Room permissions

    alt User is a room administrator
        API->>DB: Remove song
        API->>DB: Load ordered playlist
        API->>Events: Publish queue_reordered
        API-->>Platform: 204 No Content
        Events-->>Platform: Updated playlist
    else User is not an administrator
        API-->>Platform: 403 Forbidden
    end
```

## Vote for a song

Each signed session can vote once for a given song. Votes affect playlist order
but do not replace the listener's local player state.

```mermaid
sequenceDiagram
    actor Listener
    participant Platform
    participant API as Vibes backend
    participant DB as PostgreSQL
    participant Events as Room event stream

    Listener->>Platform: Vote for song
    Platform->>API: POST /rooms/{id}/songs/{songId}
    API->>DB: Record vote for session and song

    alt Session already voted
        DB-->>API: Already-voted error
        API-->>Platform: 409 Conflict
    else Vote accepted
        DB-->>API: Vote recorded
        API->>DB: Load newly ordered playlist
        API->>Events: Publish queue_reordered
        API-->>Platform: 204 No Content
        Events-->>Platform: Updated playlist order
    end
```
