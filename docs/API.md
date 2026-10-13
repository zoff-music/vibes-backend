# Zoff API Contract

High-level API contract for frontend-backend communication. The generated
Swagger UI at `/api/swagger/` provides the route and schema reference with Zoff
branding. It is served by the backend and loads the logo from
`https://zoff.me/logo.png`. The header matches the platform navigation, wordmark,
logo sizing, and pixel settings icon. Typography uses the frontend's `MSW98UI`
font assets under `/assets/` on the same origin. Keep the content-hashed font URLs
in `static/swagger/theme.css` aligned with the frontend build when the fonts change;
a standalone local preview must also serve those font assets. Endpoint categories start expanded; individual operations remain collapsed.

`/api/swagger/doc.json` always returns the public specification, excluding admin
operations and schemas used only by those operations. `/api/swagger/admin.json`
returns the full specification only after the existing signed session and admin
cookie checks, including expiry and database session-version validation. It is
unavailable when admin authentication is disabled.

The UI selects the appropriate specification on load, on window focus, after an
admin session request in Swagger, and once a minute while visible. **Refresh
access** also checks immediately after signing in or out elsewhere. Documentation
responses use `Cache-Control: private, no-store`; admin API authorization remains
unchanged. The public source annotations still describe the full API contract; documentation filtering is not a substitute for API access
control.

---

## Base URL

```
Development: http://localhost:8080
Production: https://zoff.me
```

Each endpoint below includes its API version. Current clients use v2
playlist-item REST contracts and the v3 room stream. Shared session, chat,
OAuth, provider, and administration operations retain their supported v1 routes.
Deprecated room, provider, remote, and admin contracts have been removed. See
[sessions and live updates](flows/sessions.md) for snapshot and replay behavior.

## Current MUSIC and WATCH contracts

| Route | Responsibility |
| --- | --- |
| `POST /api/v2/rooms` | Create an immutable `roomType`: `MUSIC` (default) or `WATCH` |
| `GET /api/v2/rooms/{id}` | Room type, settings, host identity and room state |
| `/api/v2/rooms/{id}/playlist-items` | Queue reads and additions using `playlistItem`, `publisher` and item identifiers |
| `/api/v2/rooms/{id}/playlist-items/{playlistItemId}` | Voting and removal |
| `GET /api/v2/rooms/{id}/search/{provider}` | Room-type-aware search for an enabled provider |
| `/api/v2/rooms/{id}/states` | Playlist-item playback state and controls |
| `POST /api/v2/rooms/{id}/failures` | Report provider playback restrictions |
| `POST /api/v2/rooms/{id}/playlists` | Validate and stage playlist imports for the room type |
| `POST /api/v2/rooms/generation` | Generate a new MUSIC or WATCH room |
| `GET /api/v3/rooms/{id}/events` | Playlist-item snapshots, deltas and host updates |
| `GET /api/v3/rooms/public?roomType=WATCH` | Public discovery filtered by room type |
| `GET /api/v2/stats?roomType=WATCH` | Community totals for that type |

MUSIC supports enabled YouTube and SoundCloud sources. WATCH supports YouTube
only and searches across video categories. Room type is independent of playback
mode and cannot be changed through room settings. Generated Swagger describes
the exact request/response fields for each supported route.

---

## Authentication

- **Room Admin**: Password-based authentication via `POST /api/v2/rooms/{id}/sessions`
- **Regular User**: Anonymous session auto-created via middleware on first request
- **Global Admin**: Password-based authentication via `POST /api/v1/admin/sessions`
- **Session Storage**: HTTP-only cookie with UUID-based user ID
- **OAuth**: YouTube and SoundCloud via OAuth 2.0 flow

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

#### Browse Public Rooms (v3)
`GET /api/v3/rooms/public?roomType=MUSIC&live=true&from=0&to=9`

Returns public rooms with password-protected admin controls. The password is
not required to listen. Private rooms are never included.

| Parameter | Meaning | Default |
| --- | --- | --- |
| `q` | Case-insensitive, literal substring of the room name; up to 100 characters | Empty |
| `roomType` | `MUSIC` or `WATCH` | `MUSIC` |
| `live` | `true` limits results to rooms with active listeners; `false` includes empty rooms | `false` |
| `from` | First row, zero-based | `0` |
| `to` | Last row, inclusive; at most 100 rooms per request | `from + 9` |

Rooms are ordered by listener count, playlist-item count, then ID, all descending.
Playlist-item counts include enabled providers. Listener counts use current presence;
cast receivers only count as one listener when there are no other listeners.

```json
{
  "rooms": [
    { "id": "electro", "name": "electro", "roomType": "MUSIC", "listenerCount": 3, "playlistItemCount": 24 }
  ],
  "from": 0,
  "to": 0,
  "total": 1,
  "count": 1
}
```

`to` identifies the last returned row. With no matches, `rooms` is an empty
array, `count` is zero, and `to` equals `from`. `total` remains available for
requests beyond the last page. Invalid filters and ranges return `400`.

Deploy migration 0025 from `zoff-music/vibes-migrator` for indexed room-name
search, then the backend, then clients using this endpoint. The index is
additive; this query also works before the migration is applied.

