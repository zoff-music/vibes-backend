# Vibez Project Context

## Overview
Collaborative music queue with synchronized playback. Supports YouTube, Spotify, and SoundCloud.
Monorepo-style structure containing backend, frontend (with unified SSR), and database migrator.

## Key Directories

### `/` (Root)
- `backend/`: Go backend server
- `frontend/`: Bun workspace (React + Bun build system with SSR)
- `migrator/`: Go database migration tool
- `.env.sample`: Environment configuration template
- `docker-compose.yml`: Multi-service development environment
- `Caddyfile`: HTTPS reverse proxy configuration
- `Makefile`: Development and build commands

### `backend/`
- `cmd/server/`: Entrypoint (`main.go`)
- `client/`: External integrations (Database, YouTube, Spotify, SoundCloud, PubSub)
- `server/`: HTTP router, middleware, and handlers
- `server/internal/handler/`: Business logic handlers
- `vibe/`: Domain types and interfaces (pure Go, no dependencies)
- `config/`: Configuration management
- `monitoring/`: OpenTelemetry tracing, metrics, and telemetry
- `AGENTS.md`: Coding rules

### `frontend/`
- Workspace root with Bun workspaces
- `apps/platform/`: Main React application (Bun build system with SSR)
    - `src/api/`: `wiretyped` client and `yup` schemas
    - `src/stores/`: Zustand stores
    - `src/components/`: UI components
    - `server.tsx`: SSR server (Bun runtime)
    - `client.tsx`: Client hydration
    - `scripts/build.ts`: Custom Bun build script with hashing
- `apps/cast/`: Standalone Cast Receiver (Bun build system with SSR)
    - `server.tsx`: SSR server (Bun runtime)
    - `client.tsx`: Client hydration
    - `scripts/build.ts`: Custom Bun build script with hashing
- `packages/`: Shared packages
    - `api/`: Shared API client
    - `models/`: Shared types and schemas
    - `shared/`: Shared utilities (safeWrap, stores)
    - `player/`: Video player components
- `AGENTS.md`: Coding rules

## API
```
POST   /api/v1/rooms                    # Create room
GET    /api/v1/rooms/:id                # Get room
PATCH  /api/v1/rooms/:id/settings       # Update settings
POST   /api/v1/rooms/:id/sessions       # Join room
POST   /api/v1/rooms/:id/skips          # Skip song
PUT    /api/v1/rooms/:id/states         # Update playback state

GET    /api/v1/rooms/:id/songs          # Get queue
POST   /api/v1/rooms/:id/songs          # Add song
DELETE /api/v1/rooms/:id/songs/:songId  # Remove song
POST   /api/v1/rooms/:id/songs/:songId  # Vote/Reorder

GET    /api/v1/rooms/:id/events         # SSE stream

# Music Providers
GET    /api/v1/youtube/search           # Search YouTube
GET    /api/v1/youtube/videos/:id       # Get Video
GET    /api/v1/soundcloud/search        # Search SoundCloud
GET    /api/v1/soundcloud/tracks/:id    # Get Track
GET    /api/v1/spotify/search           # Search Spotify
GET    /api/v1/spotify/tracks/:id       # Get Track

# Authorization
GET    /api/v1/authorizations           # List connected providers
GET    /api/v1/authorizations/spotify   # Connect Spotify
GET    /api/v1/providers                # List enabled providers
```

### `migrator/`
- `main.go`: Entrypoint for running migrations
- `migrations/`: SQL migration files (`.up.sql`, `.down.sql`)
- **Convention**: SQL filenames are sequentially numbered (e.g. `0001_initial.up.sql`)
- **Automatic execution**: Runs automatically in Docker and local-dev

## Commands

### Local Development with HTTPS (Recommended)
```bash
make local-dev
```
*Starts Backend + Platform (SSR) + Cast Receiver (SSR) + Caddy Proxy*
*Access at https://localhost*

### Docker Development
```bash
make dev
```
*Full stack via Docker Compose with SSR-enabled services*

### Backend
```bash
cd backend
go run cmd/server/main.go
```
*Port: 8080*

### Frontend (Both Apps with SSR)
```bash
cd frontend
bun install
bun dev
```
*Platform: http://localhost:3000 (SSR), Cast: http://localhost:3001 (SSR)*

### Build Frontend Apps
```bash
cd frontend
bun run build
```
*Builds both apps with content hashing and manifest generation*

### Database Migrations
```bash
make migrate-up    # Run all up migrations
make migrate-down  # Run down migrations (1 step)
make migrate-down STEPS=3  # Run multiple down steps
```

## Critical Coding Rules (Summary)

### General
- **No `any` types**
- **No `try/catch`** (use `safeWrap`/`safeWrapAsync` from `@vibez/shared`)
- **Explicit types** everywhere
- **SSR Support**: Both platform and cast apps support server-side rendering

### Backend (Go)
- **No Service/Repo layers** - Keep it simple: Handler -> Client/Domain
- **No `New*` constructors** - Use struct literals
- **No inline error assignment** (`if err := ...`)
- **Wrap ALL errors** (`fmt.Errorf("doing X: %w", err)`)
- **No transactions** in DB client - Use atomic queries or CTEs
- **Use Prepared Statements**
- **Domain types in `vibe/`** - ALL business logic types here

### Frontend (TypeScript)
- **Use `wiretyped`** for ALL API calls / SSE - **No `fetch()` or `EventSource`**
- **Use `yup`** for validation
- **Tailwind CSS v4** for styling with dark mode support
- **Zustand** for state management
- **No inline styles**
- **Error handling**: `safeWrap`/`safeWrapAsync` from `@vibez/shared`

## Architecture Notes
- **Database**: SQLite (modernc.org/sqlite) - CGO-free
- **Real-time**: SSE (Server-Sent Events) for updates
- **Playback**: Synchronized via backend state + SSE
- **HTTPS Development**: Caddy provides automatic SSL certificates
- **SSR**: Both apps use unified Bun-based build system with SSR
- **Asset Management**: Content hashing with manifest-based resolution
- **Environment**: `.env.sample` template with comprehensive configuration
