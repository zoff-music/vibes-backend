# Music Search and Provider Caching

YouTube and SoundCloud searches use the normalized Redis cache before spending
provider search quota. Spotify follows the same public API shape but currently
queries its provider client directly.

```mermaid
sequenceDiagram
    actor Listener
    participant Platform
    participant API as Vibes backend
    participant Cache as Redis
    participant Provider as Enabled music provider

    Listener->>Platform: Enter a search query
    Platform->>API: GET /{provider}/search?q={query}

    opt Provider search caching is enabled
        API->>Cache: Read normalized provider and query key
        alt Cached results exist
            Cache-->>API: Cached music tracks
            API-->>Platform: Search results
        else Cache miss
            API->>Provider: Search provider
            Provider-->>API: Provider tracks and metadata
            API->>Cache: Cache normalized tracks
            API-->>Platform: Search results
        end
    end

    opt Provider search caching is not enabled
        API->>Provider: Search provider
        Provider-->>API: Provider tracks and metadata
        API-->>Platform: Search results
    end
```
