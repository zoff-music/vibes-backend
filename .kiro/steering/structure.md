# Project Structure

## Root Layout

```
├── backend/           # Go API server
├── frontend/          # React web application (Bun workspace)
├── migrator/          # Database migration tool
├── docs/              # API documentation
├── .kiro/             # Kiro configuration and steering
├── docker-compose.yml # Development environment
├── Caddyfile          # Reverse proxy config
└── Makefile           # Build and development commands
```

## Backend Structure (`backend/`)

```
backend/
├── cmd/server/        # Application entrypoint
├── client/            # External service clients
│   ├── database/      # SQLite operations
│   ├── youtube/       # YouTube API client
│   ├── soundcloud/    # SoundCloud API client
│   ├── spotify/       # Spotify API client
│   └── internalpubsub/ # SSE broadcasting
├── server/            # HTTP server and routing
│   ├── internal/handler/ # HTTP request handlers
│   ├── internal/middleware/ # HTTP middleware
│   └── internal/helper/ # Utility functions
├── vibe/              # Domain types and interfaces (CRITICAL)
├── config/            # Configuration management
├── monitoring/        # Telemetry, tracing, metrics
└── data/              # SQLite database files
```

### Key Backend Conventions

- **Domain types**: ALL in `vibe/vibe.go` - never in handlers
- **No service layers**: Direct client usage via interfaces
- **Client pattern**: External integrations under `client/`
- **Handler pattern**: Functions returning `http.HandlerFunc`
- **Database**: Prepared statements with 1:1 naming convention

## Frontend Structure (`frontend/`)

```
frontend/
├── apps/platform/     # Main React application
│   ├── src/api/       # wiretyped client + Yup schemas
│   ├── src/components/ # UI components
│   │   ├── ui/        # Base components (Button, Input, etc.)
│   │   ├── player/    # Video player components
│   │   └── queue/     # Queue management components
│   ├── src/hooks/     # Custom React hooks
│   ├── src/stores/    # Zustand state management
│   ├── src/pages/     # Route components
│   └── src/utils/     # Utility functions (error handling)
└── packages/shared/   # Shared code across apps
```

### Key Frontend Conventions

- **Monorepo**: Bun workspaces with `apps/*` and `packages/*`
- **API Layer**: wiretyped + Yup validation for all HTTP calls
- **State Management**: Zustand stores for global state
- **Error Handling**: `safeWrap`/`safeWrapAsync` utilities (no try/catch)
- **Styling**: Tailwind CSS only

## Migration Structure (`migrator/`)

```
migrator/
├── main.go            # Migration runner
└── migrations/        # SQL migration files
    ├── 0001_*.up.sql   # Forward migrations
    └── 0001_*.down.sql # Rollback migrations
```

## Configuration Files

- **Backend**: `backend/go.mod`, `.env`
- **Frontend**: `frontend/package.json`, `frontend/apps/platform/package.json`
- **Docker**: `docker-compose.yml`, `Dockerfile`
- **Proxy**: `Caddyfile` (routes traffic between services)
- **Build**: `Makefile` (development and build commands)

## Critical File Locations

- **API Contract**: `docs/API.md` - Complete API specification
- **Backend Rules**: `backend/AGENTS.md` - Non-negotiable coding conventions
- **Frontend Rules**: `frontend/AGENTS.md` - TypeScript and React conventions
- **Domain Types**: `backend/vibe/vibe.go` - ALL business logic types
- **API Schemas**: `frontend/apps/mobile/src/api/schemas/` - Yup validation schemas

## Development Workflow

1. **API First**: Changes start with `docs/API.md` updates
2. **Schema Validation**: Frontend uses Yup schemas matching API contract
3. **Domain Types**: Backend types defined in `vibe/vibe.go`
4. **Database Changes**: Migrations in `migrator/migrations/`
5. **Testing**: Backend tests with `go test`, frontend with type checking