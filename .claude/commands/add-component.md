# Add Frontend Component

Create a React component following project conventions.

## Requirements

Provide:
- Component name
- Props needed
- Which folder (ui, player, queue, cast, room, or new)
- Which app (platform or cast)

## Template

File: `frontend/apps/{app}/src/components/{folder}/{Name}.tsx`

```tsx
interface {Name}Props {
  // Define props with explicit types
  children?: React.ReactNode;
  className?: string;
  onClick?: () => void;
  disabled?: boolean;
}

export function {Name}({ children, className, onClick, disabled, ...props }: {Name}Props) {
  return (
    <div 
      className={`
        bg-white text-gray-900 border border-gray-200 rounded-lg p-4
        dark:bg-gray-800 dark:text-white dark:border-gray-700
        transition-colors duration-200
        focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2
        dark:focus:ring-offset-gray-800
        ${disabled ? 'opacity-50 cursor-not-allowed' : 'hover:bg-gray-50 dark:hover:bg-gray-700'}
        ${className || ''}
      `}
      onClick={disabled ? undefined : onClick}
      {...props}
    >
      {children}
    </div>
  );
}
```

## Component Folders

### Platform App (`frontend/apps/platform/src/components/`)
- `ui/` - Base components (Button, Input, Text, Toast, Card)
- `player/` - Video player components (PlayerControls)
- `queue/` - Queue management (QueueItem, QueueList, AddToQueueModal)
- `cast/` - Casting components (CastButton, DeviceSelector)
- `room/` - Room-specific components (UserCount)

### Cast App (`frontend/apps/cast/src/components/`)
- Cast receiver specific components

### Shared Components (`frontend/packages/shared/src/components/`)
- Components used across multiple apps

## Styling Guidelines

### Dark Mode Support
Always include dark mode variants:
```css
bg-white dark:bg-gray-800
text-gray-900 dark:text-white
border-gray-200 dark:border-gray-700
```

### Responsive Design
Use mobile-first approach:
```css
text-sm md:text-base lg:text-lg
p-2 md:p-4
```

### Accessibility
Include proper focus states:
```css
focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2
dark:focus:ring-offset-gray-800
```

### Glass Morphism (Existing Pattern)
For special components:
```css
glass p-4 rounded-xl hover:shadow-retro active:scale-95 transition-all
```

## State Management

Use appropriate hooks and stores:

```tsx
import { useRoomStore } from '../stores/roomStore';
import { usePlaybackStore } from '@vibez/shared';
import { safeWrapAsync } from '@vibez/shared';
import { api } from '@vibez/api';

export function MyComponent() {
  const { room, isAdmin } = useRoomStore();
  const { currentSong, isPlaying } = usePlaybackStore();

  const handleAction = async () => {
    const [error, result] = await safeWrapAsync(
      api.post('/rooms/{id}/action', { id: room?.id }, { action: 'play' })
    );
    
    if (error) {
      console.error('Action failed:', error);
      return;
    }
    
    // Handle success
  };

  return (
    <div>
      {/* Component content */}
    </div>
  );
}
```

## Checklist

- [ ] Named export (not default)
- [ ] Props interface above component
- [ ] Tailwind CSS v4 with dark mode support
- [ ] No `any` types
- [ ] Error handling with `safeWrap`/`safeWrapAsync` from `@vibez/shared`
- [ ] Responsive design (mobile-first)
- [ ] Accessibility (focus states, ARIA labels)
- [ ] Proper TypeScript types
- [ ] Consistent with existing design system
- [ ] SSR-compatible (no browser-only code in render)

## API Integration

For components that interact with the API:

```tsx
import { api } from '@vibez/api';
import { safeWrapAsync } from '@vibez/shared';

const [error, data] = await safeWrapAsync(
  api.get('/rooms/{id}', { id: roomId })
);
```

## Room Mode Handling

Handle different room modes appropriately:

```tsx
// Server mode: Show controls for all users
if (room?.mode === 'server') {
  return <ServerModeControls />;
}

// Host mode: Show controls only for host/admin
if (room?.mode === 'host' && (room.hostId === userId || isAdmin)) {
  return <HostModeControls />;
}

return <LimitedControls />;
```
