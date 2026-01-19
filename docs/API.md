# Vibez API Contract

Complete API specification for frontend-backend communication.

---

## Base URL

```
Development: https://127.0.0.1/api/v1 (via make local-dev)
Production: https://api.vibez.app/api/v1
```

---

## Authentication

- **Room Admin**: Password-based, returns session token in cookie
- **Regular User**: Anonymous session, auto-created on room join

---

## Endpoints

### Rooms

#### Create Room
```
POST /rooms
```

**Request:**
```json
{
  "name": "Friday Night Vibes",
  "adminPassword": "optional-password",
  "settings": {
    "skipAllowed": true,
    "democraticSkip": false,
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
  "createdAt": "2024-01-15T20:00:00Z",
  "hasPassword": true,
  "settings": { ... }
}
```

---

#### Get Room
```
GET /rooms/:id
```

**Response:** `200 OK`
```json
{
  "id": "friday-night-vibes-a1b2",
  "name": "Friday Night Vibes",
  "createdAt": "2024-01-15T20:00:00Z",
  "hasPassword": true,
  "settings": {
    "skipAllowed": true,
    "democraticSkip": false,
    "skipVoteThreshold": 0.5,
    "maxContinuousAdds": 3,
    "removeOnPlay": true,
    "loopQueue": false,
    "allowDuplicates": false
  },
  "userCount": 5
}
```

---

#### Update Room (Admin)
```
PATCH /rooms/:id
```

**Headers:** `X-Admin-Token: <token>`

**Request:**
```json
{
  "name": "Updated Name",
  "settings": {
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
      "addedAt": "2024-01-15T20:05:00Z",
      "position": 0
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
  "addedAt": "2024-01-15T20:05:00Z",
  "position": 5
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

#### Reorder Songs (Admin)
```
PATCH /rooms/:id/songs/reorder
```

**Headers:** `X-Admin-Token: <token>`

**Request:**
```json
{
  "itemId": "item-uuid",
  "newPosition": 0
}
```

**Response:** `200 OK`
```json
{
  "items": [ ... ]
}
```

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
