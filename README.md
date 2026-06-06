# Vibez

Collaborative music queue with a Go backend and React frontend.

## Features

- **Real-time Sync**: Server-Sent Events (SSE) for low-latency state distribution
- **Provider Aggregator**: Unified API for YouTube, Spotify, and SoundCloud
- **PostgreSQL Storage**: Atomic database operations through prepared statements
- **OAuth2 Flow**: Centralized authorization management for music providers
- **OpenTelemetry**: Comprehensive monitoring with metrics and tracing
- **Automatic Migrations**: Database schema management via migrator tool

## Getting Started

### Prerequisites
- Go 1.26.4+
- pnpm
- Environment variables configured (see `.env.sample`)

### Development

```bash
# Run with automatic database migration in another shell
go run cmd/migrator/main.go
go run cmd/server/main.go

# Or use the Makefile for full stack
make dev
```

### Environment Configuration

Copy and configure the environment file:
```bash
cp .env.sample .env
# Configure required API keys:
# - YOUTUBE_API_KEY (required)
# - SPOTIFY_CLIENT_ID, SPOTIFY_CLIENT_SECRET (optional)
# - SOUNDCLOUD_API_KEY (optional)
```

## Architecture

- **`cmd/server`**: Application entrypoint and dependency injection
- **`cmd/migrator`**: Migration entrypoint
- **`client/`**: External integrations (Database, YouTube, Spotify, SoundCloud, PubSub)
- **`client/frontend/render`**: Frontend pnpm workspace
- **`server/`**: HTTP router, middleware, and handlers
- **`server/internal/handler`**: Domain-specific HTTP handlers
- **`vibe/`**: Core domain types and interfaces (pure Go, no dependencies)
- **`config/`**: Configuration management and environment variables
- **`monitoring/`**: OpenTelemetry tracing, metrics, and telemetry

## Key Conventions

- **No service layers**: Direct client usage via interfaces
- **Domain types in `vibe/`**: All business logic types defined here
- **Error handling**: All errors wrapped with context
- **Database**: Prepared statements with 1:1 naming convention
- **HTTP clients only**: All external integrations under `client/`
- **Database Transactions**: Avoid transactions unless no single-statement approach is reasonable.

---

For complete coding conventions, read the [AGENTS.md](./AGENTS.md).
