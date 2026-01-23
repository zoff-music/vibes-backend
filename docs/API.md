# Vibez API Contract

Complete API specification for frontend-backend communication.

---

## Base URL

```
Development: https://localhost/api/v1 (via make local-dev)
Production: https://api.vibez.app/api/v1
```

---

## Authentication

- **Room Admin**: Password-based authentication via `POST /rooms/{id}/sessions`
- **Regular User**: Anonymous session auto-created via middleware on first request
- **Session Storage**: HTTP-only cookie with UUID-based user ID
- **OAuth**: Spotify, YouTube, SoundCloud via OAuth 2.0 flow

---

## Room Modes

### Server Mode (`"server"`)
- Server controls playback automatically
- Auto-plays first song when added to empty queue
- Continues to next song when current ends
- Perfect for 24/7 radio stations
- Skip settings apply to all users

### Host Mode (`"host"`)
- Only the host can control playback (play/pause/seek/skip)
- Other users can only add songs and vote
- Host is the room creator or assigned user
- Great for parties with a DJ
- Democratic skip voting disabled (host decides)

---

## Endpoints

### Room Management

#### Create Room
```
POST /rooms
```

**Request:**
```json
{
  "name": "Friday Night Vibes",
  "mode": "server",
  "password": "optional-admin-password",
  "settings": {
    "skipAllowed": true,
    "democraticSkip": true,
    "skipVoteThreshold": 0.5,
    "maxContinuousAdds": 3,
    "removeOnPlay": true,
    "loopQueue": false,
    "allowDuplicates": false
  }
}
```

**Response:** `201 Created`
```json
{
  "id": "friday-night-vibes-a1b2",
  "name": "Friday Night Vibes",
  "mode": "server",
  "createdAt": "2024-01-15T20:00:00Z",
  "hasPassword": true,
  "hostId": "user-uuid-if-host-mode",
  "activeSources": ["youtube", "spotify"],
  "settings": {
    "skipAllowed": true,
    "democraticSkip": true,
    "skipVoteThreshold": 0.5,
    "maxContinuousAdds": 3,
    "removeOnPlay": true,
    "loopQueue": false,
    "allowDuplicates": false
  }
}
```

---

#### Get Room
```
GET /rooms/{id}
```

**Response:** `200 OK`
```json
{
  "id": "friday-night-vibes-a1b2",
  "name": "Friday Night Vibes",
  "mode": "server",
  "createdAt": "2024-01-15T20:00:00Z",
  "hasPassword": true,
  "hostId": "user-uuid-if-host-mode",
  "activeSources": ["youtube", "spotify", "soundcloud"],
  "userCount": 5,
  "isAdmin": false,
  "userId": "current-user-uuid",
  "settings": {
    "skipAllowed": true,
    "democraticSkip": true,
    "skipVoteThreshold": 0.5,
    "maxContinuousAdds": 3,
    "removeOnPlay": true,
    "loopQueue": false,
    "allowDuplicates": false
  }
}
```

---

#### Authenticate as Admin
```
POST /rooms/{id}/sessions
```

**Purpose:** Authenticate as room admin using the admin password. Regular users join rooms automatically via session middleware - no explicit join endpoint needed.

**Request:**
```json
{
  "password": "admin-password",
  "nickname": "DJ Mike"
}
```

**Response:** `200 OK`
```json
{
  "userId": "user-uuid",
  "isAdmin": true,
  "nickname": "DJ Mike"
}
```

**Note:** Regular room participation happens automatically when accessing any room endpoint. This endpoint is only used when a user wants to authenticate as the room admin.

---

#### Update Room Settings (Admin Only)
```
PATCH /rooms/{id}/settings
```

**Request:**
```json
{
  "mode": "host",
  "settings": {
    "skipAllowed": false,
    "democraticSkip": false,
    "skipVoteThreshold": 0.6,
    "maxContinuousAdds": 5,
    "removeOnPlay": false,
    "loopQueue": true,
    "allowDuplicates": true
  }
}
```

**Response:** `200 OK`
```json
{
  "settings": { /* updated settings */ }
}
```

---

### Queue Management

#### Get Queue
```
GET /rooms/{id}/songs
```

**Response:** `200 OK`
```json
[
  {
    "id": "song-uuid",
    "sourceType": "youtube",
    "sourceId": "dQw4w9WgXcQ",
    "title": "Never Gonna Give You Up",
    "artist": "Rick Astley",
    "thumbnailUrl": "https://img.youtube.com/vi/dQw4w9WgXcQ/maxresdefault.jpg",
    "duration": 213,
    "addedBy": "user-uuid",
    "addedByNickname": "DJ Mike",
    "addedAt": "2024-01-15T20:30:00Z",
    "voteCount": 3
  }
]
```

---

#### Add Song to Queue
```
POST /rooms/{id}/songs
```

