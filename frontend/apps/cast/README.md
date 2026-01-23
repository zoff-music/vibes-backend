# Vibez Cast Receiver

The standalone Chromecast Receiver application for the Vibez ecosystem. It handles synchronized playback of multiple music providers on Google Cast devices with server-side rendering for optimal performance.

## 🚀 Getting Started

### Local Development
The receiver runs on port 3001 with SSR and is served via HTTPS through the root Caddy proxy.

```bash
cd frontend/apps/cast
bun install
bun dev
```

**Receiver URL**: `https://localhost/casting/receiver/`

### Build System
The cast app uses a custom build system with Bun that includes:

- **Custom Build Script**: `scripts/build.ts` handles client-side bundling with environment variable injection
- **File Watching**: Automatic rebuilds on `.tsx`, `.ts`, `.css`, and `public/` file changes
- **Bun Compatibility**: Falls back to Node.js `fs.watch` for Bun versions < 1.4.0
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
bun dev

# Production build (all assets)
bun run build

# Individual build steps
bun run build:css      # Tailwind CSS compilation
bun run build:client   # Client-side bundle
bun run build:server   # SSR server bundle

# Production server
bun start

# Type checking
bun run typecheck

# Linting
bun run lint
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
1. **CSS Processing**: Tailwind CSS v4 compilation with dark mode support
2. **Client Bundle**: Bun builds React app with environment variable injection
3. **Server Bundle**: Separate SSR server build for production deployment
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
- **Bun v1.4.0+**: Uses native `Bun.watch` API
- **Bun < v1.4.0**: Falls back to Node.js `fs.watch` with recursive directory monitoring
- **File Types**: Watches `.tsx`, `.ts`, `.css`, and `public/` directory changes
- **Debouncing**: Prevents excessive rebuilds during rapid file changes

## 🧩 Integration

This app is built with:
- **React 19**: Modern component architecture with SSR streaming
- **Bun**: Ultra-fast runtime and package manager
- **Tailwind CSS v4**: Theme-driven styling with enhanced dark mode
- **@vibez/ui**: Shared UI + playback engine across platform and cast apps
- **@vibez/shared**: Shared hooks, stores, and utilities
- **@vibez/models**: Type-safe API schemas and validation

## 🔧 Troubleshooting

### Common Issues

**Bun.watch errors**: If you see `TypeError: Bun.watch is not a function`, the build script will automatically fall back to `fs.watch`. Look for `[Build] Using fs.watch, watching /path` in the logs.

**SSR errors**: Check the server logs for `[SSR Error]` messages. The app includes graceful error recovery.

**HMR not working**: Ensure WebSocket connection is established. Look for `[HMR] Cast Client connected` in server logs.

---

For more details on the architecture, see [MUSIC-PROVIDERS.md](../../../docs/MUSIC-PROVIDERS.md).
