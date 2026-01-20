# Vibez

Collaborative music queue with synchronized playback. Supports YouTube, Spotify, and SoundCloud.

## Commands

```bash
# Local Development (Recommended)
make local-dev
# Runs backend + platform + cast receiver + Caddy (HTTPS).
# Platform: https://localhost
# Cast Receiver: https://localhost/casting/receiver/
# API: https://localhost/api

# Backend Manual
cd backend && go build ./cmd/server && ./server

# Frontend
cd frontend && bun install && bun dev

# Migrator
cd migrator && go run main.go
```

## Stack

| Layer | Tech |
|-------|------|
| Frontend | React 19 + Vite + Tailwind + TypeScript |
| Backend | Go + SQLite (`modernc.org/sqlite`) |
| Migrator | Go + SQLite |
| Real-time | Server-Sent Events |
| Package Manager | Bun |

## Structure

```
backend/
├── AGENTS.md              # Go coding rules (READ FIRST)
├── cmd/server/            # Entrypoint
├── client/                # DB, YouTube, PubSub clients
├── server/internal/handler/   # HTTP handlers
└── vibe/vibe.go           # ALL domain types

frontend/
├── apps/platform/         # React web app (Main)
│   ├── src/components/    # UI components
│   ├── src/stores/        # Zustand stores
│   └── src/pages/         # Route components
├── apps/cast/             # Cast Receiver App (Standalone)
│   ├── src/App.tsx        # Receiver Entrypoint
│   └── vite.config.ts     # Vite Config (Port 3001)
└── packages/              # Shared packages
    ├── api/               # API client
    ├── models/            # Types & Schemas
    └── shared/            # Utilities

migrator/                  # Database migration tool
├── main.go                # Entrypoint
└── migrations/            # SQL migration files
```

## API

```
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

# Providers & Auth
GET    /api/v1/youtube/search           # Search YouTube
GET    /api/v1/soundcloud/search        # Search SoundCloud
GET    /api/v1/spotify/search           # Search Spotify
GET    /api/v1/authorizations           # User auths
GET    /api/v1/providers                # Configured providers
```

Full contract: `docs/API.md`

## Environment

```bash
PORT=8080
DATABASE_PATH=./vibez.db
YOUTUBE_API_KEY=required
VITE_API_URL=http://127.0.0.1:8080
```

## Coding Rules

Read before writing code:
- **Backend:** `backend/AGENTS.md`
- **Frontend:** `frontend/AGENTS.md`

Critical rules:
- No `any` type, no `try/catch` (use `safeWrap`)
- All errors wrapped with context
- No inline error assignments (`if err := ...`)
- HTTP only through clients, consumed via interfaces
- Domain types in `vibe/vibe.go`, not in handlers
- **Providers**: YouTube, Spotify, SoundCloud supported for search/auth
