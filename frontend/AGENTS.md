# Frontend AGENTS

This document defines strict conventions for frontend work in this repo. Follow it.

---

## 1) TypeScript rules (strict)

- Do not use the `any` type. Prefer explicit, accurate types and compose them when needed.
- Do not use `@ts-ignore` or `@ts-nocheck`. Fix the type error or adjust types instead.
- Do not use `try {}`/`catch {}`. Use `safeWrap`/`safeWrapAsync` from `frontend/src/utils/wrap.ts` for error capture and propagation.

---

## 2) Networking & schema validation (strict)

- Use `wiretyped` instead of raw `fetch` and `EventSource`.
- When using `wiretyped`, always provide full schema validation with `yup`.

---

## 3) Support & help (general)

- Long-running tooling must use explicit timeouts or non-interactive/batch mode.

---
