# Vibez Project Context

## Overview
**Vibez** is a collaborative music queue application with synchronized playback, designed for shared listening experiences. Supports YouTube, Spotify, and SoundCloud with real-time synchronization via Server-Sent Events (SSE).

Monorepo structure with Go backend, React frontend (unified SSR), and database migrator.

## Key Directories

### `/` (Root)
- `backend/`: Go API server with SQLite database
- `frontend/`: Bun workspace with React apps and unified SSR
- `migrator/`: Go database migration tool
- `docs/`: API documentation and architecture guides
- `.env.sample`: Environment configuration template
- `docker-compose.yml`: Multi-service development environment
- `Caddyfile`: HTTPS reverse proxy configuration
- `Makefile`: Development and build commands

### `backend/`
- `cmd/server/`: Application entrypoint (`main.go`)
- `client/`: External service integrations
  - `database/`: SQLite operations (rooms, songs, playback, participants, skip voting, OAuth)
  - `youtube/`: YouTube Data API v3 client
  - `spotify/`: Spotify Web API client with OAuth
  - `soundcloud/`: SoundCloud API client with OAuth
  - `internalpubsub/`: SSE broadcasting system
- `server/`: HTTP server, routing, and middleware
  - `internal/handler/`: HTTP request handlers (rooms, songs, playback, search, auth)
  - `internal/middleware/`: Session, CORS, tracing middleware
  - `internal/helper/`: Utility functions (session, cookies, slugify)
- `vibe/`: Domain types and interfaces (ALL business logic types)
  - `rooms.go`: Room, RoomSettings types and interfaces
  - `songs.go`: Song, queue management types and interfaces
  - `playback.go`: PlaybackState, room actions and interfaces
  - `events.go`: SSE event types and interfaces
  - `participants.go`: User session types and interfaces
  - `skip.go`: Skip voting interfaces
  - `authorization.go`: OAuth types and interfaces
- `config/`: Configuration management
- `monitoring/`: OpenTelemetry tracing, Prometheus metrics
- `AGENTS.md`: Non-negotiable coding conventions

### `frontend/`
Bun workspace with unified SSR build system:
- `apps/platform/`: Main collaborative music queue interface
  - `src/components/`: UI components (ui, player, queue, cast, room)
  - `src/hooks/`: Custom React hooks (useRoom, useQueue, usePlayback, useSSE)
  - `src/stores/`: Zustand stores (roomStore, queueStore, castStore, themeStore)
  - `src/pages/`: Route components (Home, CreateRoom, RoomView, Callback)
  - `src/services/`: Business logic (castManager)
  - `server.tsx`: SSR server (Bun runtime)
  - `client.tsx`: Client hydration
  - `scripts/build.ts`: Custom Bun build script with content hashing
- `apps/cast/`: Chromecast receiver application
  - `server.tsx`: SSR server (Bun runtime)
  - `client.tsx`: Client hydration
  - `scripts/build.ts`: Custom Bun build script with content hashing
- `packages/`: Shared code across apps
  - `api/`: wiretyped API client with Yup validation
  - `models/`: Shared TypeScript types and Yup schemas
  - `shared/`: Utilities (safeWrap, playbackStore, constants)
  - `player/`: Video player components (SpotifyPlayer, SoundCloudPlayer, VideoPlayer)
- `AGENTS.md`: TypeScript and React conventions

### `migrator/`
- `main.go`: CLI tool for running database migrations
- `migrations/`: SQL migration files (sequential numbering)
  - `0001_initial_schema.up.sql`: Core tables (rooms, songs, playback_state, etc.)
  - `0002_add_song_votes_and_constraints.up.sql`: Skip voting system
  - `0003_add_room_modes.up.sql`: Server vs Host mode support
  - And more... (11 migrations total)
- **Convention**: `NNNN_description.up.sql` / `NNNN_description.down.sql`

## API Endpoints

### Room Management
```
POST   /api/v1/rooms                    # Create room
GET    /api/v1/rooms/{id}               # Get room details
PATCH  /api/v1/rooms/{id}/settings      # Update room settings (admin)
POST   /api/v1/rooms/{id}/sessions      # Authenticate as admin
```

### Queue Management
```
GET    /api/v1/rooms/{id}/songs         # Get queue
POST   /api/v1/rooms/{id}/songs         # Add song to queue
DELETE /api/v1/rooms/{id}/songs/{songId} # Remove song
PATCH  /api/v1/rooms/{id}/songs/{songId} # Reorder song
POST   /api/v1/rooms/{id}/songs/{songId} # Vote for song
```

### Playback Control
```
GET    /api/v1/rooms/{id}/states        # Get playback state
PUT    /api/v1/rooms/{id}/states        # Update playback (play/pause/seek)
POST   /api/v1/rooms/{id}/skips         # Skip song (force or vote)
```

### Real-time Events
```
GET    /api/v1/rooms/{id}/events        # SSE stream for live updates
```

