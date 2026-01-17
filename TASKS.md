# Vibez - V1 Task Breakdown

Detailed implementation tasks for MVP.

---

## Phase 1: Project Setup (Foundation) ✅ COMPLETE

### 1.1 Monorepo Initialization ✅
- [x] Initialize Bun workspace (`bunfig.toml`, `package.json`)
- [x] Create root `package.json` with workspace scripts
- [x] Create root `tsconfig.json` for shared settings
- [x] Create `.gitignore` with node_modules, dist, .expo, *.db, .env

### 1.2 Backend Setup ✅
- [x] Backend files already exist from boilerplate
- [x] `go.mod` already set to `github.com/zoff-music/vibes`
- [x] Verify `go build` works
- [x] SQLite database client in `client/database/` (note: `client/` not `clients/`)
- [x] SQLite driver: using `modernc.org/sqlite` (pure Go, no CGO)

### 1.3 Frontend Setup (Expo) ✅
- [x] Migrated monorepo from pnpm to bun
- [x] Created fresh Expo SDK 54 app with React 19.1.0
- [x] expo-router 6.0.21 configured with file-based routing
- [x] Created home screen (`app/index.tsx`)
- [x] Created create room screen (`app/room/create.tsx`)  
- [x] Created room view screen (`app/room/[id]/index.tsx`)
- [x] React Native StyleSheet with Vibez dark theme
- [x] Dependencies installed: yup, zustand, react-player
- [ ] Install wiretyped for API client (optional, can use fetch)
- [ ] Add NativeWind/Tailwind after babel config issues resolved

### 1.4 Shared Package ✅
- [x] Create `packages/shared/package.json`
- [x] Create `packages/shared/tsconfig.json`
- [x] Create `packages/shared/src/types.ts` (Room, Song, PlaybackState)
- [x] Create `packages/shared/src/constants.ts`

### 1.5 Design System Foundation ✅
- [x] Created Vibez dark theme colors (background, surface, primary purple, etc.)
- [x] Using plain React Native StyleSheet for now
- [x] Create `apps/mobile/src/utils/wrap.ts` (safeWrap/safeWrapAsync)

### 1.6 API Client Setup
- [ ] Create `apps/mobile/src/api/client.ts` - wiretyped or fetch-based client
- [ ] Create `apps/mobile/src/api/schemas/room.ts` (yup schemas)
- [ ] Create `apps/mobile/src/api/schemas/songs.ts`
- [ ] Create `apps/mobile/src/api/schemas/playback.ts`


---

## Phase 2: Backend Core (Domain & Database) ✅ COMPLETE

### 2.1 Domain Types (all in `vibe/vibe.go`) ✅
- [x] `Room` struct + `RoomSettings` struct
- [x] `RoomFetcher` interface (GetRoom)
- [x] `RoomCreator` interface (CreateRoom)
- [x] `RoomUpdater` interface (UpdateRoom)
- [x] `Song` struct
- [x] `SongsFetcher` interface (GetSongs, GetSong)
- [x] `SongsModifier` interface (AddSong, RemoveSong, ReorderSongs, GetNextSong)
- [x] `PlaybackState` struct
- [x] `PlaybackFetcher` interface (GetPlaybackState)
- [x] `PlaybackController` interface (UpsertPlaybackState)
- [x] `User` struct
- [x] `UserFetcher` interface (GetUser, GetUsersInRoom, CountUsersInRoom)
- [x] `UserManager` interface (CreateUser, UpdateUserLastSeen, RemoveUser, CleanupInactiveUsers)
- [x] `SkipVote` struct and interfaces
- [x] SSE `EventType` constants and `RoomEvent` struct

### 2.2 Database Schema ✅
- [x] `client/database/migrations.go` with schema:
  - `rooms` table
  - `songs` table with position ordering
  - `playback_state` table
  - `room_users` table
  - `skip_votes` table
  - Appropriate indexes

### 2.3 Database Client Implementation ✅
- [x] `client/database/database.go` - SQLite connection, Init, Close, prepared statements
- [x] `client/database/rooms.go` - GetRoom, CreateRoom, UpdateRoom
- [x] `client/database/songs.go` - GetSongs, GetSong, AddSong, RemoveSong, ReorderSongs, GetNextSong
- [x] `client/database/users.go` - User CRUD, counting, cleanup
- [x] `client/database/playback.go` - GetPlaybackState, UpsertPlaybackState
- [x] `client/database/skipvotes.go` - GetSkipVotes, HasUserVoted, AddSkipVote, ClearSkipVotes
- [x] Server integrated with database client

