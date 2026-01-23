# Vibez Backend

High-performance Go server for the Vibez ecosystem. Handles room state, real-time synchronization, and multi-provider music integration.

## Features

- **Real-time Sync**: Server-Sent Events (SSE) for low-latency state distribution
- **Provider Aggregator**: Unified API for YouTube, Spotify, and SoundCloud
- **SQLite Storage**: Atomic database operations using `modernc.org/sqlite` (CGO-free)
- **OAuth2 Flow**: Centralized authorization management for music providers
- **OpenTelemetry**: Comprehensive monitoring with metrics and tracing
- **Automatic Migrations**: Database schema management via migrator tool

## Getting Started

### Prerequisites
- Go 1.23+
- Environment variables configured (see `.env.sample`)

### Development

```bash
# Run with automatic database migration
cd backend
go run cmd/server/main.go

# Or use the Makefile for full stack
make local-dev  # Includes HTTPS via Caddy
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
- **`client/`**: External integrations (Database, YouTube, Spotify, SoundCloud, PubSub)
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
- **Database Transactions**: No transactions are allowed, with the sole exception of the `AddSong` method to handle duplicate song logic correctly.

---

For complete coding conventions, read the [AGENTS.md](./AGENTS.md).
