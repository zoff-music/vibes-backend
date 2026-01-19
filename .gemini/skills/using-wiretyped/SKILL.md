---
name: Using Wiretyped
description: Guide for using wiretyped for type-safe API calls
---

# Using Wiretyped

The frontend uses `wiretyped` for all API communication.

**CRITICAL RULE: NEVER use `fetch()`, `axios`, or `new EventSource()` directly. Use the typed clients.**

## API Client

Import the client from `@vibez/api`.

```typescript
import { api } from '@vibez/api';

// GET request
const [error, data] = await safeWrapAsync(api.get('/rooms/{id}', { id: '123' }));

// POST request
const [error, item] = await safeWrapAsync(api.post('/rooms/{id}/songs', {
    sourceType: 'youtube',
    sourceId: 'dQw4w9WgXcQ'
}, { id: '123' }));
```

## SSE (Server-Sent Events)

Use `useSSE` hook which uses `wiretyped` internally.

## Validation

All requests and responses are validated against Yup schemas defined in `frontend/packages/models/src/schemas/`.
