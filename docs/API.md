# Zoff API Contract

High-level API contract for frontend-backend communication. The generated
Swagger UI at `/api/swagger/` provides the route and schema reference with Zoff
branding. It is served by the backend and loads the logo from
`https://zoff.me/logo.png`. The header matches the platform navigation, wordmark,
logo sizing, and pixel settings icon. Typography uses the frontend's `MSW98UI`
font assets under `/assets/` on the same origin. Keep the content-hashed font URLs
in `documentationpage.go` aligned with the frontend build when the fonts change;
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
Development: http://localhost:8080/api/v1
Production: https://zoff.me/api/v1
```

The incremental room stream is available separately at
`/api/v2/rooms/{id}/events`; paginated public-room browsing also has a
`/api/v2/rooms/public` endpoint. Most REST operations retain their v1 paths. See
[sessions and live updates](flows/sessions.md) for snapshot and replay behavior.

---

## Authentication

- **Room Admin**: Password-based authentication via `POST /rooms/{id}/sessions`
- **Regular User**: Anonymous session auto-created via middleware on first request
- **Global Admin**: Password-based authentication via `POST /admin/sessions`
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

#### Browse Public Rooms (v2)
`GET /api/v2/rooms/public?live=true&from=0&to=9`

Returns public rooms with password-protected admin controls. The password is
not required to listen. Private rooms are never included.

| Parameter | Meaning | Default |
| --- | --- | --- |
| `q` | Case-insensitive, literal substring of the room name; up to 100 characters | Empty |
| `live` | `true` limits results to rooms with active listeners; `false` includes empty rooms | `false` |
| `from` | First row, zero-based | `0` |
| `to` | Last row, inclusive; at most 100 rooms per request | `from + 9` |

Rooms are ordered by listener count, song count, then ID, all descending.
Song counts include enabled providers. Listener counts use current presence;
cast receivers only count as one listener when there are no other listeners.

```json
{
  "rooms": [
    { "id": "electro", "name": "electro", "listenerCount": 3, "songCount": 24 }
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
The v1 public list keeps its existing top-three live-room contract.

Deploy migration 0025 from `zoff-music/vibes-migrator` for indexed room-name
search, then the backend, then clients using this endpoint. The index is
additive; this query also works before the migration is applied.

---

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
    "allowDuplicates": false,
    "enabledSources": ["youtube", "soundcloud"],
    "onlyAdminAddSongs": false
  }
}
```

**Response:** `201 Created`

---

#### Suggest Room Name
`GET /rooms/suggestions`

Returns an easy-to-say, currently available room name for the create-room form:

```json
{
  "name": "fly-banana-otter"
}
```

---

#### Check Whether a Room Exists
`HEAD /rooms/{id}`

Returns `200 OK` when the room ID exists and `404 Not Found` when it is
available.

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
`GET /{youtube|soundcloud}/search?q=query`

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

#### Delete Room
`DELETE /admin/rooms/{id}`

Returns `204 No Content` and publishes the refreshed room list through the admin
event stream.

---

#### Global Admin Events (SSE)
`GET /admin/events`

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
