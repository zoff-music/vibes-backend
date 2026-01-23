# Start Development

## Recommended (HTTPS with Unified SSR)

Run the full stack (Backend + Frontend SSR + Caddy) with HTTPS enabled:

```bash
make local-dev
```

- **Platform App**: [https://localhost](https://localhost) (Main collaborative music queue interface)
- **Cast Receiver**: [https://localhost/casting/receiver/](https://localhost/casting/receiver/) (Chromecast receiver app)
- **API**: [https://localhost/api/v1](https://localhost/api/v1) (Backend API)

Both apps use unified Bun-based build system with:
- Server-side rendering (SSR)
- Content-based hashing for cache busting
- Manifest-based asset resolution
- Hot reload with file watching
- Tailwind CSS v4 compilation

## Architecture

The development stack includes:
- **Backend**: Go API server (port 8080)
- **Platform SSR**: Bun server (port 3000)
- **Cast SSR**: Bun server (port 3001)
- **Caddy**: Reverse proxy with HTTPS (port 443)

Caddy routes:
- `/api/*` → Backend API
- `/assets/platform/*` → Platform assets
- `/assets/cast/*` → Cast assets
- `/casting/receiver/*` → Cast app
- `/*` → Platform app (default)

## Docker Development

```bash
make dev
```

Full stack via Docker Compose with unified SSR-enabled frontend services.

## Manual Development

### 1. Start Backend
```bash
cd backend && go run cmd/server/main.go
```

### 2. Start Frontend (Both Apps)
```bash
cd frontend && bun dev
```

This starts both SSR servers:
- Platform: http://localhost:3000 (Bun SSR)
- Cast: http://localhost:3001 (Bun SSR)

### 3. Individual App Development
```bash
# Platform app only
cd frontend/apps/platform && bun run dev

# Cast app only
cd frontend/apps/cast && bun run dev
```

## Environment Variables

### Backend (.env)
```bash
PORT=8080
DATABASE_PATH=./data/vibes.db
YOUTUBE_API_KEY=your-youtube-api-key
SPOTIFY_CLIENT_ID=your-spotify-client-id
SPOTIFY_CLIENT_SECRET=your-spotify-client-secret
SOUNDCLOUD_CLIENT_ID=your-soundcloud-client-id
SOUNDCLOUD_CLIENT_SECRET=your-soundcloud-client-secret
```

### Frontend
```bash
VITE_API_URL=https://localhost/api/v1  # For local-dev
# or
VITE_API_URL=http://localhost:8080/api/v1  # For manual dev
```

## Database Setup

```bash
# Run migrations
make migrate-up

# Reset database (if needed)
make migrate-down STEPS=0
make migrate-up
```

## Troubleshooting

### Asset 404 Errors
- Check manifest.json generation in `dist/` folders
- Verify Caddy routing configuration
- Ensure SSR servers are running

### Hydration Issues
- Check SSR data injection in HTML
- Verify client-side data extraction
- Ensure consistent server/client state

### Build Issues
- Run `bun run typecheck` for TypeScript errors
- Run `bun run lint` for code quality issues
- Check build logs for specific errors
