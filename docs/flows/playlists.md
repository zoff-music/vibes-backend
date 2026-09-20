# Playlist Staging and Incremental Import

## Import a provider playlist

The HTTP handler validates permissions and resolves provider metadata before
publishing work. Items are staged one by one; an incomplete staging attempt is
not visible as a runnable import.

```mermaid
sequenceDiagram
    actor Listener
    participant App
    participant API as Vibes backend
    participant Provider as Music provider
    participant DB as PostgreSQL
    participant Worker as Import app event
    participant Events as Redis room stream

    Listener->>App: Import selected playlist tracks
    App->>API: Submit playlist import
    API->>DB: Read room settings and administrator state
    alt Import is not permitted
        API-->>App: Permission error with user-facing reason
    else Import is permitted
        API->>Provider: Resolve and validate track metadata
        loop Each accepted track
            API->>DB: Stage complete item with explicit position
        end
        API->>DB: Validate staged set and publish import
        API-->>App: Import accepted
        loop Scheduled every 100 ms
            Worker->>DB: Claim and import one item atomically
            DB-->>Worker: Imported song and playback result
            Worker->>Events: Publish queue and playback changes
            Events-->>App: Song delta and playback update
        end
    end
```

The staging request has a two-minute overall budget, while individual database
calls remain bounded. Failed staging is cleaned up and abandoned work is covered
by maintenance. Item identity and position travel together, rather than being
zipped from parallel SQL arrays. The worker starts playback when an import adds
the first song to an empty room.
