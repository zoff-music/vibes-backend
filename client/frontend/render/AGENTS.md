# Frontend Coding Rules

Non-negotiable conventions. Follow strictly.

## Critical Rules

- **No `any` type** - use explicit types, compose when needed
- **Limit Return Values** - NEVER return more than 2 values. 3 or more is strictly illegal.
- **No `@ts-ignore` or `@ts-nocheck`** - fix the type
- **No `try/catch`** - use `safeWrap`/`safeWrapAsync` from `@vibez/shared`
- **Use Biome** for ALL linting and formatting. No ESLint.
- **Run `pnpm lint`** to check both format and lint rules before committing.
- **Run `pnpm typecheck`** to verify TypeScript compilation before committing.
- **Use wiretyped + yup** for ALL API calls / SSE.
- **NEVER use `fetch()` or `new EventSource()`**. Only `wiretyped` clients.
- **Unified Build System** - Apps build through the pnpm workspace
- **Content Hashing** - All assets use content-based hashing for cache busting

## File Layout

```
apps/platform/src/
├── components/
│   ├── ui/                # Deprecated; shared UI now lives in @vibez/ui
│   ├── player/            # Deprecated; shared player UI now lives in @vibez/ui
│   ├── queue/             # QueueItem, QueueList, AddToQueueModal
│   ├── cast/              # CastButton, DeviceSelector
│   └── room/              # UserCount, room components
├── hooks/                 # Custom React hooks (useRoom, useQueue, usePlayback, useSSE)
├── stores/                # Zustand stores (roomStore, queueStore, castStore, themeStore)
├── pages/                 # Route components (Home, CreateRoom, RoomView, Callback)
├── services/              # castManager, etc.
├── vite.config.ts         # Vite build
└── client.tsx             # Client hydration

apps/cast/src/
├── App.tsx                # Cast receiver entrypoint
├── components/            # Cast-specific components
├── vite.config.ts         # Vite build
└── client.tsx             # Client hydration

packages/
├── api/                   # wiretyped API client
├── models/                # Shared types and Yup schemas
├── shared/                # Utilities, hooks, stores
│   ├── src/utils/wrap.ts  # safeWrap utilities
│   ├── src/stores/        # Shared Zustand stores (playbackStore)
│   ├── src/hooks/         # Shared hooks (useProviderToken)
│   └── src/constants.ts   # Shared constants
└── ui/                    # Shared UI + player components (PlayerControls, SpotifyPlayer, VideoPlayer)
```

## Packages

- `@vibez/api`: API client (`import { api } from '@vibez/api'`)
- `@vibez/models`: Shared types and schemas (`import { Room, Song, PlaybackState } from '@vibez/models'`)
- `@vibez/shared`: Shared utilities (`import { safeWrap, usePlaybackStore, SourceType } from '@vibez/shared'`)
- `@vibez/ui`: Shared UI + player components (`import { SpotifyPlayer, VideoPlayer, Toast } from '@vibez/ui'`)

## Error Handling

Never use try/catch. Use the wrap utilities from `@vibez/shared`:

```typescript
import { safeWrap, safeWrapAsync } from '@vibez/shared';

// Sync
const [error, result] = safeWrap(() => JSON.parse(data));
if (error) {
  // handle error
  return;
}
// use result

// Async - CRITICAL: destructure as [error, data]
const [error, data] = await safeWrapAsync(api.get('/rooms/{id}', { id: roomId }));
if (error) {
  // handle error
  return;
}
// use data
```

**Important**: `safeWrapAsync` returns `[Error | null, T | null]` - always destructure as `[error, data]`, not `[data, error]`.

## API Calls

Always use wiretyped with yup validation:

```typescript
import { api } from '@vibez/api';

// GET request
const [error, room] = await safeWrapAsync(api.get('/rooms/{id}', { id: roomId }));

// POST request
const [error, song] = await safeWrapAsync(api.post('/rooms/{id}/songs', { id: roomId }, {
  sourceType: 'youtube',
  sourceId: 'dQw4w9WgXcQ',
  title: 'Never Gonna Give You Up',
  artist: 'Rick Astley',
  thumbnailUrl: 'https://img.youtube.com/vi/dQw4w9WgXcQ/maxresdefault.jpg',
  duration: 213
}));

// PUT request
const [error, playbackState] = await safeWrapAsync(api.put('/rooms/{id}/states', { id: roomId }, {
  action: 'play',
  positionMs: 45000
}));
```

## Server-Side Rendering (SSR)

The platform app uses React Router with Node.js serving:

```typescript
// Production server
import { renderToString } from 'react-dom/server';
import { StaticRouter } from 'react-router';

// Asset resolution via manifest
react-router-serve ./build/server/index.js
const mainJS = manifest['main.js'] ? `/assets/platform/${manifest['main.js']}` : '/assets/platform/client.js';

// Data injection for hydration
const initialData = {
  createRoomName: searchParams.get('name') || '',
  theme: 'dark'
};

// client.tsx - Client hydration
import { hydrateRoot } from 'react-dom/client';

// Extract SSR data
const initialData = (window as any).__INITIAL_DATA__ || {};
```

### Build System

Both apps use the pnpm workspace:

```bash
# Development (with watch mode and SSR)
pnpm dev

# Production build (with content hashing)
pnpm build
```

Build features:
- Content-based hashing for cache busting (`client-abc123.js`)
- Manifest generation for asset resolution
- CSS compilation with Tailwind v4
- SSR server bundling

### Hydration Pattern

Prevent hydration mismatches:

