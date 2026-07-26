# Room Names, Creation, and Settings

## Generate and reserve a room name

Room name suggestions come from the generated name pool and are reserved for the
current signed session before the frontend presents them.

```mermaid
sequenceDiagram
    actor Listener
    participant Platform
    participant API as Vibes backend
    participant DB as PostgreSQL

    Listener->>Platform: Request a room-name suggestion
    Platform->>API: GET /rooms/suggestions
    API->>DB: Select one unconsumed generated name
    DB->>DB: Lock candidate and create session reservation
    DB-->>API: Name, reservation token and expiry
    API-->>Platform: Suggested reserved name
```

## Check and reserve room name availability

A lightweight HEAD request checks whether a room already exists. The reservation
endpoint provides the authoritative availability check used before creation.

```mermaid
sequenceDiagram
    actor Listener
    participant Platform
    participant API as Vibes backend
    participant DB as PostgreSQL

    Listener->>Platform: Enter room name
    Platform->>API: HEAD /rooms/{slug}
    API->>DB: Check existing room
    DB-->>API: Exists or available
    API-->>Platform: 200 exists or 404 available

    Listener->>Platform: Continue creating room
    Platform->>API: POST /rooms/reservations
    API->>DB: Reserve normalized name for session

    alt Name is available
        DB-->>API: Reservation token and expiry
        API-->>Platform: 201 Reserved
    else Name exists or another session reserved it
        DB-->>API: Unavailable
        API-->>Platform: 409 Conflict
    end
```

## Create a room

```mermaid
sequenceDiagram
    actor Listener
    participant Platform
    participant API as Vibes backend
    participant DB as PostgreSQL

    Listener->>Platform: Submit room settings
    Platform->>API: POST /rooms
    API->>API: Normalize name and validate settings
    API->>DB: Check room does not already exist
    opt Password was supplied
        API->>API: Hash room administrator password
    end
    API->>DB: Create room using reservation token
    DB-->>API: Created room
    API-->>Platform: 201 Room
    Platform->>Platform: Navigate to room
```

## Update room settings

```mermaid
sequenceDiagram
    actor Admin as Room administrator
    participant Platform
    participant API as Vibes backend
    participant DB as PostgreSQL
    participant Events as Room event stream

    Admin->>Platform: Change room mode or settings
    Platform->>API: PATCH /rooms/{id}/settings
    API->>DB: Load room and verify administrator permission
    API->>DB: Persist room settings
    DB-->>API: Updated room
    API->>Events: Publish settings_update
    API-->>Platform: Updated room
    Events-->>Platform: Apply settings to connected listeners
```
