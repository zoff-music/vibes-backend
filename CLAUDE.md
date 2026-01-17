# CLAUDE.md - Vibez

> Shared music queue with synchronized playback and casting support.

---

## Project Overview

**Vibez** is a collaborative music queue application where:
- Anyone can join a room via URL and add videos/songs to the queue
- All connected devices see the same playback position (synced to the second)
- One device can cast to Chromecast/AirPlay while others control the queue
- Room admins have password-protected settings control
- Extensible to Spotify/Soundcloud sources and addons (visualizers, shot timers)

---

## Architecture Decisions

### Stack
| Layer | Technology | Notes |
|-------|------------|-------|
| Frontend | Expo SDK 54 + React 19 + expo-router 6 + TypeScript | Mobile-first responsive design |
| Package Manager | Bun | Migrated from pnpm for faster installs |
| Styling | React Native StyleSheet (plain) | NativeWind pending babel config fixes |
| Network | fetch or wiretyped + yup | Type-safe API calls with validation |
| Backend | Go | Following existing AGENTS.md patterns |
| Database | SQLite | Single-file, easy deployment |
| Real-time | Server-Sent Events (SSE) via wiretyped | Server pushes state; clients send via HTTP |
| Video Player | react-player | Multi-source support for future expansion |
| Casting | Chromecast + AirPlay SDKs | Native casting protocols |
| Deployment | Docker (backend + bundled web), EAS (mobile) | |

## Build & Run Commands

-   Install Dependencies: `bun install`
-   Dev (All): `bun dev`
-   Dev (Mobile): `bun run --filter @vibez/mobile dev:web`
-   Build (Mobile): `bun run --filter @vibez/mobile build`
-   Backend (Manual): `bun backend`
-   Type Check: `bun typecheck`

### Monorepo Structure
```
vibez/
├── CLAUDE.md                    # This file
├── AGENTS.md                    # Coding conventions (from boilerplate)
├── TASKS.md                     # Task breakdown and progress
├── package.json                 # Root workspace config (using bun)
├── bunfig.toml                  # Bun workspace config
├── docker-compose.yml           # Local dev environment
│
├── backend/                     # Go backend
│   ├── AGENTS.md               # Backend-specific conventions
│   ├── go.mod
│   ├── cmd/server/main.go
│   ├── config/
│   ├── client/                 # External integrations & clients
│   │   ├── database/
│   │   │   ├── database.go     # SQLite client + Init
│   │   │   ├── migrations.go   # Schema migrations
│   │   │   ├── rooms.go        # Room CRUD
│   │   │   ├── songs.go        # Song queue operations
│   │   │   ├── users.go        # User session management
│   │   │   ├── playback.go     # Playback state
│   │   │   └── skipvotes.go    # Skip voting
│   │   ├── internalpubsub/     # In-memory pubsub for SSE
│   │   ├── error.go
│   │   └── http.go
│   │
│   ├── monitoring/             # DO NOT MODIFY
│   ├── internalerror/
│   ├── server/
│   │   ├── server.go
│   │   ├── router.go
│   │   └── internal/
│   │       ├── handler/        # HTTP handlers
│   │       ├── middleware/
│   │       └── event/          # Internal pubsub events
│   │
│   └── vibe/                   # ALL domain types & interfaces
│       └── vibe.go             # Room, Song, Playback, User types
│
├── apps/
│   └── mobile/                 # Expo SDK 54 app (React 19)
│       ├── package.json        # @vibez/mobile
│       ├── app.json            # Expo config
│       ├── tsconfig.json
│       ├── app/                # Expo Router file-based routing
│       │   ├── _layout.tsx     # Root layout with dark theme
│       │   ├── index.tsx       # ✅ Home/landing (Create/Join room)
│       │   └── room/
│       │       ├── create.tsx  # ✅ Create room screen
│       │       └── [id]/
│       │           └── index.tsx # ✅ Room view (placeholder)
│       │
│       └── src/                # (To be created)
│           ├── api/            # API client (fetch or wiretyped)
│           ├── components/     # Reusable components
│           │   ├── ui/         # Design system primitives
│           │   ├── player/     # Video player components
│           │   └── queue/      # Queue list components
│           ├── hooks/
│           │   ├── useRoom.ts
│           │   ├── useQueue.ts
│           │   ├── usePlayback.ts
│           │   └── useSSE.ts   # Server-Sent Events hook
│           ├── stores/         # Zustand stores
│           │   ├── roomStore.ts
│           │   ├── queueStore.ts
│           │   └── playbackStore.ts
│           └── utils/
│               └── wrap.ts     # safeWrap/safeWrapAsync
│
└── packages/
    └── shared/                 # Shared types between frontend/backend
        ├── package.json
        └── src/
            ├── types.ts        # Room, QueueItem, PlaybackState types
            └── constants.ts    # Shared constants
```

