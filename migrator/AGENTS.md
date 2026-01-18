# Migrator Coding Rules

Non-negotiable conventions. Follow strictly.

## Critical Rules

- **No `New*` constructors** - prefer struct literals.
- **No inline error assignment** - never `if err := ...; err != nil {}`.
- **All errors wrapped** with context: `fmt.Errorf("error doing X: %w", err)`.
- **SQL Files naming**: `NNNN_description.up.sql` / `NNNN_description.down.sql` (e.g. `0001_initial.up.sql`).

## File Layout

```
migrator/
├── main.go                # Entrypoint
└── migrations/            # SQL migration files
```

## Migration Pattern

- Migrations are applied using `golang-migrate` (or custom logic if implemented).
- Always include both UP and DOWN migrations.
- **Atomic operations**: Each migration file should ideally run in a transaction (unless utility prevents it).
