# Vibes Backend

Go API and background processing for [Zoff](https://zoff.me), a free shared
MUSIC queues for listening together and WATCH rooms for watching together.
Participants join without creating an account;
each room has its own settings and optional administrator password.

## Architecture Overview

[![Zoff application architecture](docs/architecture.svg)](docs/ARCHITECTURE.md)

See the [full architecture](docs/ARCHITECTURE.md) for application boundaries,
storage, permissions, event delivery, and deployment lifecycle. The
[application flow diagrams](docs/FLOWS.md) walk through joining rooms, chat,
playlist imports, generation, playback, and remote control.

## Features

- MUSIC rooms with YouTube and SoundCloud, and YouTube-only WATCH rooms.
- Shared playlist items, voting, playlist imports, and
  synchronized playback through official provider players.
- Server-controlled playback or host-controlled rooms, administrator-only
  adding/skipping, democratic skip votes, and public-room browsing.
- Room chat and activity for additions, votes, removals, skips, and name changes.
- Versioned REST endpoints and replayable SSE, including incremental v3 playlist-item
  updates instead of repeatedly sending the full playlist.
- Paired remote controls, room-scoped Cast tokens, and shared contracts for web,
  mobile, television, Cast, and embed clients.
- Prompt-based playlist generation with configurable Grok or Gemini providers,
  provider metadata refresh, and bounded background retries.
- Protected global administration, usage counters, rate limiting, metrics, and
  OpenTelemetry tracing.

The [frontend](https://github.com/zoff-music/vibes-frontend) owns user interfaces.
The [migrator](https://github.com/zoff-music/vibes-migrator) owns PostgreSQL schema
changes. This service coordinates state; it does not host or extract media streams.

## Getting Started

### Prerequisites

- Go matching [go.mod](go.mod), currently 1.27.2 or newer.
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

Use deployment-specific random secrets. `COOKIE_SECRET` is required and must
contain at least 32 bytes; the old development default is rejected. A configured
`CAST_TOKEN_SECRET` must also contain at least 32 bytes. Review
[config/config.go](config/config.go) for the complete configuration and validation
rules.

### Environment Configuration

| Setting | Purpose |
| --- | --- |
| `DATABASE_URL`, `REDIS_URL` | Required storage connections |
| `PORT`, `INTERNAL_PORT` | Public API and internal health/metrics listeners; defaults 8080 and 8081 |
| `COOKIE_SECRET`, `ADMIN_PASSWORD_PEPPER`, `CAST_TOKEN_SECRET` | Session signing, global-admin password protection, and Cast token signing |
| `REQUEST_BODY_MAX_BYTES` | API request body limit, including chunked bodies; defaults to 1 MiB |
| `YOUTUBE_API_KEY` | Enables YouTube; OAuth client settings are separate |
| `SOUNDCLOUD_CLIENT_ID`, `SOUNDCLOUD_CLIENT_SECRET` | SoundCloud provider credentials |
| `AI_MODEL` | `PROVIDER:model` selection; default is defined in config |
| `GROK_API_KEY`, `GEMINI_API_KEY` | Credential for the selected generation provider |
| `RATE_LIMIT_ENABLED` | Enables general route-policy rate limiting; defaults to false. Admin sign-in is always rate-limited, including internal requests |
| `ROOM_EVENT_REPLAY_MAX_EVENTS`, `ROOM_EVENT_REPLAY_MAX_AGE` | Replay bounds; defaults 100 events and 15m. Older cursors receive a fresh snapshot. |
| `CORS_ALLOWED_ORIGINS` | Allowed browser origins |
| `OTEL_*` | Telemetry endpoint, service identity, sampling, and export timing |

### Checks and Builds

```bash
make test          # Swagger generation and race-enabled package compilation
make security      # gosec and govulncheck; also required before make build
GOFLAGS=-mod=mod go vet ./...
make build
make docs          # Regenerate Swagger from handler/domain declarations
make docker        # Build the production image
```

The production Docker target uses a
statically linked Go binary in a scratch image.

The production workflow requires both security scanners to pass before building
or publishing the image. Documentation generation remains a separate `docs`
target and CI step; `make build` generates docs, runs security, then compiles.
On a fresh checkout, run `make docs` before standalone scanner checks.
`GOSEC_VERSION` pins the current upstream commit while
`GOVULNCHECK_VERSION` defaults to the latest release. Both can be overridden when
running Make; scanner binaries are resolved by Go rather than taken from `PATH`.
Pull requests do not trigger workflows; security checks run within releases.
The [release workflow](.github/workflows/release.yaml) only builds and publishes
production images on pushes to `main` in `zoff-music/vibes-backend`, including
PR merges and direct commits, checking security on that exact pushed commit.
Forks and other branches cannot publish. Registry credentials
use a temporary Docker configuration that is removed after publishing or failure.

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
