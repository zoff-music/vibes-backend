# Provider Authorization and Tokens

## Authenticate with Spotify, SoundCloud, or YouTube

The provider-specific clients share the same OAuth sequence. Pending state and
PKCE data are stored before redirecting, then consumed atomically by the
callback.

```mermaid
sequenceDiagram
    actor Listener
    participant Platform
    participant API as Vibes backend
    participant DB as PostgreSQL
    participant Provider as Spotify, SoundCloud or YouTube

    Listener->>Platform: Connect provider
    Platform->>API: GET /authorizations/{provider}
    API->>API: Generate state and PKCE verifier
    API->>DB: Save pending OAuth state
    API-->>Platform: Redirect to provider
    Platform->>Provider: Authorization request
    Provider-->>API: GET /callbacks/{provider}?code and state
    API->>DB: Validate and delete pending state
    API->>Provider: Exchange code and PKCE verifier
    Provider-->>API: Access and refresh tokens
    API->>DB: Upsert authorization and access tokens
    API-->>Platform: Redirect to successful callback page
```

## Retrieve or refresh a provider token

```mermaid
sequenceDiagram
    participant Platform
    participant API as Vibes backend
    participant DB as PostgreSQL
    participant Provider as Spotify, SoundCloud or YouTube

    Platform->>API: GET /tokens/{provider}
    API->>DB: Load the session's provider token

    alt Access token is still valid
        DB-->>API: Valid token
        API-->>Platform: Access token and expiry
    else Refresh token is valid
        API->>Provider: Refresh access token
        Provider-->>API: Refreshed token
        API->>DB: Upsert refreshed token
        API-->>Platform: Access token and expiry
    else Authorization is missing or expired
        API-->>Platform: Reauthorization required
    end
```
