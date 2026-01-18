# Frontend Coding Rules

Non-negotiable conventions. Follow strictly.

## Critical Rules

- **No `any` type** - use explicit types, compose when needed
- **No `@ts-ignore` or `@ts-nocheck`** - fix the type
- **No `try/catch`** - use `safeWrap`/`safeWrapAsync` from `src/utils/wrap.ts`
- **Use wiretyped + yup** for ALL API calls / SSE.
- **NEVER use `fetch()` or `new EventSource()`**. Only `wiretyped` clients.

## File Layout

```
apps/mobile/src/
├── api/
│   ├── client.ts          # wiretyped client
│   └── schemas/           # yup schemas
├── components/
│   ├── ui/                # Button, Input, Card, etc.
│   ├── player/            # VideoPlayer, PlayerControls
│   └── queue/             # QueueItem, QueueList
├── hooks/
│   ├── useRoom.ts
│   ├── useQueue.ts
│   ├── usePlayback.ts
│   └── useSSE.ts
├── stores/                # Zustand stores
│   ├── roomStore.ts
│   ├── queueStore.ts
│   └── playbackStore.ts
├── pages/                 # Route components
└── utils/
    └── wrap.ts            # Error handling utilities
```

## Error Handling

Never use try/catch. Use the wrap utilities:

```typescript
import { safeWrap, safeWrapAsync } from '@/utils/wrap';

// Sync
const [result, error] = safeWrap(() => JSON.parse(data));

// Async
const [data, error] = await safeWrapAsync(api.get('/rooms'));
if (error) {
  // handle error
  return;
}
// use data
```

## API Calls

Always use wiretyped with yup validation:

```typescript
import { api } from '@/api/client';

// Typed and validated
const room = await api.post('/rooms', { name: 'My Room' });
```

## Styling

Use Tailwind CSS. No inline styles or CSS-in-JS.

```tsx
// Good
<button className="bg-primary text-white px-4 py-2 rounded-lg">

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
