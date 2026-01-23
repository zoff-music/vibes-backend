# Music Providers

Vibez supports multiple music providers for search and synchronized playback.

## Supported Providers

| Provider | Search | Playback | Auth Required | Notes |
|----------|--------|----------|---------------|-------|
| YouTube | ✅ | ✅ (IFrame) | No (API Key only) | Always available |
| Spotify | ✅ | ✅ (Web SDK) | ✅ (OAuth Premium) | Requires Premium account |
| SoundCloud | ✅ | ✅ (Widget) | ✅ (OAuth) | API key + OAuth |

---

## 📺 YouTube

### Integration
- **Search**: `GET /api/v1/youtube/search?q=query`
- **Details**: `GET /api/v1/youtube/videos/{id}`
- **Playback**: Uses `react-player/youtube` (YouTube IFrame API)

### Configuration
```bash
# Backend environment (.env)
YOUTUBE_API_KEY=your-youtube-data-api-v3-key
```

### Features
- No authentication required for users
- Full search and playback functionality
- Automatic video duration and metadata extraction
- Thumbnail support with multiple resolutions

---

## 🎧 Spotify

### Integration
- **Search**: `GET /api/v1/spotify/search?q=query`
- **Details**: `GET /api/v1/spotify/tracks/{id}`
- **Playback**: Uses Spotify Web Playback SDK
- **Authentication**: OAuth 2.0 flow with PKCE

### Configuration
```bash
# Backend environment (.env)
SPOTIFY_CLIENT_ID=your-spotify-app-client-id
SPOTIFY_CLIENT_SECRET=your-spotify-app-client-secret
```

### Authentication Flow
1. User clicks "Connect Spotify" in the UI
2. Frontend redirects to `GET /api/v1/authorizations/spotify`
3. Backend generates OAuth state and redirects to Spotify
4. Spotify redirects to `GET /api/v1/callbacks/spotify?code=...&state=...`
5. Backend exchanges code for access/refresh tokens
6. Backend stores tokens in SQLite (`access_tokens` table)
7. Backend serves success page with `postMessage` to parent window
8. Frontend receives success message and updates UI

### Token Management
- Access tokens automatically refreshed when expired
- Refresh tokens stored securely in database
- Token validation before each API call
- Automatic re-authentication flow when refresh fails

### Requirements
- **Spotify Premium** account required for Web Playback SDK
- Non-premium users can search but cannot play tracks
- Proper redirect URI configuration in Spotify app settings

---

## ☁️ SoundCloud

### Integration
- **Search**: `GET /api/v1/soundcloud/search?q=query`
- **Details**: `GET /api/v1/soundcloud/tracks/{id}`
- **Playback**: Uses `react-player/soundcloud` (SoundCloud Widget API)
- **Authentication**: OAuth 2.0 flow

### Configuration
```bash
# Backend environment (.env)
SOUNDCLOUD_CLIENT_ID=your-soundcloud-app-client-id
SOUNDCLOUD_CLIENT_SECRET=your-soundcloud-app-client-secret
```

### Authentication Flow
Similar to Spotify OAuth flow:
1. User clicks "Connect SoundCloud"
2. OAuth flow via `GET /api/v1/authorizations/soundcloud`
3. Callback handling at `GET /api/v1/callbacks/soundcloud`
4. Token storage and management

---

## Technical Implementation

### API Protocol
All provider searches return a consistent `SearchResult` model:

```typescript
interface SearchResult {
  id: string;
  sourceType: 'youtube' | 'spotify' | 'soundcloud';
  title: string;
  artist: string;
  thumbnailUrl: string;
  duration: number; // seconds
}
```

### Track Details
Individual track endpoints return detailed `Track` information:

```typescript
interface Track {
  id: string;
  sourceType: 'youtube' | 'spotify' | 'soundcloud';
  title: string;
  artist: string;
  thumbnailUrl: string;
  duration: number;
  // Provider-specific metadata
}
```

### Provider Status
Check which providers are enabled:

```bash
GET /api/v1/providers
```

Response:
```json
{
  "youtube": {
    "enabled": true,
    "requiresAuth": false
  },
  "spotify": {
    "enabled": true,
    "requiresAuth": true
  },
  "soundcloud": {
    "enabled": false,
    "requiresAuth": true
  }
}
```

### Frontend Integration

