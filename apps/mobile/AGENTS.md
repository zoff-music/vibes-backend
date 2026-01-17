# Frontend AGENTS

This document defines strict conventions for frontend work in this repo. Follow it.

---

## 1) TypeScript rules (strict)

- Do not use the `any` type. Prefer explicit, accurate types and compose them when needed.
- Do not use `@ts-ignore` or `@ts-nocheck`. Fix the type error or adjust types instead.
- Do not use `try {}`/`catch {}`. Use `safeWrap`/`safeWrapAsync` from `@/utils/wrap.ts` for error capture and propagation.

---

## 2) Networking & schema validation (strict)

- Use `wiretyped` instead of raw `fetch` and `EventSource`.
- When using `wiretyped`, always provide full schema validation with `yup`.
- All API calls go through `@/api/client.ts`.

---

## 3) Styling (strict)

- Use `react-native-unistyles` v3 for all styling.
- Use theme tokens from `@/styles/theme.ts` - never hardcode colors, spacing, etc.
- Use `createStyleSheet` and `useStyles` pattern.
- Mobile-first responsive design using breakpoints.

---

## 4) State Management

- Use `zustand` for global state (room, queue, playback).
- Stores live in `@/stores/`.
- Keep stores focused - one store per domain (room, queue, playback).

---

## 5) Component Structure

- UI primitives in `@/components/ui/` (Button, Input, Card, etc.)
- Feature components in `@/components/<feature>/`
- Keep components focused and composable.
- Prefer composition over prop drilling.

---

## 6) File naming

- Use kebab-case for files: `room-header.tsx`, `use-room.ts`
- Use PascalCase for component exports: `RoomHeader`
- Use camelCase for hooks: `useRoom`

---

## 7) Imports

- Use path aliases: `@/` for `src/`, `@vibez/shared` for shared package
- Group imports: external, then shared, then local
- No circular imports

---

## 8) Support & help (general)

- Long-running tooling must use explicit timeouts or non-interactive/batch mode.

---
