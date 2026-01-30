# Vibez

Collaborative music queue with synchronized playback. Supports YouTube, Spotify, and SoundCloud.

## Commands

```bash
# Local Development with HTTPS (Recommended)
make local-dev
# Runs backend + platform + cast receiver + Caddy (HTTPS)
# Platform: https://localhost
# Cast Receiver: https://localhost/casting/receiver/
# API: https://localhost/api

# Docker Development
make dev
# Full stack via Docker Compose

# Backend Manual
cd backend && go run cmd/server/main.go

# Frontend (Both Apps with SSR)
cd frontend && bun install && bun dev

# Database Migrations
make migrate-up    # Run all up migrations
make migrate-down  # Run down migrations
```

## Stack

| Layer | Tech |
|-------|------|
| Frontend | React 19 + Vite + Tailwind CSS v3 + TypeScript + SSR |
| Backend | Go + SQLite (`github.com/mattn/go-sqlite3`) + OpenTelemetry |
| Migrator | Go + SQLite |
| Real-time | Server-Sent Events (SSE) |
| Package Manager | Bun |
| Proxy | Caddy (HTTPS, SSL certificates) |
| Error Handling | safeWrap/safeWrapAsync utilities |

## Structure

```
backend/
├── AGENTS.md              # Go coding rules (READ FIRST)
├── cmd/server/            # Entrypoint
├── client/                # External service clients
│   ├── database/          # SQLite operations
│   ├── youtube/           # YouTube API client
│   ├── soundcloud/        # SoundCloud API client
│   ├── spotify/           # Spotify API client
│   └── internalpubsub/    # SSE broadcasting
├── server/                # HTTP server and routing
│   ├── internal/handler/  # HTTP handlers
│   ├── internal/middleware/ # HTTP middleware
│   └── internal/helper/   # Utility functions
├── vibe/                  # Domain types and interfaces (CRITICAL)
├── config/                # Configuration management
└── monitoring/            # Telemetry, tracing, metrics

frontend/
├── apps/platform/         # React web app (Main, SSR-enabled)
│   ├── src/components/    # UI components
│   ├── src/stores/        # Zustand stores
│   ├── src/pages/         # Route components
│   ├── server.tsx         # SSR server
│   └── client.tsx         # Client hydration
├── apps/cast/             # Cast Receiver App (SSR-enabled)
│   ├── src/App.tsx        # Receiver Entrypoint
│   ├── server.tsx         # SSR server
│   └── client.tsx         # Client hydration
└── packages/              # Shared packages
    ├── api/               # wiretyped API client
    ├── models/            # Types & Yup schemas
    ├── shared/            # Utilities (safeWrap, stores)
    └── player/            # Video player components

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
# Backend Configuration (.env)
PORT=8080
DATABASE_PATH=./data/db/vibes.db
YOUTUBE_API_KEY=required
SPOTIFY_CLIENT_ID=optional
SPOTIFY_CLIENT_SECRET=optional
SOUNDCLOUD_CLIENT_ID=optional
SOUNDCLOUD_CLIENT_SECRET=optional

# OpenTelemetry (optional)
OTEL_SAMPLER_PARAM=1
OTEL_EXPORTER_TIMEOUT=1s

# Cast Configuration
CAST_APP_ID=1FAF5D9F
CAST_RECEIVER_URL=https://zoff.me/casting/receiver

# Frontend URLs
FRONTEND_URL=https://localhost
VITE_API_URL=https://localhost/api
```

Copy `.env.sample` to `.env` and configure your API keys.

## Coding Rules

Read before writing code:
- **Backend:** `backend/AGENTS.md`
- **Frontend:** `frontend/AGENTS.md`

Critical rules:
- No `any` type, no `try/catch` (use `safeWrap`/`safeWrapAsync`)
- All errors wrapped with context
- No inline error assignments (`if err := ...`)
- HTTP only through clients, consumed via interfaces
- Domain types in `vibe/vibe.go`, not in handlers
- **SSR**: Both platform and cast apps support server-side rendering
- **Error Handling**: Use `safeWrap`/`safeWrapAsync` from `@vibez/shared`
- **Providers**: YouTube, Spotify, SoundCloud supported for search/auth