#### Provider Token Management
```typescript
import { useProviderToken } from '@vibez/shared';

const { getToken, isAuthenticated, authenticate } = useProviderToken('spotify');

// Check authentication status
if (!isAuthenticated) {
  await authenticate(); // Triggers OAuth flow
}

// Get valid token (automatically refreshes if needed)
const token = await getToken();
```

#### Search Integration
```typescript
import { api } from '@vibez/api';
import { safeWrapAsync } from '@vibez/shared';

// Search across providers
const searchProviders = async (query: string) => {
  const [youtubeError, youtubeResults] = await safeWrapAsync(
    api.get('/youtube/search', {}, { q: query })
  );
  
  const [spotifyError, spotifyResults] = await safeWrapAsync(
    api.get('/spotify/search', {}, { q: query })
  );
  
  // Combine results
  return [
    ...(youtubeResults || []),
    ...(spotifyResults || [])
  ];
};
```

#### Player Components
```typescript
import { SpotifyPlayer, VideoPlayer, SoundCloudPlayer } from '@vibez/player';

// Render appropriate player based on source type
const renderPlayer = (song: Song) => {
  switch (song.sourceType) {
    case 'spotify':
      return <SpotifyPlayer track={song} />;
    case 'youtube':
      return <VideoPlayer url={`https://youtube.com/watch?v=${song.sourceId}`} />;
    case 'soundcloud':
      return <SoundCloudPlayer url={song.sourceId} />;
  }
};
```

### Casting Integration

When casting to Chromecast receivers, authentication tokens are automatically handled:

1. **Token Injection**: `castManager.ts` fetches current tokens and includes them in cast session data
2. **Receiver Authentication**: Cast receiver extracts tokens from session data
3. **Automatic Playback**: Receiver uses tokens for authenticated provider playback

```typescript
// Cast manager automatically handles token injection
const castCurrentSong = async (song: Song) => {
  const tokens = await getAllProviderTokens();
  
  await castManager.loadMedia({
    contentId: song.sourceId,
    contentType: song.sourceType,
    customData: {
      tokens, // Automatically injected
      roomId,
      songData: song
    }
  });
};
```

### Error Handling

#### Provider Availability
```typescript
// Check if provider is available before using
const [error, providers] = await safeWrapAsync(api.get('/providers'));

if (providers?.spotify?.enabled) {
  // Spotify is available
} else {
  // Show fallback or disable Spotify features
}
```

#### Authentication Errors
```typescript
// Handle authentication failures gracefully
const [error, results] = await safeWrapAsync(
  api.get('/spotify/search', {}, { q: query })
);

if (error?.message.includes('unauthorized')) {
  // Trigger re-authentication
  await authenticate();
  // Retry search
}
```

#### Rate Limiting
```typescript
// Handle rate limiting from providers
if (error?.message.includes('rate limit')) {
  // Show user-friendly message
  setError('Too many requests. Please wait a moment.');
  // Implement exponential backoff
}
```

### Database Schema

Provider authentication data is stored in these tables:

#### `access_tokens`
```sql
CREATE TABLE access_tokens (
  user_id TEXT NOT NULL,
  provider TEXT NOT NULL,
  access_token TEXT NOT NULL,
  refresh_token TEXT,
  expires_at DATETIME,
  refresh_expires_at DATETIME,
  last_checked_at DATETIME DEFAULT now,
  PRIMARY KEY (user_id, provider)
);
```

#### `auth_tokens` (OAuth state)
```sql
CREATE TABLE auth_tokens (
  user_id TEXT NOT NULL,
  provider TEXT NOT NULL,
  code TEXT,
  state TEXT NOT NULL,
  expires_at DATETIME NOT NULL,
  PRIMARY KEY (user_id, provider)
);
```

#### `pending_oauth_state` (CSRF protection)
```sql
CREATE TABLE pending_oauth_state (
  user_id TEXT NOT NULL,
  state TEXT NOT NULL,
  expires_at DATETIME NOT NULL,
  PRIMARY KEY (user_id, state)
);
```

### Security Considerations

1. **OAuth State Validation**: All OAuth flows use CSRF-protected state parameters
2. **Token Encryption**: Consider encrypting stored tokens in production
3. **Token Rotation**: Refresh tokens are automatically rotated when possible
4. **Scope Limitation**: OAuth scopes are limited to minimum required permissions
5. **HTTPS Only**: All OAuth redirects require HTTPS in production