**Request:**
```json
{
  "sourceType": "youtube",
  "sourceId": "dQw4w9WgXcQ",
  "title": "Never Gonna Give You Up",
  "artist": "Rick Astley",
  "thumbnailUrl": "https://img.youtube.com/vi/dQw4w9WgXcQ/maxresdefault.jpg",
  "duration": 213
}
```

**Response:** `201 Created`
```json
{
  "id": "song-uuid",
  "position": 5
}
```

---

#### Remove Song from Queue
```
DELETE /rooms/{id}/songs/{songId}
```

**Response:** `204 No Content`

---

#### Vote for Song
```
POST /rooms/{id}/songs/{songId}
```

Vote for a song in the queue. Songs are ordered by vote count (highest first), with `added_at` timestamp used as a tiebreaker for songs with equal votes.

**Response:** `204 No Content`

**Notes:**
- Users can only vote once per song
- Voting affects queue ordering - songs with more votes appear higher in the queue
- Votes are cleared when a song is skipped or removed
- Queue is automatically reordered by: `vote_count DESC, added_at ASC`

---

---

### Playback Control

#### Get Playback State
```
GET /rooms/{id}/states
```

**Response:** `200 OK`
```json
{
  "currentSong": {
    "id": "song-uuid",
    "sourceType": "youtube",
    "sourceId": "dQw4w9WgXcQ",
    "title": "Never Gonna Give You Up",
    "artist": "Rick Astley",
    "thumbnailUrl": "https://img.youtube.com/vi/dQw4w9WgXcQ/maxresdefault.jpg",
    "duration": 213,
    "addedBy": "user-uuid",
    "addedByNickname": "DJ Mike",
    "addedAt": "2024-01-15T20:30:00Z",
    "voteCount": 0
  },
  "isPlaying": true,
  "positionMs": 45000,
  "updatedAt": "2024-01-15T20:30:45Z",
  "serverTimeMs": 1705349445000
}
```

---

#### Update Playback State
```
PUT /rooms/{id}/states
```

**Request:**
```json
{
  "action": "play",
  "positionMs": 45000
}
```

**Actions:**
- `"play"` - Start playback
- `"pause"` - Pause playback  
- `"seek"` - Seek to position (requires `positionMs`)

**Response:** `200 OK` (same as Get Playback State)

---

#### Skip Song
```
POST /rooms/{id}/skips
```

**Response:** `200 OK` (returns updated playback state)

**Skip Logic:**
- **Force Skip**: Admin or host can skip immediately
- **Vote Skip**: Regular users vote; requires threshold of active participants
- **Minimum 2 votes**: Enforced to prevent single-user skips in groups
- **Single participant**: Can skip alone
- **Vote clearing**: Votes cleared when song skips or is removed

---

### Real-time Events (SSE)

#### Subscribe to Room Events
```
GET /rooms/{id}/events
```

**Response:** Server-Sent Events stream

**Event Types:**
- `playback_update` - Playback state changed
- `song_added` - Song added to queue
- `song_removed` - Song removed from queue
- `songs_update` - Queue reordered
- `new_host` - Host changed (host mode)
- `user_joined` - User joined room
- `user_left` - User left room
- `users_update` - User count updated
- `settings_update` - Room settings changed

**Event Format:**
```
event: playback_update
data: {"type":"playback_update","payload":{"currentSong":{...},"isPlaying":true,"positionMs":45000},"userId":"triggering-user-id"}

event: users_update
data: {"type":"users_update","payload":5}
```

---

### Music Search & Track Details

#### Search YouTube
```
GET /youtube/search?q=never+gonna+give+you+up
```

**Response:** `200 OK`
```json
{
  "results": [
    {
      "id": "dQw4w9WgXcQ",
      "title": "Rick Astley - Never Gonna Give You Up (Official Video)",
      "artist": "Rick Astley",
      "thumbnailUrl": "https://img.youtube.com/vi/dQw4w9WgXcQ/maxresdefault.jpg",
      "duration": 213,
      "sourceType": "youtube"
    }
  ]
}
```

---

#### Get YouTube Track Details
```
GET /youtube/videos/{id}
```

**Response:** `200 OK` (same format as search result)

---

#### Search Spotify
```
GET /spotify/search?q=never+gonna+give+you+up
```

**Response:** `200 OK` (same format as YouTube search)

---

#### Get Spotify Track Details
```
GET /spotify/tracks/{id}
```

**Response:** `200 OK` (same format as search result)

---

#### Search SoundCloud
```
GET /soundcloud/search?q=never+gonna+give+you+up
```

**Response:** `200 OK` (same format as YouTube search)

---

#### Get SoundCloud Track Details
```
GET /soundcloud/tracks/{id}
```

**Response:** `200 OK` (same format as search result)

---

### OAuth & Authorization

