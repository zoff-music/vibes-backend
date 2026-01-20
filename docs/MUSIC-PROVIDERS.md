# Music Providers

Vibez supports multiple music providers for search and synchronized playback.

## Supported Providers

| Provider | Search | Playback | Auth Required |
|----------|--------|----------|---------------|
| YouTube | ✅ | ✅ (IFrame) | No (API Key only) |
| Spotify | ✅ | ✅ (Web SDK) | ✅ (OAuth Premium) |
| SoundCloud | ✅ | ✅ (Widget) | No (API Key only) |

---

## 📺 YouTube

### Integration
- **Search**: `GET /api/v1/youtube/search?q=query`
- **Details**: `GET /api/v1/youtube/videos/{id}`
- **Playback**: Uses `react-player/youtube` (IFrame API).

### Environment
- `YOUTUBE_API_KEY`: Required on backend for search.

---

## 🎧 Spotify

### Integration
- **Search**: `GET /api/v1/spotify/search?q=query`
- **Details**: `GET /api/v1/spotify/tracks/{id}`
- **Playback**: Uses `react-spotify-web-playback` (Spotify Web Playback SDK).

### Authentication Flow
1. User clicks "Connect Spotify".
2. Frontend redirects to `/api/v1/authorizations/spotify`.
3. Backend handles OAuth2 flow and stores tokens in SQLite.
4. On success, backend serves a script that `postMessage("spotify-auth-success")` to the opener.
5. Frontend refreshes authorizations via `api.get('/authorizations')`.

### Requirements
- **Spotify Premium** account is required for playback via the Web SDK.
- Non-premium users can search but cannot play.

---

## ☁️ SoundCloud

### Integration
- **Search**: `GET /api/v1/soundcloud/search?q=query`
- **Details**: `GET /api/v1/soundcloud/tracks/{id}`
- **Playback**: Uses `react-player/soundcloud` (SoundCloud Widget).

### Environment
- `SOUNDCLOUD_API_KEY`: Required on backend for search (Client ID).

---

## Technical Implementation Details

### API Protocol
All provider searches return a consistent `MusicTrack` model:
```json
{
  "id": "string",
  "source": "youtube | spotify | soundcloud",
  "title": "string",
  "channelTitle": "string",
  "thumbnailUrl": "string",
  "duration": "ISO8601 Duration"
}
```

### Authorization Caching
The frontend uses a `useAuthCache` hook to deduplicate authorization requests and an `AuthOverlay` to handle interactive auth prompts when a user attempts to play/add restricted content.

### Casting
When casting to a Chromecast receiver, tokens are automatically injected into the `customData` of the `LOAD` request:
1. `castManager.ts` fetches tokens via `getToken(provider)`.
2. Receiver `App.tsx` intercepts `LOAD`, extracts tokens, and calls `setCachedToken()`.
3. Players on the receiver use these handles for authorized playback.
