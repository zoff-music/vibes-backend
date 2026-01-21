---
name: Safe Error Handling
description: Guide for error handling without try/catch
---

# Safe Error Handling

**CRITICAL RULE: Do NOT use `try/catch` blocks in frontend code.**

Use the `safeWrap` utilities from `@vibez/shared`.

## Async (Promises)

```typescript
import { safeWrapAsync } from '@vibez/shared';

const [data, error] = await safeWrapAsync(promise);

if (error) {
    console.error('Failed:', error);
    return;
}

// data is safe to use here
console.log(data);
```

## Sync (Functions)

```typescript
import { safeWrap } from '@vibez/shared';

const [result, error] = safeWrap(() => JSON.parse(jsonString));

if (error) {
    console.error('Parse failed:', error);
    return;
}

// result is safe to use here
console.log(result);
```

## API Calls

```typescript
import { api } from '@vibez/api';
import { safeWrapAsync } from '@vibez/shared';

const [room, error] = await safeWrapAsync(
    api.get('/rooms/{id}', { id: roomId })
);

if (error) {
    // Handle API error
    return;
}

// room is typed and safe to use
console.log(room.name);
```

## SSR Usage

Safe error handling is especially important in SSR contexts:

```typescript
// server.tsx
const [ssrErr, stream] = await safeWrapAsync(
    renderToReadableStream(...)
);

if (ssrErr || !stream) {
    console.error('[SSR Error]', ssrErr);
    return new Response('Internal Server Error', { status: 500 });
}
```
