# Start Development

Start frontend and backend dev servers.

## Steps

1. Start backend:
```bash
cd backend && go build ./cmd/server && ./server
```

2. Start frontend:
```bash
cd frontend && bun dev
```

Both run concurrently. Backend on :8080, frontend on :5173.
