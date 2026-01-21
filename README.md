# Vibez 🎵

Collaborative music queue with perfectly synchronized playback across devices. Support for YouTube, Spotify, and SoundCloud.

## ✨ Features

- **Collaborative Rooms**: Create rooms, invite friends, and manage a shared queue
- **Synchronized Playback**: Low-latency sync using Server-Sent Events (SSE)
- **Multi-Provider**: Search and play from YouTube, Spotify (Premium), and SoundCloud
- **Cast Support**: Dedicated Chromecast receiver application for big-screen vibes
- **Server-Side Rendering**: SSR-enabled React apps for better performance and SEO
- **HTTPS Development**: Local development with automatic SSL certificates via Caddy
- **Dark Mode Support**: System preference detection with manual toggle

## 🚀 Quick Start

The fastest way to get started is using the provided `Makefile`:

```bash
# Start the entire local development stack with HTTPS (Recommended)
make local-dev
```

This starts:
- **Backend**: Go server with database migrations
- **Platform App**: Main React application (SSR)
- **Cast Receiver**: Chromecast receiver app (SSR)
- **Caddy Proxy**: HTTPS reverse proxy with automatic SSL

Access at: **https://localhost**

### Alternative: Docker Development
```bash
# Full stack via Docker Compose
make dev
```

### Manual Development
```bash
# Backend only
cd backend && go run cmd/server/main.go

# Frontend only (both apps)
cd frontend && bun dev
```

## 📚 Documentation

- **Project Overview**: [CLAUDE.md](./CLAUDE.md)
- **API Reference**: [docs/API.md](./docs/API.md)
- **Music Providers**: [docs/MUSIC-PROVIDERS.md](./docs/MUSIC-PROVIDERS.md)
- **Cast Development**: [CAST_DEVELOPMENT_GUIDE.md](./CAST_DEVELOPMENT_GUIDE.md)
- **Deployment Guide**: [docs/DEPLOYMENT.md](./docs/DEPLOYMENT.md)
- **Frontend Rules**: [frontend/AGENTS.md](./frontend/AGENTS.md)
- **Backend Rules**: [backend/AGENTS.md](./backend/AGENTS.md)

## 🔧 Environment Setup

Copy the sample environment file and configure your API keys:

```bash
cp .env.sample .env
# Edit .env with your API keys (YouTube, Spotify, SoundCloud)
```

## 🐳 Deployment

Vibez supports blue-green deployments via Docker Compose and Caddy. See [docs/DEPLOYMENT.md](./docs/DEPLOYMENT.md) for details.