### Music Search & Track Details
```
GET    /api/v1/youtube/search?q=query   # Search YouTube
GET    /api/v1/youtube/videos/{id}      # Get YouTube video details
GET    /api/v1/spotify/search?q=query   # Search Spotify
GET    /api/v1/spotify/tracks/{id}      # Get Spotify track details
GET    /api/v1/soundcloud/search?q=query # Search SoundCloud
GET    /api/v1/soundcloud/tracks/{id}   # Get SoundCloud track details
```

### OAuth & Authorization
```
GET    /api/v1/authorizations/{provider} # Start OAuth flow
GET    /api/v1/callbacks/{provider}     # OAuth callback
GET    /api/v1/tokens/{provider}        # Get/refresh access token
GET    /api/v1/providers               # List enabled providers
```

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
- Host is determined by `room.HostID`
- Great for parties with a DJ
- Democratic skip voting disabled (host decides)

## Commands

### Local Development with HTTPS (Recommended)
```bash
make local-dev
```
**Starts**: Backend + Platform SSR + Cast SSR + Caddy Proxy  
**Access**: https://localhost (Platform), https://localhost/casting/receiver/ (Cast)

### Docker Development
```bash
make dev
```
**Full stack via Docker Compose with SSR-enabled services**

### Manual Development
```bash
# Backend
cd backend && go run cmd/server/main.go  # Port 8080

# Frontend (both apps with SSR)
cd frontend && bun dev  # Platform: 3000, Cast: 3001

# Individual apps
cd frontend/apps/platform && bun run dev
cd frontend/apps/cast && bun run dev
```

### Build & Deploy
```bash
# Build frontend apps (with content hashing)
cd frontend && bun run build

# Build backend binary
make build

# Docker containers
make docker
```

### Database Operations
```bash
make migrate-up           # Run all up migrations
make migrate-down STEPS=1 # Run down migrations (1 step)
make build-migrator       # Build migrator binary
```

### Testing & Quality
```bash
make test                         # Backend tests
cd frontend && bun run typecheck  # TypeScript checking
cd frontend && bun run lint       # Biome linting
make gosec                        # Security scanning
make govulncheck                  # Vulnerability checking
```

## Architecture Details

### Database Schema
- **SQLite** with modernc.org/sqlite (CGO-free)
- **Core Tables**: rooms, room_settings, songs, playback_state, room_users
- **Skip Voting**: skip_votes table with democratic threshold logic
- **OAuth Integration**: auth_tokens, access_tokens, pending_oauth_state
- **Session Tracking**: room_participants for active user management

### Real-time Synchronization
- **SSE (Server-Sent Events)** for live updates
- **Event Types**: playback_update, song_added, song_removed, users_update, etc.
- **Client Deduplication**: Events from triggering user are filtered out
- **Connection Management**: Reference counting with grace period cleanup

### Frontend State Management
- **Zustand Stores**: roomStore, queueStore, playbackStore, castStore, themeStore
- **Custom Hooks**: useRoom, useQueue, usePlayback, useSSE for data fetching
- **Optimistic Updates**: Immediate UI feedback with rollback on error
- **SSR Support**: Server-side rendering with client hydration

### Build System
- **Unified SSR**: Both apps use identical Bun-based build system
- **Content Hashing**: All assets use content-based hashing for cache busting
- **Manifest Generation**: JSON manifest maps logical names to hashed filenames
- **Asset Resolution**: SSR servers use manifest for correct asset URLs

## Critical Coding Rules (Summary)

### General
- **No `any` types** - Use explicit TypeScript types
- **No `try/catch`** - Use `safeWrap`/`safeWrapAsync` from `@vibez/shared`
- **SSR Support** - Both apps support server-side rendering
- **Dark Mode** - All components support light/dark themes

### Backend (Go)
- **No Service/Repository layers** - Direct client usage via interfaces
- **No `New*` constructors** - Use struct literals
- **No inline error assignment** - Never `if err := ...; err != nil {}`
- **Wrap ALL errors** - `fmt.Errorf("error doing X: %w", err)`
- **No transactions** - Use atomic queries only
- **Prepared Statements** - 1:1 naming convention
- **Domain types in `vibe/` directory** - ALL business logic types

### Frontend (TypeScript)
- **Use wiretyped** - ALL API calls via wiretyped client (no `fetch()`)
- **Use Yup validation** - All API schemas validated
- **Tailwind CSS v4** - Styling with dark mode support
- **Zustand** - State management
- **Error handling** - `safeWrap`/`safeWrapAsync` utilities only
- **Named exports** - No default exports for components

## Environment Variables

### Backend (.env)
```bash
# Required
PORT=8080
DATABASE_PATH=./data/vibes.db
YOUTUBE_API_KEY=your-youtube-api-key

# Optional OAuth
SPOTIFY_CLIENT_ID=your-spotify-client-id
SPOTIFY_CLIENT_SECRET=your-spotify-client-secret
SOUNDCLOUD_CLIENT_ID=your-soundcloud-client-id
SOUNDCLOUD_CLIENT_SECRET=your-soundcloud-client-secret

# Optional
LOG_LEVEL=info
CORS_ORIGINS=https://localhost,https://yourdomain.com
```

### Frontend
```bash
VITE_API_URL=https://localhost/api/v1  # For local-dev
# or
VITE_API_URL=http://localhost:8080/api/v1  # For manual dev
```
