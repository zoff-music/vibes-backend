# Vibez

Collaborative music queue with synchronized playback. YouTube-only for V1.

## Commands

```bash
# Backend
cd backend && go build ./cmd/server && ./server

# Frontend
cd frontend && bun install && bun dev
```

## Stack

| Layer | Tech |
|-------|------|
| Frontend | React 19 + Vite + Tailwind + TypeScript |
| Backend | Go + SQLite (`modernc.org/sqlite`) |
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

frontend/apps/mobile/      # React web app
├── src/api/               # wiretyped client + yup schemas
├── src/components/        # UI components
├── src/hooks/             # useRoom, useQueue, usePlayback, useSSE
└── src/stores/            # Zustand stores
```

## API

```
POST   /api/v1/rooms                    # Create room
GET    /api/v1/rooms/:id                # Get room
POST   /api/v1/rooms/:id/sessions       # Join room
GET    /api/v1/rooms/:id/songs          # Get queue
POST   /api/v1/rooms/:id/songs          # Add song
DELETE /api/v1/rooms/:id/songs/:songId  # Remove song
POST   /api/v1/rooms/:id                # Actions (play/pause/seek/skip/vote)
GET    /api/v1/rooms/:id/events         # SSE stream
```

Full contract: `docs/API.md`

## Environment

```bash
PORT=8080
DATABASE_PATH=./vibez.db
YOUTUBE_API_KEY=required
VITE_API_URL=http://localhost:8080
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
