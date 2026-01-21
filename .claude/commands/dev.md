# Start Development

## Recommended (HTTPS with SSR)

Run the full stack (Backend + Frontend SSR + Caddy) with HTTPS enabled:

```bash
make local-dev
```

- **App**: [https://localhost](https://localhost) (Platform with SSR)
- **Cast**: [https://localhost/casting/receiver/](https://localhost/casting/receiver/) (Cast with SSR)
- **API**: [https://localhost/api](https://localhost/api)

## Docker Development

```bash
make dev
```

Full stack via Docker Compose with SSR-enabled frontend services.

## Manual (HTTP)

1. Start backend:
```bash
cd backend && go run cmd/server/main.go
```

2. Start frontend (both apps with SSR):
```bash
cd frontend && bun dev
```
