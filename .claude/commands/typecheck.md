# Type Check

Run type checking for frontend and backend.

## Frontend (All Apps)

```bash
cd frontend && bun run typecheck
```

This checks both platform and cast apps plus all packages.

## Backend

```bash
cd backend && go build ./...
```

## Full Project Check

```bash
# From project root
make test          # Backend tests
cd frontend && bun run typecheck  # Frontend types
cd frontend && bun run lint       # Frontend linting
```

Fix any errors before committing. Both SSR apps must pass type checking.
