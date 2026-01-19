# Safe Error Handling

**CRITICAL RULE: Do NOT use `try/catch` blocks in frontend code.**

Use the `safeWrap` utilities from `@vibez/shared` (or `src/utils/wrap`).

## Async (Promises)

```typescript
import { safeWrapAsync } from '@vibez/shared';

const [error, data] = await safeWrapAsync(promise);

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
    // handle error
}
```