---

#### Create Room
`POST /api/v2/rooms`

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
    "allowDuplicates": false,
    "enabledSources": ["youtube", "soundcloud"],
    "onlyAdminAddPlaylistItems": false
  }
}
```

**Response:** `201 Created`

---

#### Suggest Room Name
`GET /api/v1/rooms/suggestions`

Returns an easy-to-say, currently available room name for the create-room form:

```json
{
  "name": "fly-banana-otter"
}
```

---

#### Check Whether a Room Exists
`HEAD /api/v1/rooms/{id}`

Returns `200 OK` when the room ID exists and `404 Not Found` when it is
available.

---

#### Get Room
`GET /api/v2/rooms/{id}`

---

#### Create Session (Join Room / Authenticate as Admin)
`POST /api/v2/rooms/{id}/sessions`

**Request Body:**
```json
{
  "nickname": "DJ Cool",
  "password": "optional-admin-password"
}
```

---

#### Update Room Settings (Admin/Host Only)
`PATCH /api/v2/rooms/{id}/settings`

---

### Queue Management

#### Get Playlist Items
`GET /api/v2/rooms/{id}/playlist-items`

---

#### Add Playlist Item to Queue
`POST /api/v2/rooms/{id}/playlist-items`

**Request Body:**
```json
{
  "sourceType": "youtube",
  "sourceId": "dQw4w9WgXcQ"
}
```

---

#### Remove Playlist Item from Queue
`DELETE /api/v2/rooms/{id}/playlist-items/{playlistItemId}`

---

#### Vote for Playlist Item
`POST /api/v2/rooms/{id}/playlist-items/{playlistItemId}`

---

### Playback Control

#### Get Playback State
`GET /api/v2/rooms/{id}/states`

---

#### Update Playback State
`PUT /api/v2/rooms/{id}/states`

**Request Body:**
```json
{
  "action": "play" | "pause" | "seek",
  "positionMs": 45000
}
```

---

#### Skip Playlist Item
`POST /api/v2/rooms/{id}/skips`

---

### Real-time Events (SSE)

#### Subscribe to Room Events
`GET /api/v3/rooms/{id}/events`

---

### Provider Search & Item Details

#### Search an Enabled Room Provider
`GET /api/v2/rooms/{id}/search/{provider}?q=query`

#### Get Item Details
`GET /api/v2/youtube/videos/{id}`

`GET /api/v2/soundcloud/items/{id}`

---

### OAuth & Authorization

#### Start OAuth Flow
`GET /api/v1/authorizations/{provider}`

#### Get/Refresh Access Token
`GET /api/v1/tokens/{provider}`

#### Get Enabled Providers
`GET /api/v1/providers`

---

### Global Admin Routes (Admin Only)

#### Admin Login
`POST /api/v1/admin/sessions`

**Request Body:**
```json
{
  "password": "global-admin-password"
}
```

---

#### Admin Logout
`DELETE /api/v1/admin/sessions`

---

#### List All Rooms
`GET /api/v2/admin/rooms`

---

#### Delete Room
`DELETE /api/v2/admin/rooms/{id}`

Returns `204 No Content` and publishes the refreshed room list through the admin
event stream.

---

#### Global Admin Events (SSE)
`GET /api/v2/admin/events`

---

## Data Types

### SourceType
`"youtube" | "soundcloud"`

### RoomMode
`"server" | "host"`

---

## Error Responses

Ordinary handler failures use `vibe.ErrorResponse`:

```json
{
  "error": "something went wrong"
}
```

There is no `code` field. Detailed internal errors are logged, not returned to the
caller. Swagger references this concrete schema instead of an arbitrary string
map, so the example and model show the actual `error` field.

Session, permission, and rate-limit middleware can reject requests with plain-text
bodies such as `unauthorized` or `forbidden`. Locally defined public
errors may include `namespace`, `error`, `message`,
and `propagate`, with the `X-preserve-error: 1` response header. Clients must not
assume every failed response is the standard JSON object.

A `204` response and every HEAD response have no body. OAuth `307` responses
redirect through the `Location` header. SSE endpoints return an ongoing sequence
of event frames whose `data` values depend on the event type; the Swagger stream
descriptions identify the payloads and replay cursor parameters.


Error handling rules: use `handleError` for handler failures. Unexpected errors
return the generic JSON message above; keep their causes in server logs. For an
intentional user-facing message, use `client.ErrorCodeWrapper` with `ResponseBody.Propagate` set to `true`.
Its internal `Err` is never serialized. Author safe response codes and messages
locally; never populate them from database, provider, or request diagnostics.
Wrappers without propagation and wrappers with invalid error statuses fall back
to the generic response. Upstream HTTP
error bodies cannot opt into propagation, even if they contain `propagate: true`.


The documentation UI lives in `static/swagger/index.html`, `theme.css`, and
`theme.js`. `DOCUMENTATION_DIRECTORY` defaults to `./static/swagger`; set it to an
absolute path when running the binary from another working directory. The
production image copies the static directory into `/app/static`.