```typescript
const [isHydrated, setIsHydrated] = useState(false);

// Initialize with SSR data
const [name, setName] = useState(() => {
  if (initialData?.createRoomName) {
    return initialData.createRoomName;
  }
  return '';
});

useEffect(() => {
  setIsHydrated(true);
  // Fix hydration mismatch: ensure client state matches server state
  if (initialData?.createRoomName) {
    setName(initialData.createRoomName);
  }
}, []);

// Render different content during hydration if needed
if (!isHydrated) {
  return <div>Loading...</div>;
}
```

## State Management (Zustand)

### Room Store (`useRoomStore`)
```typescript
const { room, users, userId, isAdmin, setRoom, setSession } = useRoomStore();
```

### Queue Store (`useQueueStore`)
```typescript
const { songs, setSongs, addSong, removeSong, reorderSongs } = useQueueStore();
```

### Playback Store (`usePlaybackStore`)
```typescript
const { currentSong, isPlaying, positionMs, actualPositionMs, setPlaybackState } = usePlaybackStore();
```

### Theme Store (`useThemeStore`)
```typescript
const { isDarkMode, toggleDarkMode } = useThemeStore();
```

### Cast Store (`useCastStore`)
```typescript
const { isConnected, availableDevices, connectToDevice, castCurrentSong } = useCastStore();
```

## Custom Hooks

### useRoom(roomId)
```typescript
const { room, users, fetchRoom, joinRoom, updateRoomSettings } = useRoom(roomId);
```

### useQueue(roomId)
```typescript
const { songs, fetchQueue, addToQueue, removeFromQueue, moveInQueue } = useQueue(roomId);
```

### usePlayback(roomId)
```typescript
const { currentSong, isPlaying, play, pause, seek, skip, vote } = usePlayback(roomId);
```

### useSSE(roomId)
```typescript
// Automatically manages SSE connection lifecycle
// Reference counting for multiple subscribers
// Grace period before cleanup (2 seconds)
useSSE(roomId);
```

## Real-time Updates (SSE)

SSE events are handled automatically by `useSSE` hook:

```typescript
// Event types received:
// - playback_update: Playback state changed
// - song_added: Song added to queue
// - song_removed: Song removed from queue
// - songs_update: Queue reordered
// - users_update: User count updated
// - settings_update: Room settings changed
```

## Music Providers

The app supports multiple music providers:

```typescript
// Source types
type SourceType = 'youtube' | 'spotify' | 'soundcloud';

// Provider authentication
const { getToken, isAuthenticated } = useProviderToken('spotify');

// Search across providers
const [error, results] = await safeWrapAsync(api.get('/youtube/search', {}, { q: 'never gonna give you up' }));
const [error, results] = await safeWrapAsync(api.get('/spotify/search', {}, { q: 'never gonna give you up' }));
const [error, results] = await safeWrapAsync(api.get('/soundcloud/search', {}, { q: 'never gonna give you up' }));
```

## Styling

Use Tailwind CSS v4 with dark mode support. No inline styles or CSS-in-JS.

```tsx
// Good - with dark mode support
<button className="bg-white text-gray-900 dark:bg-gray-800 dark:text-white px-4 py-2 rounded-lg transition-colors duration-200">

// Glass morphism pattern (existing design language)
<div className="glass p-4 rounded-xl hover:shadow-retro active:scale-95 transition-all">

// Responsive design
<div className="text-sm md:text-base lg:text-lg">

// Bad
<button style={{ backgroundColor: 'purple' }}>
```

## Component Guidelines

- Props interfaces defined above component
- Destructure props in function signature
- Export named components (not default)
- Support dark mode
- Include proper focus states for accessibility

```typescript
interface ButtonProps {
  variant: 'primary' | 'secondary';
  children: React.ReactNode;
  onClick?: () => void;
  disabled?: boolean;
}

export function Button({ variant, children, onClick, disabled }: ButtonProps) {
  return (
    <button
      className={`
        px-4 py-2 rounded-lg font-medium transition-colors duration-200
        focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2
        dark:focus:ring-offset-gray-800
        ${variant === 'primary' 
          ? 'bg-primary text-white hover:bg-primary-dark dark:bg-primary-light dark:hover:bg-primary' 
          : 'bg-gray-200 text-gray-900 hover:bg-gray-300 dark:bg-gray-700 dark:text-white dark:hover:bg-gray-600'
        }
        ${disabled ? 'opacity-50 cursor-not-allowed' : ''}
      `}
      onClick={onClick}
      disabled={disabled}
    >
      {children}
    </button>
  );
}
```

## Room Modes

Handle different room modes in the UI:

```typescript
// Server mode: Server controls playback automatically
if (room?.mode === 'server') {
  // Show play/pause controls for all users
  // Show skip voting UI
}

// Host mode: Only host can control playback
if (room?.mode === 'host') {
  if (room.hostId === userId || isAdmin) {
    // Show full playback controls
  } else {
    // Show limited UI (add songs, vote)
  }
}
```

## Casting Integration

```typescript
// Initialize casting
const { initialize, connectToDevice, castCurrentSong } = useCastStore();

useEffect(() => {
  initialize();
}, []);

// Cast current song
const handleCast = async (deviceId: string) => {
  await connectToDevice(deviceId);
  if (currentSong) {
    await castCurrentSong(currentSong);
  }
};
```

## Development Workflow

1. **Type Safety**: Run `pnpm typecheck` before committing
2. **Code Quality**: Run `pnpm lint` for formatting and linting
3. **API Integration**: Use wiretyped + yup for all API calls
4. **Error Handling**: Use `safeWrap`/`safeWrapAsync` utilities
5. **State Management**: Use Zustand stores for global state
6. **Styling**: Use Tailwind CSS v4 with dark mode support
7. **SSR**: Test both server-side rendering and client hydration
