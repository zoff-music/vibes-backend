# Vibez Frontend

A React + Vite + TypeScript monorepo using Bun workspaces with Server-Side Rendering (SSR).

## Applications

- **`apps/platform`**: The main web application for room management, queueing, and social interaction (SSR-enabled)
- **`apps/cast`**: A standalone Chromecast Receiver application for synchronized playback on Google Cast devices (SSR-enabled)

## Shared Packages

- **`packages/api`**: Type-safe API client generated with wiretyped
- **`packages/models`**: Shared domain types, interfaces, and validation schemas
- **`packages/shared`**: Shared React hooks, utilities, and Zustand stores (includes safeWrap error handling)
- **`packages/player`**: Shared video player components for YouTube, Spotify, and SoundCloud

## Development

```bash
# Install dependencies
bun install

# Run the main platform app (Port 3000, SSR-enabled)
bun dev

# Run all apps (Platform + Cast, both with SSR)
bun run dev --filter '*'

# Run with HTTPS via Caddy (Recommended)
make local-dev  # From project root
```

## Server-Side Rendering (SSR)

Both applications now support SSR for improved performance and SEO:

- **Platform App**: SSR with room data prefetching
- **Cast App**: SSR for faster Chromecast loading
- **Development**: Hot module replacement with SSR
- **Production**: Optimized SSR builds

## Tooling

- **Linting & Formatting**: [Biome](https://biomejs.dev/) (`bun run lint`, `bun run fix`)
- **Type Checking**: TypeScript (`bun run typecheck`)
- **Testing**: Vitest & Playwright
- **Error Handling**: `safeWrap`/`safeWrapAsync` utilities (no try/catch)
- **Styling**: Tailwind CSS v4 with dark mode support

## Key Features

- **Dark Mode**: System preference detection with manual toggle
- **Error Handling**: Safe error handling with `safeWrap` utilities
- **Type Safety**: Full TypeScript with wiretyped API client
- **Real-time**: SSE integration for live updates
- **Responsive**: Mobile-first design with Tailwind CSS v4

## Rules

Please read the [AGENTS.md](./AGENTS.md) for non-negotiable frontend coding conventions and file layout rules.
