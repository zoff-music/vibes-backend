---
name: Using Wiretyped
description: Guide for using wiretyped for type-safe API calls
---

# Using Wiretyped

The frontend uses `wiretyped` for all API communication.

**CRITICAL RULE: NEVER use `fetch()`, `axios`, or `new EventSource()` directly. Use the typed clients.**

## API Client

Import the client from `@vibez/api` and use with `safeWrapAsync`:

```typescript
import { api } from '@vibez/api';
import { safeWrapAsync } from '@vibez/shared';

// GET request
const [room, error] = await safeWrapAsync(
    api.get('/rooms/{id}', { id: '123' })
);

if (error) {
    console.error('Failed to get room:', error);
    return;
}

// room is typed and safe to use
console.log(room.name);

// POST request
const [song, error] = await safeWrapAsync(
    api.post('/rooms/{id}/songs', {
        sourceType: 'youtube',
        sourceId: 'dQw4w9WgXcQ'
    }, { id: '123' })
);
```

## SSE (Server-Sent Events)

Use `useSSE` hook which uses `wiretyped` internally:

```typescript
import { useSSE } from '@vibez/shared';

const { events, error } = useSSE(`/rooms/${roomId}/events`);
```

## SSR Usage

In server-side rendering contexts, use the API client with proper error handling:

```typescript
// server.tsx
import { api } from '@vibez/api';
import { safeWrapAsync } from '@vibez/shared';

const [room, err] = await safeWrapAsync(
    api.get('/rooms/{id}', { id: roomId })
);

if (err || !room) {
    // Handle error or redirect
    return { data: {}, redirect: Response.redirect(...) };
}
```

## Validation

All requests and responses are validated against Yup schemas defined in `frontend/packages/models/src/schemas/`. The wiretyped client automatically handles validation and provides full TypeScript types.
