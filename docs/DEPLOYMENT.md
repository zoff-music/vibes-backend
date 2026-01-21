# Deployment Guide

Vibez uses a **Blue/Green Deployment** strategy orchestrated by Docker Compose and a Caddy reverse proxy. This ensures zero-downtime deployments and easy rollbacks.

## Architecture

### Components

1.  **Caddy Gateway (Load Balancer)**
    - Uses `docker-compose.yml` -> `caddy` service
    - Exposes ports `443` and `80` (configurable via environment variables)
    - Routes traffic dynamically to either the "Blue" or "Green" environment based on configuration
    - Manages SSL certificates automatically with `local_certs` for development

2.  **Color Environments (Blue & Green)**
    - Each color consists of **Backend**, **Platform Frontend**, and **Cast Frontend** containers
    - **Blue**: `backend-blue` + `frontend-platform-blue` + `frontend-cast-blue`
    - **Green**: `backend-green` + `frontend-platform-green` + `frontend-cast-green`
    - All services support SSR for improved performance
    - They share the same database volume (`vibes_db_shared`), as SQLite handles concurrent access well with WAL mode

3.  **Database Migration**
    - **Migrator**: Runs automatically before backend services start
    - Ensures database schema is up-to-date before application startup

### Traffic Flow

```mermaid
graph TD
    User -->|HTTPS| Caddy[Caddy Gateway]
    Caddy -->|/api/*| Backend[Active Backend (Blue or Green)]
    Caddy -->|/casting/receiver/*| CastApp[Active Cast App (Blue or Green)]
    Caddy -->|/*| PlatformApp[Active Platform App (Blue or Green)]
    
    subgraph Blue Env
    BackendBlue[Backend Blue]
    PlatformBlue[Platform Blue - SSR]
    CastBlue[Cast Blue - SSR]
    end
    
    subgraph Green Env
    BackendGreen[Backend Green]
    PlatformGreen[Platform Green - SSR]
    CastGreen[Cast Green - SSR]
    end
```

## Deployment Process

The deployment is managed by `.github/deploy/deploy.sh`.

1.  **Identify Active Color**: The script checks Caddy's configuration to see which color is currently serving traffic.
2.  **Prepare Idle Color**:
    - If Blue is active, Green is prepared (and vice versa)
    - Checks out the latest code
    - Builds new Docker images for the idle color (backend, platform, cast)
    - Runs database migrations via the migrator service
    - Starts the idle containers (`docker compose up -d backend-green frontend-platform-green frontend-cast-green`)
    - Waits for health checks to pass on all services
3.  **Switch Traffic**:
    - Updates Caddy configuration to point to the new color's IP/Container.
    - Caddy reloads configuration gracefully with zero downtime.
4.  **Cleanup**:
    - Stops the old color's containers to save resources (optional, currently kept for quick rollback).

## Caddy Configuration

### Production (`Caddyfile.prod`)
Handles the routing logic. It defines upstream variables that are swapped during deployment.

```caddyfile
# Dynamic upstreams managed by deploy script
@blue {
    header Cookie *vibez_color=blue*
}

handle /* {
    reverse_proxy {$ACTIVE_FRONTEND_HOST}:3000
}
```

### Local Development
Locally, we use a simpler setup defined in `Makefile`:

- **Caddy**: Proxies `https://localhost` to local ports with automatic SSL certificates
- **Backend**: Runs on `:8080` with automatic database migrations
- **Platform App**: Runs on `:3000` with SSR support
- **Cast App**: Runs on `:3001` with SSR support (Mapped from `/casting/receiver/*`)

Run with: `make local-dev`

## Frontend Architecture

The frontend now uses **Server-Side Rendering (SSR)** for both applications:

1.  **Platform App**: Main application served at root `/` with SSR for better performance
2.  **Cast Receiver**: Chromecast receiver served at `/casting/receiver/` with SSR for faster loading
3.  **SSR Benefits**: 
    - Faster initial page loads
    - Better SEO and social media sharing
    - Improved performance on slower devices
    - Room data prefetching for platform app
4.  **Development**: Both apps support hot module replacement with SSR during development

## Frontend Serving

The **Frontend Container** (`caddy:alpine`) serves static files for both applications:

1.  **Platform App**: Served at root `/`.
2.  **Cast Receiver**: Served at `/casting/receiver/`.
3.  **Routing**: Handled by an internal `Caddyfile` inside the frontend container.

```caddyfile
:3000 {
    handle_path /casting/receiver/* {
        root * /srv/cast-app
        try_files {path} {path}/ /index.html
    }
    handle {
        root * /srv/app
        try_files {path} {path}/ /index.html
    }
}
```
