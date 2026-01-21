# Migrator Coding Rules

Non-negotiable conventions. Follow strictly.

## Critical Rules

- **No `New*` constructors** - prefer struct literals.
- **No inline error assignment** - never `if err := ...; err != nil {}`.
- **All errors wrapped** with context: `fmt.Errorf("error doing X: %w", err)`.
- **SQL Files naming**: `NNNN_description.up.sql` / `NNNN_description.down.sql` (e.g. `0001_initial.up.sql`).

## Command Line Interface

The migrator accepts these flags only:
- `-db`: Database path (required) - e.g., `-db ../data/db/vibes.db`
- `-down`: Run down migrations (optional boolean flag)
- `-steps`: Number of migration steps to run (optional, 0 = all)

**Invalid flags**: Do not use `-dir` or other unsupported flags.

## File Layout

```
migrator/
├── main.go                # Entrypoint
└── migrations/            # SQL migration files
```

## Migration Pattern

- Migrations are applied using custom logic with SQLite.
- Always include both UP and DOWN migrations.
- **Atomic operations**: Each migration file should ideally run in a transaction (unless utility prevents it).
- **Migration discovery**: The migrator searches for migration files in `./migrations` or `./backend/migrator/migrations` relative to the working directory.
