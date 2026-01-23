# Type Check

Run type checking for frontend and backend.

## Frontend (All Apps)

```bash
cd frontend && bun run typecheck
```

This checks:
- Platform app (`apps/platform/`)
- Cast app (`apps/cast/`)
- All packages (`packages/api/`, `packages/models/`, `packages/shared/`, `packages/player/`)

## Backend

```bash
cd backend && go build ./...
```

Checks all Go packages including:
- Main server (`cmd/server/`)
- Client packages (`client/database/`, `client/youtube/`, etc.)
- Domain types (`vibe/`)
- Handlers (`server/internal/handler/`)

## Full Project Check

```bash
# From project root
make test                         # Backend tests + build
cd frontend && bun run typecheck  # Frontend TypeScript
cd frontend && bun run lint       # Frontend linting (Biome)
```

## Individual App Checks

```bash
# Platform app only
cd frontend/apps/platform && bun run typecheck

# Cast app only
cd frontend/apps/cast && bun run typecheck

# Specific package
cd frontend/packages/shared && bun run typecheck
```

## Common Issues

### Frontend TypeScript Errors
- Missing types in `@vibez/models` package
- Incorrect API call signatures (check `@vibez/api`)
- SSR hydration type mismatches
- Missing props in component interfaces

### Backend Build Errors
- Missing interfaces in `vibe/` directory
- Incorrect prepared statement names
- Import path issues between packages
- Missing error wrapping

## Fix Before Committing

Both SSR apps and all packages must pass type checking. The build system requires:
- Zero TypeScript errors
- Clean Biome linting
- Successful Go build
- All tests passing
