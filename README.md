# Vibez

**A collaborative music queue with synchronized playback.**

**Start here:** read `CLAUDE.md` for project overview and architecture. Read `backend/AGENTS.md` for backend coding conventions.

---

## What this repo is for

- A consistent way to write Go across Embroidery services
- A reference for how to structure code (files, packages, routing, handlers)
- A baseline for new repos and refactors (without introducing new architecture layers)

---

## Core principles

- **Readability is king.**
- **No service/repository patterns.** Don’t introduce “service” or “repository” layers.
- **Avoid generics** unless truly necessary.
- **Prefer struct literals** over `New*` constructors unless there’s strong reuse or invariants.
- **IO belongs in clients** (`clients/`), consumed via interfaces.
- **API handlers are thin**: request parsing, calling interfaces, response writing.
- **Tracing for API handlers is done by middleware** (not inside each handler function).
- **Errors are never hot-potato’d**: always wrap with descriptive context.
- **Never use inline error assignments** in conditionals (`if err := ...`).

---

## Suggested layout

A typical service will look like:

- `users/users.go` — domain-ish types + interfaces for the users area (used by handlers)
- `server/internal/handler/users.go` — HTTP handler functions for `/users` endpoints
- `server/handler/middleware/` — middleware implementations (including tracing)
- `clients/` — external integrations (db, pubsub, slack, etc.)
- `server/server.go` — initializes clients and wires routes/middleware

---

## Example: users package types live in `users/users.go`

This file holds the interfaces + types used by the handler and injected from server wiring.

- Interfaces use action/verb naming (`DataFetcher`, `Notifier`, etc.)
- PubSub notifier method begins with `Notify...`
- PubSub events must include `Date`

```go
package users

import (
	"context"
	"time"
)

type DataFetcher interface {
	GetUserData(ctx context.Context, id string) (*Data, error)
}

type Notifier interface {
	NotifyUserDataFetched(ctx context.Context, evt FetchedEvent) error
}

type Data struct {
	ID   string    `json:"id"`
	Date time.Time `json:"date"`
}

func (d Data) IsEmpty() bool {
	return d.ID == ""
}

type FetchedEvent struct {
	Date time.Time `json:"date"`
	ID   string    `json:"id"`
}
```

> Note: Avoid name stuttering. Package is `users`, so types should not start with `User...`.

---

## Example: API handler lives in `server/internal/handler/users.go`

**Rules demonstrated**
- Handler is a function returning `http.HandlerFunc`
- Dependencies injected as interfaces from `users/users.go`
- Tracing is **not** created in the handler; it is handled by middleware
- Uses `handleError` (including for “streaming” scenarios)
- No `if err := ...` inline error assignments
- Errors are wrapped verbosely
- PubSub notify uses `context.WithoutCancel(ctx)` to decouple publish from request cancellation

```go
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/embroidery-io/YOUR-APP-NAME/users"
)

func GetUserData(
	u users.DataFetcher,
	ps users.Notifier,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id := r.PathValue("id")
		if id == "" {
			handleError(
				w,
				http.StatusBadRequest,
				fmt.Errorf("error missing id path param"),
				true,
			)

			return
		}

		data, err := u.GetUserData(ctx, id)
		if err != nil {
			handleError(
				w,
				http.StatusInternalServerError,
				fmt.Errorf("error getting user data: %w", err),
				true,
			)

			return
		}

		if data.IsEmpty() {
			handleError(
				w,
				http.StatusNotFound,
				fmt.Errorf("error missing user data: %w", err),
				true,
			)
			return
		}

		event := users.FetchedEvent{
			Date: time.Now().UTC(),
			ID:   data.ID,
		}

		err = ps.NotifyUserDataFetched(context.WithoutCancel(ctx), event)
		if err != nil {
			handleError(
				w,
				http.StatusInternalServerError,
				fmt.Errorf("error notifying user data fetched: %w", err),
				true,
			)

			return
		}

		response, err := json.Marshal(data)
		if err != nil {
			handleError(
				w,
				http.StatusInternalServerError,
				fmt.Errorf("error marshaling response: %w", err),
				true,
			)

			return
		}

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(response)
	}
}
```

> Tracing spans for API requests are created by middleware, not by handler functions.

---

## Example: subscriber/processor handler style

Subscriber/processors are structured differently from HTTP handlers:
- You define a small struct containing dependencies right above the handler method
- Handler method signature: `Handle(ctx context.Context, data []byte) error`
- Uses the `internalerror` wrapping patterns (see `AGENTS.md`)
- Still: no inline error in conditionals, and errors must be descriptive

```go
package processor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/embroidery-io/YOUR-APP-NAME/internalerror"
	"github.com/embroidery-io/YOUR-APP-NAME/example"
)

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

---

## Where to look next

- `AGENTS.md` — mandatory rules for clients, errors, prepared statements, PubSub naming, routing, middleware wiring, formatting.
- Your service’s `server/server.go` — client initialization and route wiring should follow `AGENTS.md`.

---