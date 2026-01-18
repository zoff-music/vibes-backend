# Add Frontend Component

Create a React component following project conventions.

## Requirements

Provide:
- Component name
- Props needed
- Which folder (ui, player, queue, or new)

## Template

File: `frontend/apps/mobile/src/components/{folder}/{Name}.tsx`

```tsx
interface {Name}Props {
  // Define props
}

export function {Name}({ ...props }: {Name}Props) {
  return (
    <div className="">
      {/* Component content */}
    </div>
  );
}
```

## Checklist

- [ ] Named export (not default)
- [ ] Props interface above component
- [ ] Tailwind for styling
- [ ] No `any` types
- [ ] Error handling with safeWrap if needed
