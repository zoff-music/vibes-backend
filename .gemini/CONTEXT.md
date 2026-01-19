# Vibes Project Context

## Overview
Collaborative music queue with synchronized playback. Supports YouTube, Spotify, and SoundCloud.
Monorepo-style structure containing backend, frontend, and database migrator.

## Key Directories

### `/` (Root)
- `backend/`: Go backend server.
- `frontend/`: Bun workspace (React + Vite + TypeScript).
- `migrator/`: Go database migration tool.
- `.gemini/`: Context and help for AI agents.
- `CLAUDE.md`: High-level project commands and structure (human-facing).

### `backend/`
- `cmd/server/`: Entrypoint (`main.go`).
- `client/`: External integrations (Database, YouTube, PubSub). **All HTTP calls live here.**
- `server/`: HTTP router and middleware.
- `server/internal/handler/`: Business logic handlers.
- `vibe/`: Domain types and interfaces. **Pure Go, no dependencies.**
- `AGENTS.md`: Coding rules.

### `frontend/`
- Workspace root.
- `apps/platform/`: Main React application (Vite).
    - `src/api/`: `wiretyped` client and `yup` schemas.
    - `src/stores/`: Zustand stores.
    - `src/components/`: UI components.
- `packages/`: Shared packages.
    - `api/`: Shared API client.
    - `models/`: Shared types and schemas.
    - `shared/`: Shared utilities.
- `AGENTS.md`: Coding rules.

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
- `main.go`: Entrypoint for running migrations.
- `migrations/`: SQL migration files (`.up.sql`, `.down.sql`).
- **Convention**: SQL filenames are `Sequentially Numbered` (e.g. `0001_initial.up.sql`).

## Commands

### Backend
```bash
cd backend
go build ./cmd/server
./server
```
*Port: 8080*

### Local Development (HTTPS)
```bash
make local-dev
```
*Starts Backend + Frontend + Caddy Proxy.*
*Access at https://localhost*

### Frontend
```bash
cd frontend
bun install
bun dev
```
*Note: Runs `@vibez/mobile` via filter. URL: http://localhost:5173 (usually)*

### Migrator
```bash
cd migrator
export DATABASE_PATH=../vibes.db
go run main.go
```

## Critical Coding Rules (Summary)

### General
- **No `any` types.**
- **No `try/catch`** (use `safeWrap` or returns).
- **Explicit types** everywhere.

### Backend (Go)
- **No Service/Repo layers.** Keep it simple: Handler -> Client/Domain.
- **No `New*` constructors.** Use struct literals.
- **No inline error assignment** (`if err := ...`).
- **Wrap ALL errors** (`fmt.Errorf("doing X: %w", err)`).
- **No transactions** in DB client. Use atomic queries or CTEs.
- **Use Prepared Statements.**

### Frontend (TypeScript)
- **Use `wiretyped`** for ALL API calls / SSE. **No `fetch()` or `EventSource`.**
- **Use `yup`** for validation.
- **Tailwind CSS** for styling.
- **Zustand** for state management.
- **No inline styles.**

## Architecture Notes
- **Database**: SQLite (modernc.org/sqlite).
- **Real-time**: SSE (Server-Sent Events) for updates.
- **Playback**: Synchronized via backend state + SSE.
