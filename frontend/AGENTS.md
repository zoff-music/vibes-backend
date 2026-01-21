# Frontend Coding Rules

Non-negotiable conventions. Follow strictly.

## Critical Rules

- **No `any` type** - use explicit types, compose when needed
- **No `@ts-ignore` or `@ts-nocheck`** - fix the type
- **No `try/catch`** - use `safeWrap`/`safeWrapAsync` from `@vibez/shared`
- **Use Biome** for ALL linting and formatting. No ESLint.
- **Run `bun run lint`** to check both format and lint rules before committing.
- **Use wiretyped + yup** for ALL API calls / SSE.
- **NEVER use `fetch()` or `new EventSource()`**. Only `wiretyped` clients.
- **SSR Support** - Both platform and cast apps support server-side rendering

## File Layout

```
apps/platform/src/
├── components/
│   ├── ui/                # Button, Input, Card, etc.
│   ├── player/            # VideoPlayer, PlayerControls
│   ├── queue/             # QueueItem, QueueList
│   ├── cast/              # CastButton, DeviceSelector
│   └── room/              # UserCount, room components
├── hooks/                 # Custom React hooks
├── stores/                # Zustand stores
├── pages/                 # Route components
├── services/              # castManager, etc.
├── server.tsx             # SSR server
└── client.tsx             # Client hydration

apps/cast/src/
├── App.tsx                # Cast receiver entrypoint
├── components/            # Cast-specific components
├── server.tsx             # SSR server
└── client.tsx             # Client hydration

packages/
├── api/                   # wiretyped API client
├── models/                # Shared types and Yup schemas
├── shared/                # Utilities, hooks, stores
│   ├── src/utils/wrap.ts  # safeWrap utilities
│   └── src/stores/        # Shared Zustand stores
└── player/                # Video player components
```

## Packages

- `@vibez/api`: API client (`import { api } from '@vibez/api'`)
- `@vibez/models`: Shared types and schemas (`import { ... } from '@vibez/models'`)
- `@vibez/shared`: Shared utilities (`import { ... } from '@vibez/shared'`)

## Error Handling

Never use try/catch. Use the wrap utilities from `@vibez/shared`:

```typescript
import { safeWrap, safeWrapAsync } from '@vibez/shared';

// Sync
const [result, error] = safeWrap(() => JSON.parse(data));
if (error) {
  // handle error
  return;
}
// use result

// Async
const [data, error] = await safeWrapAsync(api.get('/rooms'));
if (error) {
  // handle error
  return;
}
// use data
```

## Music Search

The backend supports multiple music search providers:
- YouTube: `/api/v1/youtube/search` and `/api/v1/youtube/videos/{id}`
- SoundCloud: `/api/v1/soundcloud/search` and `/api/v1/soundcloud/tracks/{id}`
- Spotify: `/api/v1/spotify/search` and `/api/v1/spotify/tracks/{id}`

> [!NOTE]
> SoundCloud and Spotify clients may be disabled if API keys are not configured. Check the `Enabled` property or handle 500 responses gracefully.

## API Calls

Always use wiretyped with yup validation:

```typescript
import { api } from '@vibez/api';

// Typed and validated
const [room, error] = await safeWrapAsync(api.post('/rooms', { name: 'My Room' }));
if (error) {
  // handle error
  return;
}
// use room
```

## Server-Side Rendering (SSR)

Both platform and cast apps support SSR:

```typescript
// server.tsx - SSR server
import { renderToReadableStream } from 'react-dom/server';
import { StaticRouter } from 'react-router';
import { safeWrapAsync } from '@vibez/shared';

// client.tsx - Client hydration
import { hydrateRoot } from 'react-dom/client';
```

## Styling

Use Tailwind CSS v4 with dark mode support. No inline styles or CSS-in-JS.

```tsx
// Good - with dark mode support
<button className="bg-white text-gray-900 dark:bg-gray-800 dark:text-white px-4 py-2 rounded-lg">

// Bad
<button style={{ backgroundColor: 'purple' }}>
```

## Component Guidelines

- Props interfaces defined above component
- Destructure props in function signature
- Export named components (not default)

```typescript
interface ButtonProps {
  variant: 'primary' | 'secondary';
  children: React.ReactNode;
  onClick?: () => void;
}

export function Button({ variant, children, onClick }: ButtonProps) {
  return (
    <button
      className={variant === 'primary' ? 'bg-primary' : 'bg-surface'}
      onClick={onClick}
    >
      {children}
    </button>
  );
}
```
