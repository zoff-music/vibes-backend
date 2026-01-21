# Start Development

## Recommended (HTTPS with Unified SSR)

Run the full stack (Backend + Frontend SSR + Caddy) with HTTPS enabled:

```bash
make local-dev
```

- **App**: [https://localhost](https://localhost) (Platform with Bun SSR)
- **Cast**: [https://localhost/casting/receiver/](https://localhost/casting/receiver/) (Cast with Bun SSR)
- **API**: [https://localhost/api](https://localhost/api)

Both apps use unified Bun-based build system with content hashing and manifest-based asset resolution.

## Docker Development

```bash
make dev
```

Full stack via Docker Compose with unified SSR-enabled frontend services.

## Manual (HTTP)

1. Start backend:
```bash
cd backend && go run cmd/server/main.go
```

2. Start frontend (both apps with unified SSR):
```bash
cd frontend && bun dev
```

Platform: http://localhost:3000 (Bun SSR)
Cast: http://localhost:3001 (Bun SSR)
