# Backend Skill

Use these rules for Go backend work in this repository.

## Workflow

1. Identify whether the change is API routing, app-event processing, a client, domain code, or tests.
2. Read nearby code first and match the existing shape before introducing new patterns.
3. Keep handlers as readable business recipes. Put dependency-specific implementation in clients.
4. Put shared data types and minimal interfaces in the `vibe` domain package.
5. Do not add tracing, OpenTelemetry, metrics middleware, or a monitoring package.
6. Do not use global variables or generics unless explicitly requested.
7. Run focused Go tests and the relevant Makefile target before finishing.
8. Write Go tests and smoketests as table tests. Even single-scenario tests should use a `tests := []struct{...}` table and a `for _, tt := range tests { t.Run(tt.name, ...) }` loop so new cases can be added without changing the test shape.

## Project Layout

- `cmd/server`: backend server entrypoint.
- `cmd/migrator`: migration CLI entrypoint.
- `config`: env and `.env` backed configuration.
- `vibe`: shared domain structs, request/response payloads, and minimal interfaces.
- `server`: server setup, dependency injection, and top-level router.
- `server/internal/handler`: HTTP handlers only.
- `server/internal/event`: app-event wiring and handlers.
- `client/database`: Postgres client split by mirrored feature files.
- `client/frontend`: Go frontend asset client. The actual frontend source is not here; it lives in `client/frontend/render`.
- `client/internalpubsub`: in-process event fanout client.
- `client/youtube`, `client/spotify`, and `client/soundcloud`: external music provider clients.
- `migrator/postgres`: SQL up/down migrations.

## Architecture

- `server.go` owns dependency injection and concrete client construction.
- `server/internal/event/event.go` may wire app-event handlers with concrete clients.
- Other packages should depend on domain interfaces, not concrete clients.
- The only packages allowed to import `client/...` are `server.go`, `server/internal/event/event.go`, and client package tests for their own package.
- Clients must not call other clients.
- Handlers must not declare reusable interfaces; shared interfaces belong in `vibe`.
- Handler request/response structs that cross package or API boundaries belong in `vibe`.
- Routes are defined in `router.go`.
- App events are defined in `server/internal/event/event.go`.

## Go Style

- Always write `err := thing()` and handle `if err != nil` on the following lines.
- Wrap errors with context. Prefer messages shaped like `error [thing] in [function]: %w`.
- Use early returns. Do not add unnecessary `else`.
- Use slices, not fixed-length arrays.
- Avoid name stutter.
- Use minimal capability interfaces such as `Fetcher`, `Creator`, `Updater`, `Deleter`, and `Notifier`; combine names when combining interfaces.
- Do not use broad interface names such as `Manager`, `Service`, `Repository`, or `Coordinator`.
- Return at most two values.
- Return either `error` or `(*StructData, error)` for functions that return actual struct data. On error, return `nil, error`; on success, return `&StructData{}, nil` or `&data, nil`.
- If empty struct data is meaningful, return a struct pointer and give the struct an `IsEmpty` method instead of returning the struct by value.
- Scalar data such as `string`, `bool`, `int`, `int64`, and slices/maps can still be returned normally with `error`.
- Only Go test files may use underscores in filenames. Non-test Go files should use plain domain names such as `song.go`; avoid hyperspecific names like `song_skip.go`, `song_next.go`, or `song_playback.go`.
- Do not use `any` or `interface{}`. Prefer explicit concrete types, even when it is more verbose.
- Never build JSON payloads with string formatting; define structs and use `json.Marshal`.
- Do not directly return another method call. Assign to a local variable first, then return it so error handling and wrapping can stay explicit.

## Database

- Postgres uses `database/sql` with the `pgx` driver and prepared statements.
- All database SQL should use prepared statements.
- Postgres placeholders should use `$1`, `$2`, and so on.
- Prefer one atomic SQL statement over Go-managed transactions.
- App-event worker claim queries that can be processed by multiple backend pods must use concurrency guards such as `FOR UPDATE SKIP LOCKED`.
- Do not add foreign keys unless explicitly requested.
- `client/database/database.go` contains the `Client` struct and `Init`.
- Feature files mirror the handlers/events using them, such as `session.go` and `events.go`.
- Keep database client files grouped by statement flow: prepare statement, method using it, row struct, row scan/rows scan, row map helpers, then next statement.
- Execute prepared statements only with context-aware methods, such as `QueryContext`, `QueryRowContext`, and `ExecContext`.
- Every database execution must use a local timeout context shaped exactly as `cctx, cancel := context.WithTimeout(ctx, 5*time.Second)` followed by `defer cancel()`.

## API

- Handler names and route names should match.
- Use plural nouns for route handlers.
- Handlers accept only the minimal interfaces they need.
- Config values must come from `config.Config` and be passed through client init or handler params.
- Set `Content-Type: application/json` for JSON responses.
- Use the existing error helper for HTTP errors.
