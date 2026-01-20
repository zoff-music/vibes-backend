# Vibez Backend

High-performance Go server for the Vibez ecosystem. Handles room state, real-time synchronization, and multi-provider music integration.

## Features

- **Real-time Sync**: Server-Sent Events (SSE) for low-latency state distribution.
- **Provider Aggregator**: Unified API for YouTube, Spotify, and SoundCloud.
- **SQLite Storage**: Atomic database operations using `modernc.org/sqlite`.
- **OAuth2 Flow**: Centralized authorization management for music providers.

## Getting Started

### Prerequisites
- Go 1.23+
- `YOUTUBE_API_KEY` (configured in environment)

### Development

```bash
cd backend
go build ./cmd/server
./server
```

## Architecture

- **`cmd/server`**: Application entrypoint and dependency injection.
- **`client/`**: External integrations (DB, Spotify, YouTube).
- **`server/internal/handler`**: Domain-specific HTTP handlers.
- **`vibe/`**: Core domain types and interfaces.

---

For non-negotiable coding conventions, read the [AGENTS.md](./AGENTS.md).
