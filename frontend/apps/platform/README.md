# Vibez Platform

The primary web application for the Vibez ecosystem. It serves as the main interface for users to create rooms, manage queues, and control synchronized playback across devices.

## 🚀 Getting Started

### Local Development
The platform app runs on port 3000 by default. For the full experience including HTTPS and proxying, use the root-level `make local-dev`.

```bash
cd frontend/apps/platform
bun install
bun dev
```

**App URL**: `https://localhost` (via Caddy) or `http://localhost:3000` (direct)

## 🛠 Features

- **Collaborative Queue**: Real-time voting, reordering, and removal of tracks.
- **Smart Search**: Unified search across YouTube, Spotify, and SoundCloud with debouncing and autocomplete.
- **Synchronized Playback**: Leveraging Server-Sent Events (SSE) to keep all participants in perfect sync.
- **Device Management**: seamlessly switch playback between your browser and local Chromecast devices.
- **Social Integration**: Shareable room links and generated QR codes for easy joining.

## 🧩 Technical Stack

- **Framework**: React 19 + TypeScript.
- **State Management**: **Zustand** for high-performance, selective store subscriptions (playback, UI, auth).
- **Styling**: **Tailwind CSS v4** with a custom "retro-futuristic" design system.
- **API Engine**: **wiretyped** + **yup** for type-safe, validated request/response handling.
- **Real-time**: **EventSource** (SSE) for low-latency state updates from the backend.

## 📁 Source Structure

- `/src/components`: UI library, separated into `ui` (primitives), `player` (playback logic), and `queue` (list management).
- `/src/stores`: Zustand global stores.
- `/src/hooks`: Shared logic for authentication, casting, and room events.
- `/src/api`: Auto-generated and custom API clients.

---

For architecture details on music providers, see [MUSIC-PROVIDERS.md](../../../docs/MUSIC-PROVIDERS.md).
