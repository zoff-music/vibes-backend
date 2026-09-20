# Remote Pairing and Targeted Control

## Pair a controller with a player

The player owns the remote session. Pairing gives another device control of that
player without making the controller an additional music listener.

```mermaid
sequenceDiagram
    actor User
    participant Player
    participant Controller
    participant API as Vibes backend
    participant DB as PostgreSQL
    participant Events as Redis remote stream

    User->>Player: Enable remote control
    Player->>API: POST /api/v1/remotes
    API->>DB: Create owned remote and pairing state
    API-->>Player: Remote identity and pairing information
    Player->>API: Subscribe to remote events
    User->>Controller: Open pairing link or scan code
    Controller->>API: POST /api/v1/remotes/{id}/sessions
    API->>DB: Validate pairing and associate controller
    API-->>Controller: Paired remote session
    Controller->>API: PATCH /api/v1/remotes/{id}
    API->>DB: Validate permissions and update remote state
    API->>Events: Publish targeted state or command
    Events-->>Player: Apply command to paired player
    Player->>API: Report player state
    API->>Events: Publish state for controller
    Events-->>Controller: Updated player state
```

Targeted player state and room-wide playback events remain separate. Clients
track the origin of changes to avoid command echoes. Ending the remote session
removes that pairing; it does not delete the room or its queue.
