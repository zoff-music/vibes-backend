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

### YouTube Search

#### Search for Music
```
GET /youtube/search?q=query
```

Search for YouTube videos by query string.

**Query Parameters:**
- `q` (required): Search query string

**Response:** `200 OK`
```json
[
  {
    "id": "dQw4w9WgXcQ",
    "source": "youtube",
    "title": "Rick Astley - Never Gonna Give You Up",
    "channelTitle": "Rick Astley",
    "thumbnailUrl": "https://i.ytimg.com/vi/dQw4w9WgXcQ/default.jpg",
    "duration": "PT3M33S"
  }
]
```

---

#### Get Video Details
```
GET /youtube/videos/:id
```

Get details for a specific YouTube video by ID.

**Response:** `200 OK`
```json
{
  "id": "dQw4w9WgXcQ",
  "source": "youtube",
  "title": "Rick Astley - Never Gonna Give You Up",
  "channelTitle": "Rick Astley",
  "thumbnailUrl": "https://i.ytimg.com/vi/dQw4w9WgXcQ/default.jpg",
  "duration": "PT3M33S"
}
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

## Yup Schemas (Frontend)

```typescript
// schemas/room.ts
import * as yup from 'yup';

export const roomSettingsSchema = yup.object({
  skipAllowed: yup.boolean().required(),
  democraticSkip: yup.boolean().required(),
  skipVoteThreshold: yup.number().min(0).max(1).required(),
  maxContinuousAdds: yup.number().min(1).max(10).required(),
  removeOnPlay: yup.boolean().required(),
  loopQueue: yup.boolean().required(),
  allowDuplicates: yup.boolean().required(),
});

export const roomSchema = yup.object({
  id: yup.string().required(),
  name: yup.string().required(),
  createdAt: yup.string().required(),
  hasPassword: yup.boolean().required(),
  settings: roomSettingsSchema.required(),
  userCount: yup.number().optional(),
});

export const createRoomRequestSchema = yup.object({
  name: yup.string().min(1).max(100).required(),
  adminPassword: yup.string().optional(),
  settings: roomSettingsSchema.optional(),
});

// schemas/songs.ts
export const songSchema = yup.object({
  id: yup.string().required(),
  sourceType: yup.string().oneOf(['youtube', 'spotify', 'soundcloud']).required(),
  sourceId: yup.string().required(),
  title: yup.string().required(),
  artist: yup.string().optional(),
  thumbnailUrl: yup.string().required(),
  duration: yup.number().required(),
  addedBy: yup.string().required(),
  addedByNickname: yup.string().optional(),
  addedAt: yup.string().required(),
  position: yup.number().required(),
});

export const songsResponseSchema = yup.object({
  songs: yup.array().of(songSchema).required(),
  totalCount: yup.number().required(),
});

// schemas/playback.ts
export const playbackStateSchema = yup.object({
  currentSongId: yup.string().nullable(),
  currentSong: songSchema.nullable(),
  isPlaying: yup.boolean().required(),
  positionMs: yup.number().required(),
  updatedAt: yup.string().required(),
  serverTimeMs: yup.number().required(),
});
```

---

## Wiretyped Endpoint Definitions

```typescript
// api/endpoints.ts
import { RequestDefinitions } from 'wiretyped';
import {
  roomSchema,
  createRoomRequestSchema,
  queueResponseSchema,
  queueItemSchema,
  playbackStateSchema,
} from './schemas';

export const endpoints: RequestDefinitions = {
  // Rooms
  '/rooms': {
    post: {
      body: createRoomRequestSchema,
      response: roomSchema,
    },
  },
  '/rooms/{id}': {
    get: {
      response: roomSchema,
    },
    patch: {
      response: roomSchema,
    },
  },
  '/rooms/{id}/sessions': {
    post: {
      body: yup.object({
        nickname: yup.string().optional(),
        password: yup.string().optional(),
      }),
      response: yup.object({
        userId: yup.string().required(),
        nickname: yup.string().optional(),
        isAdmin: yup.boolean().required(),
        room: roomSchema.required(),
      }),
    },
  },

  // Songs
  '/rooms/{id}/songs': {
    get: {
      response: songsResponseSchema,
    },
    post: {
      body: yup.object({
        sourceType: yup.string().required(),
        sourceId: yup.string().required(),
      }),
      response: songSchema,
    },
  },
  '/rooms/{id}/songs/{songId}': {
    delete: {
      response: yup.object({}),
    },
  },
  '/rooms/{id}/songs/reorder': {
    patch: {
      body: yup.object({
        songId: yup.string().required(),
        newPosition: yup.number().required(),
      }),
      response: songsResponseSchema,
    },
  },

  // Room actions (play, pause, seek, skip, vote) - POST /rooms/{id}
  // Action is determined by request body { action: "play" | "pause" | "seek" | "skip" | "vote" }
  // Response varies by action - see API documentation for details
};

// Action request schemas
export const roomActionSchema = yup.object({
  action: yup.string().oneOf(['play', 'pause', 'seek', 'skip', 'vote']).required(),
  positionMs: yup.number().when('action', {
    is: 'seek',
    then: (schema) => schema.required(),
    otherwise: (schema) => schema.optional(),
  }),
});

// Action response schemas (discriminated by action field)
export const playActionResponseSchema = yup.object({
  action: yup.string().oneOf(['play']).required(),
  playback: playbackStateSchema.required(),
});

export const pauseActionResponseSchema = yup.object({
  action: yup.string().oneOf(['pause']).required(),
  playback: playbackStateSchema.required(),
});

export const seekActionResponseSchema = yup.object({
  action: yup.string().oneOf(['seek']).required(),
  playback: playbackStateSchema.required(),
});

export const skipActionResponseSchema = yup.object({
  action: yup.string().oneOf(['skip']).required(),
  skipped: yup.boolean().required(),
  nextSong: songSchema.nullable(),
  playback: playbackStateSchema.required(),
});

export const voteActionResponseSchema = yup.object({
  action: yup.string().oneOf(['vote']).required(),
  voted: yup.boolean().required(),
  currentVotes: yup.number().required(),
  requiredVotes: yup.number().required(),
  skipped: yup.boolean().required(),
  nextSong: songSchema.nullable(),
  playback: playbackStateSchema.optional(),
});
```
