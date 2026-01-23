# Hot Module Replacement (HMR) Setup

## Overview

The platform app now uses **Vite** for development with proper Hot Module Replacement (HMR). This means:

- ✅ **Instant updates**: Changes to React components update immediately without page refresh
- ✅ **State preservation**: Component state is preserved during updates when possible
- ✅ **Fast builds**: Vite's optimized bundling for faster development
- ✅ **Better error handling**: Clear error overlays in development

## Development Commands

### Best HMR Experience (Recommended)
```bash
# Platform + Backend only (optimal HMR)
make dev-platform
# Access at: http://localhost:3001
```

### Full Development Environment
```bash
# All services with HTTPS proxy
make local-dev
# Access at: http://localhost:3001 (best HMR) or https://localhost (proxied)
```

### Platform app only
```bash
cd frontend/apps/platform
bun run dev
```

## Access URLs & HMR Performance

| URL | HMR Quality | Use Case |
|-----|-------------|----------|
| `http://localhost:3001` | ⭐⭐⭐ **Best** | Development with instant updates |
| `https://localhost` | ⭐⭐ **Good** | Testing with HTTPS/Cast features |

**Why the difference?**
- Direct Vite access (`localhost:3001`) has uninterrupted WebSocket connection
- Proxied access (`https://localhost`) goes through Caddy which can interfere with HMR WebSocket

## How HMR Works

1. **File watching**: Vite watches your source files for changes
2. **Module replacement**: When you save a file, Vite replaces just that module
3. **State preservation**: React Fast Refresh preserves component state when possible
4. **Error recovery**: Syntax errors show overlay, auto-recover when fixed

## What Updates Automatically

- ✅ React component changes
- ✅ CSS/Tailwind changes
- ✅ TypeScript changes
- ✅ Hook changes
- ✅ Store changes (Zustand)

## Troubleshooting

### HMR not working?
1. **Use direct URL**: Access `http://localhost:3001` instead of `https://localhost`
2. Check browser console for WebSocket connection errors
3. Look for TypeScript errors in the terminal
4. Try refreshing the page once

### Still having issues?
```bash
# Kill all processes and restart
make dev-platform
```

### Port conflicts?
The setup uses these ports:
- 3001: Vite dev server (platform)
- 3002: HMR WebSocket
- 3000: Cast app
- 8080: Backend API

## Migration from Old System

The old build system used:
- Custom Bun build script with file watching
- Manual browser refresh required
- Slower rebuild times

The new Vite system provides:
- Native HMR support
- Faster builds
- Better developer experience
- Automatic browser updates

## Configuration Files

- `vite.config.ts`: Vite configuration
- `index.html`: Development HTML template
- `client.tsx`: Updated entry point with HMR support