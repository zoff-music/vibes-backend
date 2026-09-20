# Vibes Backend

Go API and background processing for [Zoff](https://zoff.me), a free shared
music queue for listening together. Listeners join without creating an account;
each room has its own settings and optional administrator password.

## Features

- Shared YouTube and SoundCloud queues, song voting, playlist imports, and
  synchronized playback through official provider players.
- Server-controlled playback or host-controlled rooms, administrator-only
  adding/skipping, democratic skip votes, and public-room browsing.
- Room chat and activity for additions, votes, removals, skips, and name changes.
- Versioned REST endpoints and replayable SSE, including incremental v2 queue
  updates instead of repeatedly sending the full playlist.
- Paired remote controls, room-scoped Cast tokens, and shared contracts for web,
  mobile, television, Cast, and embed clients.
- Prompt-based playlist generation with configurable Grok or Gemini providers,
  provider metadata refresh, and bounded background retries.
- Protected global administration, usage counters, rate limiting, metrics, and
  OpenTelemetry tracing.

The [frontend](https://github.com/zoff-music/vibes-frontend) owns user interfaces.
The [migrator](https://github.com/zoff-music/vibes-migrator) owns PostgreSQL schema
changes. This service coordinates state; it does not host or extract music streams.

## Getting Started

### Prerequisites

- Go matching [go.mod](go.mod), currently 1.27.1 or newer.
- PostgreSQL with the migrator's schema applied before starting the backend.
- Redis, required for event replay, caches, and remote state even when rate
  limiting is disabled.
- Credentials for the music and playlist-generation providers you enable.

### Development

Configure a local `.env` or export environment variables:

```bash
export DATABASE_URL='postgres://user:password@localhost:5432/vibes?sslmode=disable'
export REDIS_URL='redis://localhost:6379/0'
export COOKIE_SECRET='replace-with-a-long-random-local-secret'
export YOUTUBE_API_KEY='your-youtube-api-key'
# Set the key for the configured AI provider when running playlist generation.
export GROK_API_KEY='your-grok-api-key'

make dev
```

Use deployment-specific secrets in production. Never reuse the built-in
development cookie secret. Review [config/config.go](config/config.go) for the
complete configuration and validation rules.

### Environment Configuration

| Setting | Purpose |
| --- | --- |
| `DATABASE_URL`, `REDIS_URL` | Required storage connections |
| `PORT`, `INTERNAL_PORT` | Public API and internal health/metrics listeners; defaults 8080 and 8081 |
| `COOKIE_SECRET`, `ADMIN_PASSWORD_PEPPER`, `CAST_TOKEN_SECRET` | Session signing, global-admin password protection, and Cast token signing |
| `YOUTUBE_API_KEY` | Enables YouTube; OAuth client settings are separate |
| `SOUNDCLOUD_CLIENT_ID`, `SOUNDCLOUD_CLIENT_SECRET` | SoundCloud provider credentials |
| `AI_MODEL` | `PROVIDER:model` selection; default is defined in config |
| `GROK_API_KEY`, `GEMINI_API_KEY` | Credential for the selected generation provider |
| `RATE_LIMIT_ENABLED` | Enables route-policy rate limiting; defaults to false |
| `ROOM_EVENT_REPLAY_MAX_EVENTS`, `ROOM_EVENT_REPLAY_MAX_AGE` | Replay bounds; defaults 1000 events and 2h |
| `CORS_ALLOWED_ORIGINS` | Allowed browser origins |
| `OTEL_*` | Telemetry endpoint, service identity, sampling, and export timing |

### Checks and Builds

```bash
make test          # Swagger generation and Go race tests
GOFLAGS=-mod=mod go vet ./...
make build
make docs          # Regenerate Swagger from handler/domain declarations
make docker        # Build the production image
```

Database integration tests use `VIBES_TEST_DATABASE_URL` pointing to a migrated,
disposable local PostgreSQL database. The production Docker target uses a
statically linked Go binary in a scratch image.

## Architecture

| Path | Responsibility |
| --- | --- |
| `cmd/server` | Process entrypoint |
| `server/server.go`, `server/router.go` | Client initialization, routing, middleware policies, and shutdown |
| `server/internal/handler` | Explicit HTTP handlers and scheduled app-event handlers |
| `server/internal/event` | Scheduled-event wiring and dispatch |
| `client/database` | Atomic prepared PostgreSQL operations, row scanning, and domain mapping |
| `client/redis` | Replay streams, caches, pairing state, and rate-limit storage |
| `client/youtube`, `client/soundcloud` | Music-provider APIs |
| `client/grok`, `client/gemini` | Playlist-generation APIs |
| `vibe` | Domain types, payloads, validation, and narrow capability interfaces |
| `config`, `monitoring` | Configuration, tracing, and metrics |

Chat messages are limited to 500 characters, display names to 30, and new room
names/reservations to 100 after trimming. The backend validates inputs independently
of the frontend. Existing room reads are not rejected by creation limits.

See the [architecture](docs/ARCHITECTURE.md), [application flows](docs/FLOWS.md),
[API contract](docs/API.md), and [provider notes](docs/MUSIC-PROVIDERS.md).

## Key Conventions

- No service/repository layer: handlers receive narrow domain interfaces.
- Inject each concrete client once per handler.
- Keep handler flow explicit, including response and event construction.
- Use prepared, atomic SQL statements rather than Go-managed transactions.
- Separate row scanning from `toXxx` mapping; return domain structs with errors.
- Cancel requests and workers, drain servers, and wait before closing clients.

Read [AGENTS.md](AGENTS.md) and the
[backend skill](.agents/skills/vibes-backend/SKILL.md) before contributing.