#### Start OAuth Flow
```
GET /authorizations/{provider}
```

**Providers:** `spotify`, `youtube`, `soundcloud`

**Response:** `302 Redirect` to provider's OAuth URL

---

#### OAuth Callback
```
GET /callbacks/{provider}?code=auth_code&state=csrf_token
```

**Response:** `200 OK` with postMessage to parent window

---

#### Get/Refresh Access Token
```
GET /tokens/{provider}
```

**Response:** `200 OK`
```json
{
  "accessToken": "provider-access-token",
  "expiresAt": "2024-01-15T21:00:00Z"
}
```

---

#### Get Enabled Providers
```
GET /providers
```

**Response:** `200 OK`
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
    "enabled": true,
    "requiresAuth": true
  }
}
```

---

## Error Responses

All endpoints return consistent error format:

```json
{
  "error": "Error message",
  "code": "ERROR_CODE"
}
```

**Common Status Codes:**
- `400` - Bad Request (validation errors)
- `401` - Unauthorized (missing/invalid session)
- `403` - Forbidden (insufficient permissions)
- `404` - Not Found (room/song not found)
- `409` - Conflict (duplicate song, rate limiting)
- `500` - Internal Server Error

---

## Rate Limiting

- **Add Song**: Max 3 continuous adds per user (configurable via `maxContinuousAdds`)
- **Skip Votes**: One vote per user per song
- **Search**: No explicit limits (relies on provider limits)

---

## Data Types

### SourceType
```
"youtube" | "spotify" | "soundcloud"
```

### RoomMode
```
"server" | "host"
```

### RoomAction
```
"play" | "pause" | "seek" | "skip" | "vote"
```
    "skipAllowed": false
  }
}
```

**Response:** `200 OK` (updated room object)

---

#### Create Session (Join Room)
```
POST /rooms/:id/sessions
```

Creates a session for the room. If `password` is provided and matches the admin password, the session is granted admin privileges.

**Request:**
```json
{
  "nickname": "DJ Cool",
  "password": "optional-admin-password"
}
```

**Response:** `200 OK`
```json
{
  "userId": "user-uuid",
  "nickname": "DJ Cool",
  "isAdmin": false,
  "room": { ... }
}
```

With correct admin password:
```json
{
  "userId": "user-uuid",
  "nickname": "DJ Cool",
  "isAdmin": true,
  "room": { ... }
}
```

Sets session cookie for subsequent requests.

**Errors:**
- `401 Unauthorized` - Password provided but incorrect
- `404 Not Found` - Room doesn't exist

---

### Songs

#### Get Songs
```
GET /rooms/:id/songs
```

**Response:** `200 OK`
```json
{
  "items": [
    {
      "id": "item-uuid",
      "sourceType": "youtube",
      "sourceId": "dQw4w9WgXcQ",
      "title": "Rick Astley - Never Gonna Give You Up",
      "artist": "Rick Astley",
      "thumbnailUrl": "https://i.ytimg.com/vi/dQw4w9WgXcQ/default.jpg",
      "duration": 212,
      "addedBy": "user-uuid",
      "addedByNickname": "DJ Cool",
      "addedAt": "2024-01-15T20:05:00Z"
    }
  ],
  "totalCount": 1
}
```

---

#### Add Song
```
POST /rooms/:id/songs
```

**Request:**
```json
{
  "sourceType": "youtube",
  "sourceId": "dQw4w9WgXcQ"
}
```

**Response:** `201 Created`
```json
{
  "id": "item-uuid",
  "sourceType": "youtube",
  "sourceId": "dQw4w9WgXcQ",
  "title": "Rick Astley - Never Gonna Give You Up",
  "artist": "Rick Astley",
  "thumbnailUrl": "https://i.ytimg.com/vi/dQw4w9WgXcQ/default.jpg",
  "duration": 212,
  "addedBy": "user-uuid",
  "addedAt": "2024-01-15T20:05:00Z"
}
```

**Errors:**
- `400 Bad Request` - Invalid source
- `409 Conflict` - Duplicate not allowed
- `429 Too Many Requests` - Exceeded maxContinuousAdds

---

#### Remove Song
```
DELETE /rooms/:id/songs/:songId
```

**Response:** `204 No Content`

**Errors:**
- `403 Forbidden` - Not admin and not the user who added it
- `404 Not Found` - Item doesn't exist

---

---

### Room Actions

#### Get Playback State
```
GET /rooms/:id/states
```

**Response:** `200 OK`
```json
{
  "currentSongId": "song-uuid",
  "currentSong": {
    "id": "song-uuid",
    "sourceType": "youtube",
    "sourceId": "dQw4w9WgXcQ",
    "title": "Rick Astley - Never Gonna Give You Up",
    "artist": "Rick Astley",
    "thumbnailUrl": "https://i.ytimg.com/vi/dQw4w9WgXcQ/default.jpg",
    "duration": 212,
    "addedBy": "user-uuid",
    "addedByNickname": "DJ Cool",
    "addedAt": "2024-01-15T20:05:00Z"
  },
  "isPlaying": true,
  "positionMs": 45000,
  "updatedAt": "2024-01-15T20:06:30Z",
  "serverTimeMs": 1705348990000
}
```

