# Backend AGENTS.md

This document defines **non-negotiable conventions** for coding agents. Follow it strictly.

---

## 0) Non-negotiables (read first)

- **Do not change** the `monitoring/` folder or its contents unless there is a glaring, obvious error (there isn’t).
- **Makefiles are king** for initialization and common workflows; prefer `make` targets over ad hoc commands.
- **Docker is how we run this project**; keep scripts and docs aligned with Docker/Compose usage.
- **Avoid generics** unless they are truly necessary. Prefer explicit, readable types.
- **We do not use service/repository patterns.** Do not introduce `service.*` or `repository.*` layers/structs as an architecture pattern.
- **Avoid `New*` constructors** for structs unless strictly necessary (e.g., high reuse of default values, required invariants, or expensive setup that must be centralized). Prefer struct literals.
- **All HTTP requests happen only inside clients** (packages under `clients/`).
- **Nothing imports from `clients/`** except:
  - `client.ErrorCodeWrapper` usage (explicitly allowed)
  - `server/server.go` startup initialization of clients
- **Tracing** must be started in any method that takes a `context.Context` **except HTTP handlers** (HTTP handlers get tracing from middleware).
- **Never return `nil, nil`** from `(*T, error)` functions. Illegal.
- Errors must be **verbosely wrapped**; do not hot-potato errors.
- **Never inline errors in conditionals**. Do not use `if err := ...; err != nil {}`.

---

## 1) Repo map & placement rules

### 1.1 Clients live in `clients/`
- All external integrations live under `clients/`.
- Client packages must be named after the integration:
  - PubSub → `clients/pubsub`
  - Database → `clients/database`
  - Slack → `clients/slack`
- Client entrypoint filename must match the package name:
  - `clients/slack/slack.go`
  - `clients/pubsub/pubsub.go`
  - `clients/database/database.go`

### 1.2 Nothing imports `clients/*` (with two exceptions)
- **Forbidden:** importing `clients/*` from anywhere else in the codebase.
- **Allowed exceptions only:**
  1) Using `client.ErrorCodeWrapper` where needed
  2) `server/server.go` on startup to initialize clients

### 1.3 Domain-ish area packages
For an API area like `users`, the following belongs in a package folder at repo root (or appropriate module root):

- `users/users.go` holds:
  - Interfaces used by handlers (e.g., `DataFetcher`, `Notifier`)
  - DTO-ish structs used by handlers/responses (e.g., `Data`, `FetchedEvent`)
  - Helper methods like `IsEmpty()`

**These types should not live in handler packages.**

### 1.4 API handlers live under `server/internal/handler`
- HTTP handlers for the `users` area live in:
  - `server/internal/handler/users.go`

### 1.5 File naming should mirror usage
- Types/structs used by a specific handler live in the `.go` file named the same as the handler file **unless** they are part of an area package (see 1.3).
- Methods used in clients (consumed via interfaces in handlers) should live in `.go` files named the same as the handler that uses them.
- If something is used by multiple handlers, pick a clear common denominator filename.

### 1.6 Versioned endpoints: file suffix
- For `/api/vN` endpoints, files should end with `vN`.
  - Example: `/api/v2` “users” handler code → `usersv2.go`
  - Matching area package types used only for v2 should follow similarly (e.g., `usersv2.go`) if they are version-specific.

---

## 2) API handler shape (strict)

API handlers must be **functions that take interfaces and return `http.HandlerFunc`**.

Pattern:

```go
func GetUserData(
	u users.DataFetcher,
	ps users.Notifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// ...
	}
}
```

Rules:
- Dependencies are injected as interfaces via function parameters.
- Use `ctx := r.Context()` inside the returned handler.
- **Do not create tracing spans inside HTTP handlers.** Tracing is handled via middleware.
- Still follow error wrapping and no-inline-err rules inside the handler.

### 2.1 PubSub notify from HTTP handlers must not be cancelled by request cancellation
When notifying from HTTP handlers, use:

```go
err = ps.NotifyUserDataFetched(context.WithoutCancel(ctx), evt)
```

Use this pattern when you want the notify to proceed even if the request is cancelled.

---

## 3) Clients: standard structure & initialization

### 3.1 Required client entrypoint
Each client entrypoint file (matching package name) must expose:

```go
func (c *Client) Init(ctx context.Context, cfg *config.Config) error
```

- `Init` initializes the client using `cfg` (env-driven config).
- The entrypoint must be in:
  - `clients/<name>/<name>.go` (e.g., `clients/slack/slack.go`)

### 3.2 Server owns clients; store pointers on `Server`
- `type Server struct` must hold **all clients as pointers**.

Initialize clients like:

