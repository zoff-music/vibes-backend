# Server-Side Rendering (SSR) Architecture

This document explains the SSR implementation used in Vibez for both the platform and cast applications.

## Overview

Vibez uses a unified SSR approach across both React applications:
- **Platform App**: Main collaborative music queue interface
- **Cast Receiver**: Chromecast receiver application

Both apps use identical build systems and SSR patterns for consistency and maintainability.

## Architecture Components

### 1. Build System (Bun-based)

Both apps use a unified build system with these key files:
- `scripts/build.ts` - Custom Bun build script with hashing and manifest generation
- `client.tsx` - Client-side hydration entry point
- `server.tsx` - SSR server with Bun runtime

#### Build Process
```bash
# Development (with watch mode)
bun run dev

# Production build
bun run build
```

The build process:
1. **CSS Generation**: Tailwind CSS compilation with minification
2. **Client Bundle**: JavaScript bundling with content hashing for cache busting
3. **Manifest Creation**: JSON manifest mapping logical names to hashed filenames
4. **Asset Copying**: Public files copied to dist directory

### 2. SSR Server (`server.tsx`)

Each app runs its own SSR server using Bun runtime:

```typescript
// Key features:
- React renderToString() for SSR
- Static file serving with proper MIME types
- Manifest-based asset resolution
- Route-based data fetching
- Error handling and fallbacks
```

#### Server Responsibilities
- **SSR Rendering**: Convert React components to HTML strings
- **Asset Serving**: Serve hashed static files (JS, CSS, images)
- **Data Fetching**: Pre-fetch data for initial page load
- **Manifest Resolution**: Map logical asset names to hashed filenames

### 3. Client Hydration (`client.tsx`)

Client-side hydration handles:
- **React Hydration**: Attach React to server-rendered HTML
- **Initial Data**: Extract SSR data from script tag
- **Router Setup**: Initialize client-side routing
- **Error Suppression**: Handle hydration mismatches gracefully

### 4. Asset Management

#### Hashed Filenames
All assets use content-based hashing for cache busting:
```
client.js → client-abc123.js
index.css → index-def456.css
```

#### Manifest System
The build generates a manifest mapping logical names to hashed files:
```json
{
  "main.js": "client-abc123.js",
  "index.css": "index-def456.css",
  "timestamp": "1234567890"
}
```

#### Asset Resolution
The SSR server uses the manifest to resolve correct asset URLs:
```typescript
const mainJS = manifest['main.js'] 
  ? `/assets/platform/${manifest['main.js']}` 
  : '/assets/platform/client.js';
```

## Development Workflow

### Local Development Setup

1. **Start Development Stack**:
   ```bash
   make local-dev
   ```

2. **Individual App Development**:
   ```bash
   # Platform app
   cd frontend/apps/platform && bun run dev
   
   # Cast app  
   cd frontend/apps/cast && bun run dev
   ```

### Development Features

- **Hot Reload**: File watching with automatic rebuilds
- **SSR in Development**: Full SSR even in development mode
- **HTTPS Support**: Caddy provides HTTPS with automatic certificates
- **Asset Proxying**: Caddy routes assets to correct app servers

### Routing Configuration

Caddy handles routing between apps:
```caddyfile
# Platform assets
handle /assets/platform/* {
    reverse_proxy localhost:3000
}

# Cast assets  
handle /assets/cast/* {
    reverse_proxy localhost:3001
}

# Cast receiver routes
handle /casting/receiver/* {
    reverse_proxy localhost:3001
}

# Default to platform app
handle {
    reverse_proxy localhost:3000
}
```

## Key Implementation Details

### 1. Unified Build System

Both apps share identical build configurations:
- Same `scripts/build.ts` structure
- Identical manifest format
- Consistent asset hashing
- Same development workflow

### 2. Environment Variable Handling

Fixed browser compatibility issues:
```typescript
// ❌ Problematic (process not available in browser)
if (process?.env?.API_URL) {
  return process.env.API_URL;
}

// ✅ Correct (browser-compatible)
if (import.meta.env?.VITE_API_URL) {
  return import.meta.env.VITE_API_URL;
}
```

### 3. Code Splitting Configuration

Disabled code splitting to prevent duplicate exports:
```typescript
// Build configuration
await Bun.build({
  entrypoints: ['./client.tsx'],
  splitting: false, // Prevents duplicate exports
  // ...
});
```

### 4. TypeScript Configuration

Consistent TypeScript setup across apps:
```json
{
  "compilerOptions": {
    "jsx": "react-jsx",
    "moduleResolution": "bundler",
    "isolatedModules": false, // For cast app compatibility
    // ...
  }
}
```

## Production Deployment

### Build Process
```bash
# Build all apps
cd frontend && bun run build

# Builds generate:
# - frontend/apps/platform/dist/
# - frontend/apps/cast/dist/
```

### Asset Structure
```
dist/
├── assets/platform/
│   ├── client-[hash].js
│   ├── index-[hash].css
│   └── [other assets]
├── assets/cast/
│   ├── client-[hash].js
│   ├── index-[hash].css
│   └── [other assets]
└── manifest.json
```

### Server Deployment
- Each app runs its own SSR server
- Reverse proxy (Caddy/nginx) routes requests
- Static assets served with proper caching headers
- Manifest-based asset resolution

## Troubleshooting

### Common Issues

1. **Asset 404 Errors**
   - Check manifest.json generation
   - Verify asset paths in HTML output
   - Ensure reverse proxy routing is correct

2. **Hydration Mismatches**
   - Check SSR data serialization
   - Verify client/server environment differences
   - Review conditional rendering logic

3. **Build Failures**
   - Check TypeScript configuration
   - Verify import paths and dependencies
   - Review build script configuration

### Debug Commands

```bash
# Check asset generation
ls -la frontend/apps/platform/dist/assets/platform/

# Verify manifest content
cat frontend/apps/platform/dist/manifest.json

# Test asset serving
curl -k https://localhost/assets/platform/client-[hash].js

# Check SSR output
curl -k https://localhost/ | grep -A5 -B5 "script\|link"
```

## Best Practices

### 1. Asset Management
- Always use manifest for asset resolution
- Include timestamp in manifest for cache invalidation
- Use content hashing for all static assets

### 2. SSR Data Handling
- Serialize data safely (avoid XSS)
- Handle missing data gracefully
- Keep initial data minimal

### 3. Error Handling
- Implement fallbacks for SSR failures
- Handle hydration mismatches gracefully
- Log errors for debugging

### 4. Performance
- Minimize initial bundle size
- Use proper caching headers
- Optimize asset loading order

## Future Improvements

- **Streaming SSR**: Implement React 18 streaming features
- **Edge Deployment**: Deploy SSR servers to edge locations
- **Bundle Analysis**: Add bundle size monitoring
- **Performance Metrics**: Track SSR performance metrics