---

## Domain Model

### Core Entities

```typescript
// Room
interface Room {
  id: string;              // URL-friendly slug (e.g., "friday-vibes")
  name: string;
  createdAt: Date;
  adminPasswordHash?: string;  // Optional password protection
  settings: RoomSettings;
}

interface RoomSettings {
  skipAllowed: boolean;          // Can users skip current track?
  democraticSkip: boolean;       // Require vote threshold to skip?
  skipVoteThreshold: number;     // % of users needed to skip (0.5 = 50%)
  maxContinuousAdds: number;     // Max songs one user can add in a row
  removeOnPlay: boolean;         // Remove song from queue when played?
  loopQueue: boolean;            // Loop back to start when queue ends?
  allowDuplicates: boolean;      // Allow same song twice in queue?
}

// Song (queue item)
interface Song {
  id: string;
  roomId: string;
  sourceType: 'youtube' | 'spotify' | 'soundcloud';
  sourceId: string;              // e.g., YouTube video ID
  title: string;
  artist?: string;
  thumbnailUrl: string;
  duration: number;              // seconds
  addedBy: string;               // User identifier
  addedAt: Date;
  position: number;              // Order in queue
}

// Playback State (synced across all clients)
interface PlaybackState {
  roomId: string;
  currentSongId: string | null;
  isPlaying: boolean;
  positionMs: number;            // Current playback position in ms
  updatedAt: Date;               // Server timestamp for sync
  serverTimeMs: number;          // Server's current time for drift calc
}

// User (session-based, no accounts for V1)
interface RoomUser {
  id: string;                    // Session ID
  roomId: string;
  nickname?: string;
  isAdmin: boolean;
  joinedAt: Date;
  lastSeenAt: Date;
}
```

---

## Real-Time Sync Architecture

### SSE Event Types (Server → Client)

```typescript
type SSEEvent =
  | { type: 'room:state'; data: Room }
  | { type: 'songs:update'; data: Song[] }
  | { type: 'playback:sync'; data: PlaybackState }
  | { type: 'users:update'; data: RoomUser[] }
  | { type: 'skip:vote'; data: { userId: string; songId: string } };
```

### Playback Sync Strategy

1. **Server is source of truth**: Backend tracks `positionMs` and `updatedAt`
2. **Client calculates drift**: `actualPosition = positionMs + (clientNow - serverTimeMs)`
3. **Periodic heartbeat**: Server sends `playback:sync` every 5 seconds
4. **Seek events**: Immediate sync push on seek/play/pause
5. **Cast device**: Receives same SSE stream, renders full-screen player

### HTTP Endpoints (Client → Server)

```
POST   /api/v1/rooms                    # Create room
GET    /api/v1/rooms/:id                # Get room details
PATCH  /api/v1/rooms/:id                # Update room (admin)
POST   /api/v1/rooms/:id/sessions       # Join room (+ admin auth if password provided)

GET    /api/v1/rooms/:id/songs          # Get song queue
POST   /api/v1/rooms/:id/songs          # Add song to queue
DELETE /api/v1/rooms/:id/songs/:songId  # Remove song from queue
PATCH  /api/v1/rooms/:id/songs/reorder  # Reorder songs (admin)

POST   /api/v1/rooms/:id/action         # Room actions (play, pause, seek, skip, vote)
                                        # { action: "play" | "pause" | "seek" | "skip" | "vote", ... }

GET    /api/v1/rooms/:id/events         # SSE endpoint
```