```go
var dbClient database.Client
err := dbClient.Init(ctx, config)
if err != nil {
	return fmt.Errorf("database client: %w", err)
}

s.DB = &dbClient
```

- Clients are **consumed via interfaces**, not concrete client types.

---

## 4) HTTP: only inside clients

- **All HTTP requests must be performed from within a client package** under `clients/`.
- A client must initialize the HTTP client using the shared HTTP helper in `client/http.go`:

```go
c.HTTPClient = client.NewHTTPClient(client.Parameters{Timeout: &timeout})
```

- The timeout is set in the specific client (not globally).
- No handler/service code should do HTTP calls directly.

---

## 5) Database client conventions

If a database client exists (e.g., `clients/database`):

### 5.1 Prepared statements: method + statement + runner naming (strict)

- Statements must be prepared in **`prepareSTATEMENTNAMEStmt() error`** methods.
- The underlying prepared statement field must reflect the **same name** and be suffixed with `Statement`.
- The method that runs/uses the prepared statement must be named based on the same stem.

Example:
- Prepare method: `prepareGetUserDataStmt()`
- Prepared statement field: `GetUserDataStatement`
- Runner method: `GetUserData(ctx context.Context, ...) (*SomeType, error)`

> Rule: 1:1 mapping:
> - `prepareXStmt()` prepares `XStatement`
> - `X(...)` executes `XStatement`

```go
// prepareGetUserDataStmt prepares the GetUserDataStatement.
func (c *Client) prepareGetUserDataStmt() error {
	stmt, err := c.DB.Preparex(`SELECT id FROM user_data WHERE id = $1`)
	if err != nil {
		return fmt.Errorf("error preparing GetUserDataStatement: %w", err)
	}

	c.GetUserDataStatement = stmt

	return nil
}
```

### 5.2 File-level structure inside DB client files
Within a DB client file, order sections strictly:

1) Prepare statement method  
2) Method that uses the statement  
3) Structs for returned rows  
4) Internal scan methods (direct row scans)  
5) Mapping method to domain type  

When using `QueryRowContext`, always use a dedicated row struct with a `scan(*sql.Row) error` method and map from that struct to the domain type.

Pattern:

```go
row := c.GetSessionStatement.QueryRowContext(cctx, token)

var scanned sessionRow

err := scanned.scan(row)
if err != nil {
	if err != sql.ErrNoRows {
		log.Printf("auth: fetch session: %v", err)
	}

	return sessionRow{}, false
}
```

Example scan + map:

```go
func (e *exampleDataRow) scan(rows *sqlx.Rows) error {
	return rows.Scan(
		&e.IsFake,
		&e.Date,
	)
}

func (e *exampleDataRow) toExampleData() example.Data {
	return example.Data{
		IsFake: e.IsFake.Bool,
		Date:   e.Date.Time,
	}
}
```

### 5.3 `sql.ErrNoRows` handling depends on context

#### APIs: “no rows” is expected
- Return an **empty struct pointer + nil**.
- Provide an exported `IsEmpty() bool` on the returned struct.

```go
if err != nil {
	if errors.Is(err, sql.ErrNoRows) {
		return &DataStruct{}, nil
	}

	return nil, fmt.Errorf("error fetching data: %w", err)
}
```

#### Subscribers/Processors
- For “no rows” if expected:
  - Use `ErrExpected` + `ErrNonRecoverable` if you do **not** want to retry
  - Use `ErrExpected` if you **do** want to retry
(See section 7.4.)

### 5.4 Database null types
When representing nullable database columns in row structs, always use `database/sql` null types (`sql.NullString`, `sql.NullInt64`, `sql.NullBool`, etc). Do not use proprietary or custom null wrapper types.

---

## 6) Tracing & context timeouts

### 6.1 Tracing is mandatory for context-taking methods (non-HTTP)
If a method takes `ctx context.Context`, it must begin with:

```go
span, ctx := opentracing.StartSpanFromContext(ctx, "METHODNAMEGOESHERE")
defer span.Finish()
```

- Span name must be **1:1 with the method name**.
- **Do not do this inside HTTP handlers** (middleware owns tracing).

### 6.2 Context timeouts
When applying a timeout:

```go
cctx, cancel := context.WithTimeout(ctx, INT*time.Second)
defer cancel()
```

- Always `defer cancel()` for readability.
- Use `cctx` for downstream calls.

---

## 7) Error handling rules (strict)

### 7.1 No inlined errors in conditionals (strict)
Never do:

```go
if err := method(); err != nil {
	// ...
}
```

Always do:

```go
err := method()
if err != nil {
	// ...
}
```

### 7.2 Always wrap errors verbosely
Even if calling a function with the same return signature, you still wrap:

```go
err := callOtherFunction(...)
if err != nil {
	return fmt.Errorf("error EXPLAIN ERROR: %w", err)
}
```

