# Vibez Frontend

A React + Vite + TypeScript monorepo using Bun workspaces.

## Applications

- **`apps/platform`**: The main web application for room management, queueing, and social interaction.
- **`apps/cast`**: A standalone Chromecast Receiver application for synchronized playback on Google Cast devices.

## Shared Packages

- **`packages/api`**: Type-safe API client generated with wiretyped.
- **`packages/models`**: Shared domain types, interfaces, and validation schemas.
- **`packages/shared`**: Shared React hooks, utilities, and Zustand stores.

## Development

```bash
bun install

# Run the main platform app (Port 3000)
bun dev

# Run all apps (Platform + Cast)
bun run dev --filter '*'
```

## Tooling

- **Linting & Formatting**: [Biome](https://biomejs.dev/) (`bun run lint`, `bun run fix`)
- **Type Checking**: TypeScript (`bun run typecheck`)
- **Testing**: Vitest & Playwright

## Rules

Please read the [AGENTS.md](./AGENTS.md) for non-negotiable frontend coding conventions and file layout rules.
