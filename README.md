# Vibes Backend

Go API backend for the Vibes collaborative music queue.

## Features

- **Real-time Sync**: Server-Sent Events (SSE) for low-latency state distribution
- **Provider Aggregator**: Unified API for YouTube, Spotify, and SoundCloud
- **PostgreSQL Storage**: Atomic database operations through prepared statements
- **OAuth2 Flow**: Centralized authorization management for music providers
- **OpenTelemetry**: Comprehensive monitoring with metrics and tracing

## Getting Started

### Prerequisites
- Go 1.26.7+
- PostgreSQL schema applied by the `vibes-migrator` repository
- Redis when `RATE_LIMIT_ENABLED=true`
- Environment variables configured for local development

### Development

```bash
# Run the backend server
go run cmd/server/main.go

# Or use the Makefile
make dev
```

### Environment Configuration

Copy and configure the environment file:
```bash
# Configure required API keys:
# - DATABASE_URL (required)
# - RATE_LIMIT_ENABLED (optional, defaults to false)
# - REDIS_URL (required for durable room-event replay and when rate limiting is enabled)
# - ROOM_EVENT_REPLAY_MAX_EVENTS (optional, defaults to 1000 retained events per active room)
# - ROOM_EVENT_REPLAY_MAX_AGE (optional, defaults to 2h)
# - YOUTUBE_API_KEY (required)
# - GROK_API_KEY (required when Grok is selected for AI-generated rooms)
# - GEMINI_API_KEY (required when Gemini is selected for AI-generated rooms)
# - AI_MODEL (optional, PROVIDER:model format; defaults to GROK:grok-4.3)
#   Examples: GROK:grok-4.5, GEMINI:gemini-3.6-flash
# - SPOTIFY_CLIENT_ID, SPOTIFY_CLIENT_SECRET (optional)
# - SOUNDCLOUD_CLIENT_ID, SOUNDCLOUD_CLIENT_SECRET (optional)
```

## Architecture

- **`cmd/server`**: Application entrypoint and dependency injection
- **`client/`**: External integrations (Database, Redis, YouTube, Spotify, SoundCloud)
- **`server/`**: HTTP router, middleware, and handlers
- **`server/internal/handler`**: Domain-specific HTTP handlers
- **`vibe/`**: Core domain types and interfaces (pure Go, no dependencies)
- **`config/`**: Configuration management and environment variables
- **`monitoring/`**: OpenTelemetry tracing, metrics, and telemetry

See the [architecture diagram](docs/ARCHITECTURE.md) and
[application flow diagrams](docs/FLOWS.md) for a visual overview.

## Key Conventions

- **No service layers**: Direct client usage via interfaces
- **Domain types in `vibe/`**: All business logic types defined here
- **Error handling**: All errors wrapped with context
- **Database**: Prepared statements with 1:1 naming convention
- **HTTP clients only**: All external integrations under `client/`
- **Database Transactions**: Avoid transactions unless no single-statement approach is reasonable.

---

For complete coding conventions, read the [AGENTS.md](./AGENTS.md).