For tuple returns:

```go
data, err := callOtherFunction(...)
if err != nil {
	return nil, fmt.Errorf("error EXPLAIN ERROR: %w", err)
}

return &data, nil
```

All error messages must start with `error ` and must always be rewrapped (no bare returns).

### 7.3 `(*T, error)` rules
- Prefer `(*T, error)` return signatures.
- On error, return `nil, wrappedErr`.
- On success, return `&data, nil`.
- **Never** return `nil, nil`.

### 7.4 Subscriber/Processor error rewrap policy
Use these wrappers:

#### Unrecoverable (no retry possible)
```go
return internalerror.ErrNonRecoverable{
	Err: fmt.Errorf("error EXPLAIN ERROR: %w", err),
}
```

#### Expected + retry (no logging)
```go
return internalerror.ErrExpected{
	Err: fmt.Errorf("error EXPLAIN ERROR: %w", err),
}
```

#### Expected + no retry + no logging
```go
return internalerror.ErrExpected{
	Err: internalerror.ErrNonRecoverable{
		Err: fmt.Errorf("error EXPLAIN ERROR: %w", err),
	},
}
```

#### Unexpected but attempt recovery
```go
return fmt.Errorf("error EXPLAIN ERROR: %w", err)
```

---

## 8) Interfaces, dependency injection, naming

### 8.1 Interface naming rules
- Interfaces are named by the verb/action:
  - `Fetcher`, `Getter`, `Updater`, etc. with optional prefixes.
- PubSub-related interfaces must end with **`Notifier`**.
- PubSub methods must be prefixed with **`Notify`** and then what they notify.

### 8.2 PubSub payloads must include send date
Anything notified over PubSub must include a **Date it was sent**, to prevent overwrites due to out-of-order delivery.

### 8.3 Avoid name stuttering
- Packages and structs must avoid stuttering:
  - If package is `users`, do not export `UserData`, `UserClient`, etc.
  - Avoid `users.UserData`, `users.User...` patterns.

---

## 9) API conventions (routing, middleware, handlers)

### 9.1 Endpoints are plural
- Endpoints must be pluralized.
- If pluralization sounds weird, choose a different name.

### 9.2 Proper HTTP verbs
- Preserve correct REST verb usage: `POST`, `GET`, `PUT`, `PATCH`, `DELETE`.
- `DELETE` must never contain a request body.
- `DELETE` should point directly to the instance in the URL.

### 9.3 Versioned subrouters
- Routes must be versioned with explanatory subrouters:

```go
const v1API string = "/api/v1"

apiV1 := s.Router.PathPrefix(v1API).Subrouter()
```

- Repeat the same pattern for `v2`, `v3`, etc.

### 9.4 Middlewares
- Middleware implementations live in: `server/handler/middleware`
- Middleware wiring happens through `Server` methods in server/internal code:

```go
func (s *Server) addTracingAndMetrics(routers ...*mux.Router) {
	mw := middleware.NewTracingAndMetrics(/* dependencies */)

	for _, r := range routers {
		r.Use(mw.Handle)
	}
}
```

### 9.5 Handler error writing
- Errors in handlers must **always** use `handleError`.
- This includes streaming and long-lived connections (SSE/streamed responses): still `handleError`.
- Do not write error handling on one line; keep it readable and always pass wrapped errors.

---

## 10) Subscriber handlers

### 10.1 Handler struct placement and naming
For each subscriber handler, define the dependency struct **immediately above** the handler:

```go
// Example is an example event.
type Example struct {
	DB example.DataFetcher
}

// Handle is the handler for the example event.
func (e Example) Handle(ctx context.Context, data []byte) error {
	var exampleData example.Data

	err := json.Unmarshal(data, &exampleData)
	if err != nil {
		return internalerror.ErrNonRecoverable{
			Err: fmt.Errorf("failed to unmarshal example data: %w", err),
		}
	}

	// Do stuff here

	return nil
}
```

Abbreviations for dependencies should be representative:
- `DB` (database), `PS` (pubsub), `CS` (cloud storage), `BT` (bigtable), `AI` (LLMs), etc.

---

## 11) Regex rules

- Regex must **never** be compiled/created inline at runtime.
- Regex must be part of config startup/initialization so failures occur on startup, not runtime.

---

## 12) Logging

- Use Go’s standard library logger: `log`
- Do not introduce alternative logging frameworks.

---

## 13) Formatting & readability rules

- Add newlines after `}` in conditionals for readability.
- Add newlines after `})` for readability.
- Prefer explicit, readable code over cleverness.
- Avoid generics unless clearly justified and materially beneficial.

---

## 15) Support & help (general)

- Long-running tooling must use explicit timeouts or non-interactive/batch mode.

---
