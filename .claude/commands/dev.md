# Start Development

## Recommended (HTTPS)

Run the full stack (Backend + Frontend + Caddy) with HTTPS enabled:

```bash
make local-dev
```

- **App**: [https://localhost](https://localhost)
- **API**: [https://localhost/api](https://localhost/api)

## Manual (HTTP)

1. Start backend:
```bash
cd backend && go build ./cmd/server && ./server
```

2. Start frontend:
```bash
cd frontend && bun dev
```
