# Migrator Coding Rules

Non-negotiable conventions. Follow strictly.

## Critical Rules

- **No `New*` constructors** - prefer struct literals.
- **No inline error assignment** - never `if err := ...; err != nil {}`.
- **All errors wrapped** with context: `fmt.Errorf("error doing X: %w", err)`.
- **Limit Return Values** - NEVER return more than 2 values. 3 or more is strictly illegal.
- **NO `interface{}`** - always use concrete types or specific interfaces.
- **NO Inlined Structs** - never use `struct{}{}` or anonymous structs.
- **SQL file naming**: `NNNN_description.up.sql` / `NNNN_description.down.sql`.

## Command Line Interface

The migrator accepts these flags only:

- `-db`: Postgres connection URL. Defaults to `DATABASE_URL`.
- `-down`: Run down migrations.
- `-steps`: Number of migration steps to run. `0` means all.

The migrator also accepts `up` and `down` subcommands for Docker Compose and k8s job usage.

## File Layout

```
migrator/
└── postgres/
    ├── 0001_current_schema.up.sql
    └── 0001_current_schema.down.sql
```

## Migration Pattern

- Migrations are applied using custom Postgres migration logic in `cmd/migrator/main.go`.
- Always include both up and down migrations.
- Migration discovery checks `./migrator/postgres` from repo root and `./postgres` from inside `migrator`.
- Migration tracking uses the `migrations` table.

## Development Workflow

```bash
# Create new migration files
touch migrator/postgres/0002_new_feature.up.sql
touch migrator/postgres/0002_new_feature.down.sql

# Run all up migrations
DATABASE_URL=postgres://user:password@localhost:5432/vibes?sslmode=disable make migrator

# Run down migrations
go run ./cmd/migrator/main.go -down -steps 1

# Run local integration test
make integrationtest

# Generate tbls database docs
make docs
```
