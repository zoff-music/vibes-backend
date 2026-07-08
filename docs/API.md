# Vibez API Contract

Complete API specification for frontend-backend communication.

---

## Base URL

```
Development: http://localhost:8080/api/v1
Production: https://api.vibez.app/api/v1
```

---

## Authentication

- **Room Admin**: Password-based authentication via `POST /rooms/{id}/sessions`
- **Regular User**: Anonymous session auto-created via middleware on first request
- **Global Admin**: Password-based authentication via `POST /admin/sessions`
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
`POST /rooms`

**Request Body:**
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
    "allowDuplicates": false,
    "enabledSources": ["youtube", "spotify", "soundcloud"],
    "onlyAdminAddSongs": false
  }
}
```

**Response:** `201 Created`

---

#### Get Room
`GET /rooms/{id}`

---

#### Create Session (Join Room / Authenticate as Admin)
`POST /rooms/{id}/sessions`

**Request Body:**
```json
{
  "nickname": "DJ Cool",
  "password": "optional-admin-password"
}
```

---

#### Update Room Settings (Admin/Host Only)
`PATCH /rooms/{id}/settings`

---

### Queue Management

#### Get Songs
`GET /rooms/{id}/songs`

---

#### Add Song to Queue
`POST /rooms/{id}/songs`

**Request Body:**
```json
{
  "sourceType": "youtube",
  "sourceId": "dQw4w9WgXcQ"
}
```

---

#### Remove Song from Queue
`DELETE /rooms/{id}/songs/{songId}`

---

#### Vote for Song
`POST /rooms/{id}/songs/{songId}`

---

### Playback Control

#### Get Playback State
`GET /rooms/{id}/states`

---

#### Update Playback State
`PUT /rooms/{id}/states`

**Request Body:**
```json
{
  "action": "play" | "pause" | "seek",
  "positionMs": 45000
}
```

---

#### Skip Song
`POST /rooms/{id}/skips`

---

### Real-time Events (SSE)

#### Subscribe to Room Events
`GET /rooms/{id}/events`

---

### Music Search & Track Details

#### Search Music
`GET /{youtube|spotify|soundcloud}/search?q=query`

#### Get Track Details
`GET /{youtube|videos|tracks}/{id}`

---

### OAuth & Authorization

#### Start OAuth Flow
`GET /authorizations/{provider}`

#### Get/Refresh Access Token
`GET /tokens/{provider}`

#### Get Enabled Providers
`GET /providers`

---

### Global Admin Routes (Admin Only)

#### Admin Login
`POST /admin/sessions`

**Request Body:**
```json
{
  "password": "global-admin-password"
}
```

---

#### Admin Logout
`DELETE /admin/sessions`

---

#### List All Rooms
`GET /admin/rooms`

---

#### Global Admin Events (SSE)
`GET /admin/events`

---

## Data Types

### SourceType
`"youtube" | "spotify" | "soundcloud"`

### RoomMode
`"server" | "host"`

---

## Error Responses

All endpoints return a consistent error format:

```json
{
  "error": "Error message",
  "code": "ERROR_CODE"
}
```
