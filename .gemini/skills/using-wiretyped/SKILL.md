---
name: Using Wiretyped
description: Guide for using wiretyped for type-safe API calls
---

# Using Wiretyped

The frontend uses `wiretyped` for all API communication with full TypeScript types and Yup validation.

**CRITICAL RULE: NEVER use `fetch()`, `axios`, or `new EventSource()` directly. Use the typed clients.**

## API Client

Import the client from `@vibez/api` and use with `safeWrapAsync`:

```typescript
import { api } from '@vibez/api';
import { safeWrapAsync } from '@vibez/shared';

// GET request with path parameters
const [error, room] = await safeWrapAsync(
    api.get('/rooms/{id}', { id: roomId })
);

if (error) {
    console.error('Failed to get room:', error);
    return;
}

// room is fully typed and validated
console.log(room.name, room.settings.skipAllowed);

// POST request with body and path parameters
const [error, song] = await safeWrapAsync(
    api.post('/rooms/{id}/songs', { id: roomId }, {
        sourceType: 'youtube',
        sourceId: 'dQw4w9WgXcQ',
        title: 'Never Gonna Give You Up',
        artist: 'Rick Astley',
        thumbnailUrl: 'https://img.youtube.com/vi/dQw4w9WgXcQ/maxresdefault.jpg',
        duration: 213
    })
);

// PUT request for playback control
const [error, playbackState] = await safeWrapAsync(
    api.put('/rooms/{id}/states', { id: roomId }, {
        action: 'play',
        positionMs: 45000
    })
);

// DELETE request
const [error] = await safeWrapAsync(
    api.delete('/rooms/{id}/songs/{songId}', { id: roomId, songId: songId })
);
```

## Query Parameters

For endpoints with query parameters (like search):

```typescript
// Search with query parameters
const [error, results] = await safeWrapAsync(
    api.get('/youtube/search', {}, { q: 'never gonna give you up' })
);

const [error, results] = await safeWrapAsync(
    api.get('/spotify/search', {}, { q: 'bohemian rhapsody' })
);
```

## SSE (Server-Sent Events)

Use `useSSE` hook which uses `wiretyped` internally for real-time updates:

```typescript
import { useSSE } from '../hooks/useSSE';

// SSE connection with automatic management
export function MyComponent({ roomId }: { roomId: string }) {
    useSSE(roomId); // Automatically connects and handles events
    
    // Events are automatically handled by stores:
    // - playback_update -> usePlaybackStore
    // - song_added -> useQueueStore
    // - users_update -> useRoomStore
    // - etc.
}
```

## Custom Hooks Integration

The wiretyped client is integrated with custom hooks:

```typescript
// useRoom hook
const { room, fetchRoom, updateRoomSettings } = useRoom(roomId);

// useQueue hook  
const { songs, addToQueue, removeFromQueue } = useQueue(roomId);

// usePlayback hook
const { currentSong, isPlaying, play, pause, skip } = usePlayback(roomId);
```

## Error Handling Patterns

Always use `safeWrapAsync` with proper error handling:

```typescript
const handleAddSong = async (songData: AddSongRequest) => {
    const [error, song] = await safeWrapAsync(
        api.post('/rooms/{id}/songs', { id: roomId }, songData)
    );
    
    if (error) {
        // Handle specific error cases
        if (error.message.includes('duplicate')) {
            setError('Song already in queue');
        } else if (error.message.includes('rate limit')) {
            setError('Too many songs added. Please wait.');
        } else {
            setError('Failed to add song');
        }
        return;
    }
    
    // Success - song is fully typed
    console.log(`Added: ${song.title} by ${song.artist}`);
};
```

## SSR Usage

In server-side rendering contexts, use the API client with proper error handling:

```typescript
// server.tsx
import { api } from '@vibez/api';
import { safeWrapAsync } from '@vibez/shared';

// Pre-fetch data for SSR
const url = new URL(request.url);
const roomId = url.pathname.split('/').pop();

const [error, room] = await safeWrapAsync(
    api.get('/rooms/{id}', { id: roomId })
);

if (error || !room || room.id === '') {
    // Handle error or redirect
    return new Response('Room not found', { status: 404 });
}

// Inject data for client hydration
const initialData = {
    room,
    // other SSR data
};
```

## Type Safety

All API calls are fully typed based on the API contract:

```typescript
// TypeScript knows the exact shape of responses
const [error, room] = await safeWrapAsync(api.get('/rooms/{id}', { id: roomId }));

if (room) {
    // Full IntelliSense support
    room.name;           // string
    room.settings;       // RoomSettings
    room.mode;          // "server" | "host"
    room.isAdmin;       // boolean
    room.userCount;     // number
    room.activeSources; // string[]
}

// Request bodies are also typed
const [error, song] = await safeWrapAsync(
    api.post('/rooms/{id}/songs', { id: roomId }, {
        sourceType: 'youtube', // Only valid source types allowed
        sourceId: 'abc123',    // string required
        title: 'Song Title',   // string required
        duration: 180,         // number required
        // TypeScript will error if required fields are missing
    })
);
```

## Validation

All requests and responses are validated against Yup schemas defined in `frontend/packages/models/src/schemas/`. The wiretyped client automatically:

- Validates request bodies before sending
- Validates response data after receiving
- Provides detailed error messages for validation failures
- Ensures type safety at runtime, not just compile time

## Provider-Specific Endpoints

Handle different music providers:

```typescript
// YouTube search
const [error, results] = await safeWrapAsync(
    api.get('/youtube/search', {}, { q: query })
);

// Spotify search (requires authentication)
const [error, results] = await safeWrapAsync(
    api.get('/spotify/search', {}, { q: query })
);

// Get track details
const [error, track] = await safeWrapAsync(
    api.get('/youtube/videos/{id}', { id: videoId })
);

// OAuth token management
const [error, token] = await safeWrapAsync(
    api.get('/tokens/spotify')
);
```
