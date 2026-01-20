# Vibez 🎵

Collaborative music queue with perfectly synchronized playback across devices. Support for YouTube, Spotify, and SoundCloud.

## ✨ Features

- **Collaborative Rooms**: Create rooms, invite friends, and manage a shared queue.
- **Synchronized Playback**: Low-latency sync using Server-Sent Events (SSE).
- **Multi-Provider**: Search and play from YouTube, Spotify (Premium), and SoundCloud.
- **Cast Support**: Dedicated Chromecast receiver application for big-screen vibes.
- **Retro-Futuristic Design**: Sleek, glassmorphic UI with dynamic themes.

## 🚀 Quick Start

The fastest way to get started is using the provided `Makefile`:

```bash
# Start the entire local development stack (Backend + Frontend + Caddy)
make local-dev
```

Alternatively, run components manually:

### Backend
```bash
cd backend
go build ./cmd/server && ./server
```

### Frontend
```bash
cd frontend
bun install
bun dev
```

## 📚 Documentation

- **Project Overview**: [CLAUDE.md](./CLAUDE.md)
- **API Reference**: [docs/API.md](./docs/API.md)
- **Music Providers**: [docs/MUSIC-PROVIDERS.md](./docs/MUSIC-PROVIDERS.md)
- **Frontend Rules**: [frontend/AGENTS.md](./frontend/AGENTS.md)
- **Backend Rules**: [backend/AGENTS.md](./backend/AGENTS.md)

## 🐳 Deployment

Vibez supports blue-green deployments via Docker Compose and Caddy. See [.github/deploy/deploy.sh](./.github/deploy/deploy.sh) for details.