---

## Phase 3: Backend API (Handlers & SSE) ✅ COMPLETE

### 3.1 Room Handlers ✅
- [x] Create `server/internal/handler/rooms.go`:
  - [x] `CreateRoom()` - POST /api/v1/rooms
  - [x] `GetRoom()` - GET /api/v1/rooms/:id
  - [x] `UpdateRoom()` - PATCH /api/v1/rooms/:id
  - [x] `CreateSession()` - POST /api/v1/rooms/:id/sessions (join + optional admin auth)

### 3.2 Songs Handlers ✅
- [x] Create `server/internal/handler/songs.go`:
  - [x] `GetSongs()` - GET /api/v1/rooms/:id/songs
  - [x] `AddSong()` - POST /api/v1/rooms/:id/songs
  - [x] `RemoveSong()` - DELETE /api/v1/rooms/:id/songs/:songId
  - [x] `ReorderSongs()` - PATCH /api/v1/rooms/:id/songs/reorder

### 3.3 Room Actions Handler ✅
- [x] Create `server/internal/handler/action.go`:
  - [x] `RoomAction()` - POST /api/v1/rooms/:id/action
    - [x] Handle actions: play, pause, seek, skip, vote
    - [x] Discriminate by `action` field in request body

### 3.4 SSE Implementation ✅
- [x] Create SSE broker (can use existing internalpubsub):
  - [x] Room-scoped topics
  - [x] Subscribe/unsubscribe handling
- [x] Create `server/internal/handler/events.go`:
  - [x] `RoomEvents()` - GET /api/v1/rooms/:id/events (SSE endpoint)
- [x] Integrate SSE broadcast into handlers

### 3.5 Route Wiring ✅
- [x] Update `server/router.go` with all new routes
- [ ] Add room ID extraction middleware
- [x] Add user session middleware
- [ ] Add user session middleware

---

## Phase 4: Frontend Foundation (API & State) ✅ COMPLETE

### 4.1 SSE Hook ✅
- [x] Create `apps/mobile/src/hooks/useSSE.ts`:
  - [x] Connect to room events endpoint
  - [x] Parse event types
  - [x] Reconnection with exponential backoff
  - [x] Clean disposal

### 4.2 State Management ✅
- [x] Create `apps/mobile/src/stores/roomStore.ts` (zustand):
  - [x] Room data
  - [x] Users list
  - [x] Admin status
- [x] Create `apps/mobile/src/stores/queueStore.ts`:
  - [x] Queue items
  - [x] Add/remove optimistic updates
- [x] Create `apps/mobile/src/stores/playbackStore.ts`:
  - [x] Current item
  - [x] Playing state
  - [x] Position (synced)
  - [x] Calculated actual position

### 4.3 Custom Hooks ✅
- [x] Create `apps/mobile/src/hooks/useRoom.ts`:
  - [x] Fetch room on mount
  - [x] Subscribe to SSE
  - [x] Update store on events
- [x] Create `apps/mobile/src/hooks/useQueue.ts`:
  - [x] Fetch queue
  - [x] Add/remove mutations
- [x] Create `apps/mobile/src/hooks/usePlayback.ts`:
  - [x] Sync logic (server time vs client time)
  - [x] Control methods (play, pause, seek, skip)

---

## Phase 5: Frontend Screens

### 5.1 App Layout
- [ ] Create `apps/mobile/app/_layout.tsx`:
  - Unistyles provider
  - Store providers
  - Font loading
- [ ] Create `apps/mobile/app/index.tsx`:
  - Landing page
  - "Create Room" CTA
  - "Join Room" input

### 5.2 Create Room Flow
- [ ] Create `apps/mobile/app/room/create.tsx`:
  - Room name input
  - Optional admin password
  - Basic settings preview
  - Create button → navigate to room

### 5.3 Room View
- [ ] Create `apps/mobile/app/room/[id]/_layout.tsx`:
  - SSE connection
  - Bottom tab navigation (Now Playing, Queue, Settings)
- [ ] Create `apps/mobile/app/room/[id]/index.tsx` (Now Playing):
  - Blurred background from thumbnail
  - Current track info
  - Playback controls
  - Mini queue preview
- [ ] Create `apps/mobile/app/room/[id]/queue.tsx`:
  - Full queue list
  - Add to queue button
  - Swipe to remove (if admin)
  - Drag to reorder (if admin)
- [ ] Create `apps/mobile/app/room/[id]/settings.tsx`:
  - Admin password prompt (if not admin)
  - Settings toggles (skip, democratic skip, etc.)