---

## Design System (Unistyles v3)

### Theme Tokens

```typescript
const theme = {
  colors: {
    // Dark theme base
    background: '#0a0a0a',
    surface: '#141414',
    surfaceElevated: '#1c1c1c',

    // Accent (vibrant purple/pink gradient feel)
    primary: '#a855f7',
    primaryMuted: '#7c3aed',
    secondary: '#ec4899',

    // Text
    text: '#fafafa',
    textMuted: '#a1a1aa',
    textInverse: '#0a0a0a',

    // Semantic
    success: '#22c55e',
    warning: '#f59e0b',
    error: '#ef4444',
  },

  spacing: {
    xs: 4,
    sm: 8,
    md: 16,
    lg: 24,
    xl: 32,
    xxl: 48,
  },

  radii: {
    sm: 4,
    md: 8,
    lg: 12,
    xl: 16,
    full: 9999,
  },

  fonts: {
    body: 'Inter',
    heading: 'Inter',
    mono: 'JetBrains Mono',
  },

  fontSizes: {
    xs: 12,
    sm: 14,
    md: 16,
    lg: 18,
    xl: 24,
    xxl: 32,
    xxxl: 48,
  },
};
```

### Visual Style
- **Dark lightness**: Near-black backgrounds with subtle elevation
- **"Now Playing" blur**: Current track artwork as blurred background
- **Glass morphism**: Subtle frosted glass effects on overlays
- **Smooth animations**: Spring-based transitions for queue reordering

---

## Addon System

### Addon Interface

```typescript
interface Addon {
  id: string;
  name: string;
  description: string;
  component: React.ComponentType<AddonProps>;
  settings?: AddonSetting[];
}

interface AddonProps {
  playbackState: PlaybackState;
  currentSong: Song | null;
  roomSettings: RoomSettings;
}

interface AddonSetting {
  key: string;
  label: string;
  type: 'boolean' | 'number' | 'select';
  options?: { label: string; value: string }[];
  default: unknown;
}
```

### V1 Addons
1. **Visualizer**: Audio-reactive visualization instead of video
2. **Shot Timer**: Periodic "take a shot" overlay with countdown

---

## Source Plugin Architecture

For future Spotify/Soundcloud support:

```typescript
interface SourcePlugin {
  id: 'youtube' | 'spotify' | 'soundcloud';
  name: string;

  // Search & metadata
  search(query: string): Promise<SearchResult[]>;
  getMetadata(sourceId: string): Promise<MediaMetadata>;

  // Playback URL resolution (some sources need auth)
  getPlaybackUrl(sourceId: string): Promise<string>;

  // Auth (for Spotify/Soundcloud)
  requiresAuth: boolean;
  authenticate?(): Promise<void>;
}
```

---

## V1 Scope (MVP)

### Included
- YouTube video queue only
- Public rooms with URL sharing
- Optional admin password
- Synchronized playback (all see same position)
- Cast display mode (full-screen for TV)
- Basic room settings (skip, democratic skip, max adds)
- Queue add/remove/reorder
- Mobile-first responsive web UI

### Excluded (V2+)
- User accounts / OAuth (Google, Spotify)
- Spotify source
- Soundcloud source
- Visualizer addon
- Shot timer addon
- Chromecast SDK integration (V1: just open URL on TV)
- AirPlay integration

---

```

### Environment Variables
```bash
# Backend
PORT=8080
DATABASE_PATH=./vibez.db

