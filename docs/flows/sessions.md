# Sessions and Live Room Updates

## Join a room

Opening the room event stream registers and maintains an active listener. The
signed session cookie identifies a browser without requiring account creation.

```mermaid
sequenceDiagram
    actor Listener
    participant Platform
    participant API as Vibes backend
    participant DB as PostgreSQL
    participant Events as Room event stream

    Listener->>Platform: Open room
    Platform->>API: GET /rooms/{id}
    API->>DB: Load room, settings and generation state
    DB-->>API: Room
    API-->>Platform: Room response

    Platform->>API: GET /rooms/{id}/events
    API->>Events: Subscribe connection to room topic
    API->>DB: Load current playback state
    API-->>Platform: Connected event and playback state

    loop Every five seconds while connected
        API->>DB: Update participant heartbeat
    end

    API->>Events: Publish listener count
    Events-->>Platform: users_update
```

## Set a room password or authenticate as room administrator

The room-session endpoint handles first-time password setup and subsequent
administrator authentication. Passwords are stored only as bcrypt hashes.

```mermaid
sequenceDiagram
    actor User
    participant Platform
    participant API as Vibes backend
    participant DB as PostgreSQL
    participant Events as Room event stream

    User->>Platform: Submit room password
    Platform->>API: POST /rooms/{id}/sessions
    API->>DB: Load room and password state

    alt Room does not have a password yet
        DB->>DB: Hash and store submitted password
        DB->>DB: Elevate session as room administrator
        DB-->>API: Admin authenticated, first-time setup
        API->>Events: Publish settings_update
        API-->>Platform: Admin session and updated room
    else Room already has a password
        DB->>DB: Compare submitted password with stored hash
        alt Password matches
            DB->>DB: Elevate session as room administrator
            DB-->>API: Admin authenticated
            API-->>Platform: Admin session and room
        else Password does not match
            DB-->>API: Authentication rejected
            API-->>Platform: 403 Forbidden
        end
    end
```
