# Deployment Guide

Vibez uses a **Blue/Green Deployment** strategy orchestrated by Docker Compose and a Caddy reverse proxy. This ensures zero-downtime deployments and easy rollbacks.

## Architecture

### Components

1.  **Caddy Gateway (Load Balancer)**
    - Uses `docker-compose.prod.yml` -> `caddy` service.
    - Exposes ports `443` and `80` (or mapped high ports like `42069`).
    - Routes traffic dynamically to either the "Blue" or "Green" environment based on configuration.
    - Manages SSL certificates automatically.

2.  **Color Environments (Blue & Green)**
    - Each color consists of a **Backend** container and a **Frontend** container.
    - **Blue**: `vibes-backend-blue` + `vibes-frontend-blue`.
    - **Green**: `vibes-backend-green` + `vibes-frontend-green`.
    - They share the same database volume (`vibes_data`), as SQLite handles concurrent access well with WAL mode.

### Traffic Flow

```mermaid
graph TD
    User -->|HTTPS| Caddy[Caddy Gateway]
    Caddy -->|/api/*| Backend[Active Backend (Blue or Green)]
    Caddy -->|/*| Frontend[Active Frontend (Blue or Green)]
    
    subgraph Blue Env
    BackendBlue[Backend Blue]
    FrontendBlue[Frontend Blue]
    end
    
    subgraph Green Env
    BackendGreen[Backend Green]
    FrontendGreen[Frontend Green]
    end
```

## Deployment Process

The deployment is managed by `.github/deploy/deploy.sh`.

1.  **Identify Active Color**: The script checks Caddy's configuration to see which color is currently serving traffic.
2.  **Prepare Idle Color**:
    - If Blue is active, Green is prepared (and vice versa).
    - Checks out the latest code.
    - Builds new Docker images for the idle color.
    - Starts the idle containers (`docker compose up -d backend-green frontend-green`).
    - Waits for health checks to pass.
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

- **Caddy**: Proxies `https://localhost` to local ports.
- **Backend**: Runs on `:8080`.
- **Platform App**: Runs on `:3000`.
- **Cast App**: Runs on `:3001` (Mapped from `/casting/receiver/*`).

Run with: `make local-dev`

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
