# Technology Stack

## Architecture

**Monorepo** structure with separate backend, frontend, and migration tools.

## Backend Stack

- **Language**: Go 1.23+
- **Database**: SQLite with `github.com/mattn/go-sqlite3` driver (CGO-enabled)
- **HTTP Framework**: Gorilla Mux for routing
- **Real-time**: Server-Sent Events (SSE) for live updates
- **Monitoring**: OpenTelemetry + Prometheus metrics
- **External APIs**: YouTube Data API v3, Spotify Web API, SoundCloud API

## Frontend Stack

- **Framework**: React 19 with TypeScript
- **Build Tool**: Vite
- **Styling**: Tailwind CSS v4 with dark mode support
- **State Management**: Zustand stores
- **API Client**: wiretyped with Yup schema validation
- **Package Manager**: Bun
- **Animation**: Framer Motion
- **Video Player**: react-player, react-youtube
- **Theme**: Dark mode enabled with system preference detection

## Development Tools

- **Containerization**: Docker + Docker Compose
- **Reverse Proxy**: Caddy
- **Database Migrations**: Custom Go migrator tool
- **Security**: gosec, govulncheck

## Styling & Theming

- **CSS Framework**: Tailwind CSS v4
- **Dark Mode**: Enabled with automatic system preference detection
- **Theme Toggle**: Available for manual light/dark mode switching
- **Color Scheme**: Custom design system with dark mode variants
- **Responsive Design**: Mobile-first approach with responsive breakpoints

## Common Commands

### Development Setup
```bash
# Full stack with Docker
make dev

# Backend only
cd backend && go run cmd/server/main.go

# Frontend only  
cd frontend && bun dev

# Local development (both)
# Local development (HTTPS via Caddy)
make local-dev
# URL: https://localhost
```

### Database Operations
```bash
# Run migrations
make migrate-up

# Rollback migrations
make migrate-down STEPS=1

# Build migrator
make build-migrator
```

### Testing & Quality
```bash
# Run backend tests
make test

# Type checking
cd frontend && bun run typecheck

# Security scanning
make gosec
make govulncheck

# Linting
cd frontend && bun run lint
```

### Build & Deploy
```bash
# Build backend binary
make build

# Build frontend
cd frontend && bun run build

# Docker containers
make docker
```

## Environment Variables

### Backend
```bash
PORT=8080
DATABASE_PATH=./data/vibes.db
YOUTUBE_API_KEY=required
```

### Frontend
```bash
VITE_API_URL=http://localhost:8080
```

## Package Management

- **Backend**: Go modules (`go mod`)
- **Frontend**: Bun workspaces with monorepo support
- **Workspaces**: `apps/*` and `packages/*` for shared code