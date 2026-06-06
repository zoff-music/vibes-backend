# Vibez Cast Receiver

The standalone Chromecast Receiver application for the Vibez ecosystem. It handles synchronized playback of multiple music providers on Google Cast devices with server-side rendering for optimal performance.

## 🚀 Getting Started

### Local Development
The receiver runs on port 3001 with SSR and is served via HTTPS through the root Caddy proxy.

```bash
cd client/frontend/render/apps/cast
pnpm install
pnpm dev
```

**Receiver URL**: `https://localhost/casting/receiver/`

### Build System
The cast app uses Vite for local development and production builds.

- **Custom Build Script**: `scripts/build.ts` handles client-side bundling with environment variable injection
- **File Watching**: Automatic rebuilds on `.tsx`, `.ts`, `.css`, and `public/` file changes
- **Node.js Runtime**: Served through Vite preview in the production container
- **Environment Variables**: Automatic `VITE_*` prefix mapping and defaults for Cast configuration

### Server-Side Rendering (SSR)
The cast app includes full SSR support via `server.tsx`:

- **React 19 SSR**: Uses `renderToReadableStream` for streaming HTML
- **Static Router**: Server-side routing with React Router
- **Static File Serving**: Handles both `public/` and `dist/client/` assets
- **Hot Module Replacement**: WebSocket-based HMR in development
- **Error Handling**: Graceful SSR error recovery with `safeWrapAsync`

### Development Scripts

```bash
# Development with watch mode and SSR
pnpm dev

# Production build (all assets)
pnpm build

# Individual build steps
pnpm build

# Production server
pnpm preview

# Type checking
pnpm typecheck

# Linting
pnpm lint
```

## 🛠 Features

- **Multi-Provider Playback**: Support for YouTube (IFrame), Spotify (Web Playback SDK), and SoundCloud (Widget)
- **Authentication Bridge**: Receives Spotify and SoundCloud tokens from the sender app via `LOAD` interceptors
- **Global State**: Synchronizes with the backend via a shared Zustand store (`@vibez/shared`)
- **Premium UI**: Dark-mode primary interface with glassmorphism and smooth animations
- **Server-Side Rendering**: Fast initial page loads with streaming HTML
- **Hot Module Replacement**: Instant development feedback

## 📡 Communication Protocol

The receiver communicates with sender applications (web/mobile) using standard Google Cast Message Interceptors:

- **LOAD Interceptor**: Processes incoming media requests
  - Extracts `customData.tokens` to seed the authentication cache
  - Extracts `customData.song` to update the local playback state
- **Status Reporting**: Reports playback position and volume state back to the sender

## 🏗 Architecture

### Build Pipeline
### Build Pipeline
1. **CSS Processing**: Tailwind CSS v3 compilation (downgraded from v4 for better legacy browser support)
2. **Legacy Transpilation**: Uses `@vitejs/plugin-legacy` to generate `nomodule` scripts for older Chromecast devices
3. **Client Bundle**: Vite builds React app with environment variable injection
4. **Asset Copying**: Public files copied to distribution directory

### Environment Configuration
The build system automatically handles environment variables:

```bash
# Default Cast configuration
VITE_CAST_APP_ID=1FAF5D9F
VITE_CAST_RECEIVER_URL=/casting/receiver/

# Custom configuration (optional)
CAST_APP_ID=your-app-id
CAST_RECEIVER_URL=your-receiver-url
FRONTEND_URL=your-frontend-url
```

### File Watching (Development)
The build script includes intelligent file watching:
- **Vite**: Development server and static production build
- **File Types**: Watches `.tsx`, `.ts`, `.css`, and `public/` directory changes
- **Debouncing**: Prevents excessive rebuilds during rapid file changes

## 🧩 Integration

This app is built with:
- **React 19**: Modern component architecture with SSR streaming
- **pnpm**: Workspace package manager
- **Tailwind CSS v3**: Theme-driven styling with legacy browser support
- **Legacy Transpilers**: Ensures compatibility with all Chromecast generations
- **@vibez/shared**: Shared hooks, stores, and utilities
- **@vibez/models**: Type-safe API schemas and validation

## 🔧 Troubleshooting

### Common Issues

**Preview errors**: Rebuild with `pnpm build`, then run `pnpm preview`.

**SSR errors**: Check the server logs for `[SSR Error]` messages. The app includes graceful error recovery.

**HMR not working**: Ensure WebSocket connection is established. Look for `[HMR] Cast Client connected` in server logs.

---

For more details on the architecture, see [MUSIC-PROVIDERS.md](../../../docs/MUSIC-PROVIDERS.md).

## 🐛 Debugging

### Remote Logging
The Cast Receiver implements a custom logging system that broadcasts console logs (log, info, warn, error) back to the connected Sender application. This is crucial for debugging simple Chromecasts that do not have a remote debugger interface.

**To enable remote logging:**
1. **Via URL**: Append `?debug=true` to the receiver URL (useful for local development/emulators).
2. **Via Sender**: When loading media, include `{ customData: { debug: true } }` in the Load Request.

**How it works:**
- The receiver intercepts all console methods.
- Logs are serialized and sent via the custom namespace `urn:x-cast:com.vibez.cast` with action `LOG`.
- The sender (web/mobile) listens for these messages and replays them in its own console with a `[RECEIVER]` prefix.
- **Note**: `error` level logs are *always* sent to the sender, regardless of debug mode.

### Common Issues & Fixes

**Playback Position Resetting**
If you notice the playback position resetting to `0:00` when play/pause is toggled rapidly or during sync updates:
- **Cause**: The receiver receives `updatePlayback` or `syncPlayback` messages where `positionMs` might be 0 (default fallback) or slightly out of sync.
- **Fix**: The `CastProvider` has logic to **preserve the local playback position** if:
  1. The Song ID has not changed (same track).
  2. The incoming message has `positionMs: 0` (or missing).
  3. The local player has already progressed past 1 second (`> 1000ms`).
- **Logic**: This prevents the "restart from beginning" glitch while still allowing legitimate track changes to start from 0.

**Startup Crashes**
If the receiver fails to load entirely (white screen):
- We use a **global logging initialization** in `logging.ts` that runs *before* React mounts.
- This ensures that syntax errors, missing imports, or early runtime crashes are caught and broadcast to the sender (if connected).
