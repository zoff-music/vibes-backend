# Vibez Cast Receiver

The standalone Chromecast Receiver application for the Vibez ecosystem. It handles synchronized playback of multiple music providers on Google Cast devices.

## 🚀 Getting Started

### Local Development
The receiver usually runs on port 3001 and is served via HTTPS through the root Caddy proxy.

```bash
cd frontend/apps/cast
bun install
bun dev
```

**Receiver URL**: `https://localhost/casting/receiver/`

## 🛠 Features

- **Multi-Provider Playback**: Support for YouTube (IFrame), Spotify (Web Playback SDK), and SoundCloud (Widget).
- **Authentication Bridge**: Receives Spotify and SoundCloud tokens from the sender app via `LOAD` interceptors.
- **Global State**: Synchronizes with the backend via a shared Zustand store (`@vibez/shared`).
- **Premium UI**: Dark-mode primary interface with glassmorphism and smooth animations.

## 📡 Communication Protocol

The receiver communicates with sender applications (web/mobile) using standard Google Cast Message Interceptors:

- **LOAD Interceptor**: Processes incoming media requests.
  - Extracts `customData.tokens` to seed the authentication cache.
  - Extracts `customData.song` to update the local playback state.
- **Status Reporting**: Reports playback position and volume state back to the sender.

## 🧩 Integration

This app is built with:
- **React 19**: Modern component architecture.
- **Vite**: Ultra-fast build tool.
- **Tailwind CSS v4**: Theme-driven styling.
- **@vibez/player**: Shared playback engine.
- **@vibez/shared**: Shared hooks and stores.

---

For more details on the architecture, see [MUSIC-PROVIDERS.md](../../../docs/MUSIC-PROVIDERS.md).
