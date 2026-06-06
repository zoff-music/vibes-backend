# Vibez

Collaborative music queue with synchronized playback. Supports YouTube, Spotify, and SoundCloud.

## Commands

```bash
make dev
make test
make frontend
make backend
make migrator
```

Manual commands:

```bash
go run cmd/server/main.go
go run cmd/migrator/main.go
pnpm --dir client/frontend/render install
pnpm --dir client/frontend/render dev
```

## Stack

| Layer | Tech |
|-------|------|
| Frontend | React 19 + Vite + Tailwind CSS v3 + TypeScript |
| Backend | Go + PostgreSQL + OpenTelemetry |
| Migrator | Go + PostgreSQL |
| Real-time | Server-Sent Events |
| Package Manager | pnpm 11.5.2 |
| Proxy | Caddy for local HTTPS |

## Structure

```text
cmd/server/                 # Backend entrypoint
cmd/migrator/               # Migrator entrypoint
client/                     # Database and external service clients
config/                     # Configuration
internalerror/              # Shared error helpers
monitoring/                 # Telemetry, tracing, metrics
server/                     # HTTP routes, handlers, middleware
vibe/                       # Domain types and interfaces
migrator/migrations/        # SQL migrations
client/frontend/render/     # Frontend pnpm workspace
```

## Coding Rules

Read before writing code:

- Backend: `AGENTS.md`
- Frontend: `client/frontend/render/AGENTS.md`

Critical rules:

- All errors wrapped with context.
- No inline Go error assignments.
- HTTP only through clients, consumed via interfaces.
- Domain types live in `vibe/`, not handlers.