# Frontend
EXPO_PUBLIC_API_URL=http://localhost:8080
```

---

## Conventions

### Backend
See `backend/AGENTS.md` - strictly follow:
- No service/repository patterns
- Handlers return `http.HandlerFunc` with interface injection
- Clients live in `clients/`, all domain types live in `vibe/vibe.go`
- Verbose error wrapping, no inline errors
- Tracing via middleware only

### Frontend
See `apps/mobile/AGENTS.md` - strictly follow:
- No `any` type
- No `try/catch` - use `safeWrap`/`safeWrapAsync`
- Use wiretyped + yup for all network calls
- Component-first architecture

---

## Implementation Progress

See `TASKS.md` for detailed task breakdown with checkboxes.

### ✅ Phase 1: Project Setup - COMPLETE
- Bun workspace initialized
- Expo app with expo-router configured
- Unistyles v3 theme set up
- Shared types package created
- wiretyped API client configured with yup schemas

### ✅ Phase 2: Backend Core - COMPLETE
- SQLite schema with all tables (rooms, songs, playback_state, room_users, skip_votes)
- All domain types in `vibe/vibe.go`
- Database client with prepared statements for all operations
- Server integrated with database client

### 🚧 Phase 3: Backend API - NEXT
- Room handlers (create, get, update, join)
- Songs handlers (get, add, remove, reorder)
- Playback handlers (room actions endpoint)
- SSE endpoint with event broadcasting

### Remaining Phases 4-8
See `TASKS.md` for detailed breakdown.

---

## Important Implementation Notes

### Backend Folder Structure
- **`client/` not `clients/`**: The boilerplate uses `client/` singular
- **`vibe/vibe.go`**: All domain types and interfaces live here
- **Database files**: `rooms.go`, `songs.go`, `users.go`, `playback.go`, `skipvotes.go`

### SQLite Driver
Using `modernc.org/sqlite` (pure Go) instead of `mattn/go-sqlite3` (CGO required):
- No CGO dependencies
- Easier cross-compilation
- Already in go.mod

### wiretyped API Usage
```typescript
import { RequestClient, RequestDefinitions } from 'wiretyped';

const endpoints = {
  '/rooms': {
    post: { body: createRoomRequestSchema, response: roomSchema },
  },
  '/rooms/{id}': {
    get: { response: roomSchema },
    post: { body: roomActionSchema }, // Actions: play, pause, seek, skip, vote
  },
  // ...
} satisfies RequestDefinitions;

export const api = new RequestClient({
  hostname: API_URL,
  baseUrl: API_BASE_PATH,
  endpoints,
  validation: true,
});
```

### Database Prepared Statement Pattern
Following AGENTS.md strictly:
```go
// prepareXStmt() prepares XStatement
// X() executes XStatement
func (c *Client) prepareGetRoomStmt() error {
    stmt, err := c.DB.Prepare(`SELECT ... FROM rooms WHERE id = ?`)
    if err != nil {
        return fmt.Errorf("error preparing GetRoomStatement: %w", err)
    }
    c.GetRoomStatement = stmt
    return nil
}

func (c *Client) GetRoom(ctx context.Context, id string) (*vibe.Room, error) {
    span, ctx := opentracing.StartSpanFromContext(ctx, "GetRoom")
    defer span.Finish()
    // ... use c.GetRoomStatement
}
```

### Room Actions Endpoint Design
All playback controls go through single endpoint with action discriminator:
```
POST /api/v1/rooms/:id/action
{
  "action": "play" | "pause" | "seek" | "skip" | "vote",
  "positionMs": 1234  // only for seek
}
```

### Config Changes from Boilerplate
- Removed PostgreSQL config
- Added `DatabasePath` for SQLite (default: `./data/vibes.db`)
- Added `MaxNameLength`, `MaxQueueLength`
- Added `UserInactivityTimeout`
- Made `OtelEndpoint` optional (defaults to empty)

---

## Open Questions for Future

1. **User identity**: Session-based for V1. How to handle nickname persistence?
2. **Room cleanup**: Archive inactive rooms after X days?
3. **Rate limiting**: How aggressive for queue additions?
4. **CORS**: Need to configure for web deployment
5. **YouTube API quotas**: Cache metadata aggressively