### 5.4 Cast Display Mode
- [ ] Create `apps/mobile/app/cast/[id].tsx`:
  - Full-screen video player
  - Minimal UI overlay
  - Queue sidebar (toggleable)
  - Optimized for TV viewing

### 5.5 Add to Queue Modal
- [ ] Create `apps/mobile/src/components/queue/AddToQueueModal.tsx`:
  - YouTube URL paste
  - (Future: search functionality)
  - Preview before adding

---

## Phase 6: Video Player

### 6.1 Player Component
- [ ] Create `apps/mobile/src/components/player/VideoPlayer.tsx`:
  - react-player wrapper
  - YouTube support
  - Sync to playbackStore position
  - Handle play/pause/seek events
- [ ] Create `apps/mobile/src/components/player/PlayerControls.tsx`:
  - Play/pause button
  - Progress bar with seek
  - Skip button
  - Volume (where supported)
- [ ] Create `apps/mobile/src/components/player/NowPlayingBackground.tsx`:
  - Blurred thumbnail as background
  - Gradient overlay

### 6.2 Sync Logic
- [ ] Implement position drift correction in `usePlayback.ts`:
  - Calculate: `actualPosition = serverPosition + (Date.now() - serverTime)`
  - Apply correction if drift > 2 seconds
  - Smooth seek to avoid jarring jumps

---

## Phase 7: UI Components (Design System)

### 7.1 Primitives
- [ ] `apps/mobile/src/components/ui/Button.tsx`
- [ ] `apps/mobile/src/components/ui/Input.tsx`
- [ ] `apps/mobile/src/components/ui/Card.tsx`
- [ ] `apps/mobile/src/components/ui/Text.tsx` (styled variants)
- [ ] `apps/mobile/src/components/ui/IconButton.tsx`
- [ ] `apps/mobile/src/components/ui/Modal.tsx`
- [ ] `apps/mobile/src/components/ui/Skeleton.tsx`

### 7.2 Composite Components
- [ ] `apps/mobile/src/components/queue/QueueItem.tsx`
- [ ] `apps/mobile/src/components/queue/QueueList.tsx`
- [ ] `apps/mobile/src/components/room/RoomHeader.tsx`
- [ ] `apps/mobile/src/components/room/UserCount.tsx`

---

## Phase 8: Polish & Testing

### 8.1 Error Handling
- [ ] API error display component
- [ ] SSE reconnection UI feedback
- [ ] Empty states (no queue, no room found)
- [ ] Loading skeletons

### 8.2 Responsive Design
- [ ] Mobile breakpoint styles
- [ ] Tablet breakpoint styles
- [ ] Desktop breakpoint styles (cast mode optimized)

### 8.3 Testing
- [ ] Backend: handler unit tests
- [ ] Backend: database integration tests
- [ ] Frontend: component unit tests (vitest)
- [ ] E2E: create room → add song → play flow

### 8.4 Deployment Setup
- [ ] `docker-compose.yml` for local dev (backend + SQLite volume)
- [ ] `Dockerfile` for backend
- [ ] `Dockerfile.web` for bundled Expo web + backend
- [ ] EAS configuration for mobile builds

---

## Current Status

**Completed Phases:** 1, 2, 3
**Next Phase:** 4 (Frontend Foundation)

### Files Created/Modified in Phase 2:
- `backend/vibe/vibe.go` - All domain types and interfaces
- `backend/config/config.go` - SQLite config, removed PostgreSQL
- `backend/client/database/database.go` - Client setup with prepared statements
- `backend/client/database/migrations.go` - Schema with all tables
- `backend/client/database/rooms.go` - Room CRUD
- `backend/client/database/songs.go` - Song queue operations
- `backend/client/database/users.go` - User session management
- `backend/client/database/playback.go` - Playback state
- `backend/client/database/skipvotes.go` - Skip voting
- `backend/server/server.go` - Integrated database client

---

## Dependencies Graph

```
Phase 1 (Setup) ✅
    ↓
Phase 2 (Backend Core) ✅ ──────────────┐
    ↓                                   │
Phase 3 (Backend API) ← NEXT            │
    ↓                                   ↓
Phase 4 (Frontend Foundation) ←── packages/shared
    ↓
Phase 5 (Screens) + Phase 6 (Player) + Phase 7 (UI)
    ↓
Phase 8 (Polish)
```

Phases 5, 6, 7 can be worked on in parallel once Phase 4 is complete.