**Errors:**
- `401 Unauthorized` - Not authenticated
- `404 Not Found` - Room doesn't exist

---

### Room Actions

All playback and room control actions use a single endpoint with an `action` discriminator.

```
POST /rooms/:id
```

#### Action: play
Start playback.

**Request:**
```json
{
  "action": "play"
}
```

**Response:** `200 OK`
```json
{
  "action": "play",
  "playback": {
    "currentSongId": "song-uuid",
    "currentSong": { ... },
    "isPlaying": true,
    "positionMs": 45000,
    "updatedAt": "2024-01-15T20:06:30Z",
    "serverTimeMs": 1705348990000
  }
}
```

---

#### Action: pause
Pause playback.

**Request:**
```json
{
  "action": "pause"
}
```

**Response:** `200 OK`
```json
{
  "action": "pause",
  "playback": {
    "isPlaying": false,
    "positionMs": 45000,
    ...
  }
}
```

---

#### Action: seek
Seek to a position.

**Request:**
```json
{
  "action": "seek",
  "positionMs": 60000
}
```

**Response:** `200 OK`
```json
{
  "action": "seek",
  "playback": { ... }
}
```

---

#### Action: skip
Skip to next song (admin or if skipAllowed).

**Request:**
```json
{
  "action": "skip"
}
```

**Response:** `200 OK`
```json
{
  "action": "skip",
  "skipped": true,
  "nextSong": { ... },
  "playback": { ... }
}
```

**Errors:**
- `403 Forbidden` - Skip not allowed / not admin

---

#### Action: vote
Vote to skip current song (democratic skip).

**Request:**
```json
{
  "action": "vote"
}
```

**Response:** `200 OK`
```json
{
  "action": "vote",
  "voted": true,
  "currentVotes": 3,
  "requiredVotes": 5,
  "skipped": false
}
```

If threshold is reached:
```json
{
  "action": "vote",
  "voted": true,
  "currentVotes": 5,
  "requiredVotes": 5,
  "skipped": true,
  "nextSong": { ... },
  "playback": { ... }
}
```

---

### Music Providers (Search & Tracks)

#### Search
```
GET /api/v1/{provider}/search?q=query
```
**Providers**: `youtube`, `spotify`, `soundcloud`

#### Get Track Details
```
GET /api/v1/{provider}/tracks/{id}
GET /api/v1/youtube/videos/{id} (YouTube legacy)
```

---

### Authorization & Config

#### List Authorizations
```
GET /api/v1/authorizations
```

#### Authorize Provider
```
GET /api/v1/authorizations/{provider}
```
**Providers**: `spotify`, `youtube`, `soundcloud`

#### Get Enabled Providers
```
GET /api/v1/providers
```

---

### Server-Sent Events

#### Subscribe to Room Events
```
GET /rooms/:id/events
```

**Headers:**
```
Accept: text/event-stream
Cache-Control: no-cache
```

**Event Stream:**
```
event: room:state
data: {"id":"friday-night-vibes","name":"Friday Night Vibes",...}

event: queue:update
data: {"items":[...],"totalCount":5}

event: playback:sync
data: {"currentItemId":"item-uuid","isPlaying":true,"positionMs":45000,"serverTimeMs":1705348990000}

event: users:update
data: {"users":[{"id":"user-uuid","nickname":"DJ Cool","isAdmin":false}],"count":5}

event: skip:vote
data: {"userId":"user-uuid","itemId":"item-uuid","currentVotes":3,"requiredVotes":5}

event: error
data: {"code":"room_not_found","message":"Room does not exist"}
```

**Heartbeat:** Server sends `:ping` comment every 30 seconds to keep connection alive.

---

## Error Response Format

All errors follow this structure:

```json
{
  "error": "error_code",
  "message": "Human readable message",
  "details": {}
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `room_not_found` | 404 | Room doesn't exist |
| `item_not_found` | 404 | Queue item doesn't exist |
| `invalid_request` | 400 | Malformed request body |
| `invalid_source` | 400 | YouTube video not found/invalid |
| `duplicate_item` | 409 | Song already in queue |
| `rate_limited` | 429 | Too many queue additions |
| `unauthorized` | 401 | Not authenticated |
| `forbidden` | 403 | Not authorized (e.g., not admin) |
| `internal_error` | 500 | Server error |

---

## Validation

We use `wiretyped` and `yup` schemas for request/response validation.
See `frontend/packages/models/src/schemas/` for the definitive schema definitions.
