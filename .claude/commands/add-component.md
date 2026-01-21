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
}

export function {Name}({ ...props }: {Name}Props) {
  return (
    <div className="bg-white dark:bg-gray-800 text-gray-900 dark:text-white">
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

## Shared Components

For components used across apps, create in:
`frontend/packages/shared/src/components/{Name}.tsx`
