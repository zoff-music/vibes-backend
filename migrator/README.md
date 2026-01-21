# Vibez Migrator

Database migration tool for managing SQLite schema changes.

## Features

- **Forward & Backward Migrations**: Support for both up and down migrations
- **Atomic Operations**: Each migration runs in a transaction
- **Sequential Numbering**: Migrations numbered sequentially (0001, 0002, etc.)
- **Automatic Execution**: Runs automatically in Docker and local-dev

## Usage

### Manual Execution
```bash
cd migrator
export DATABASE_PATH=../data/db/vibes.db
go run main.go
```

### With Makefile
```bash
# Run all up migrations
make migrate-up

# Run down migrations (1 step by default)
make migrate-down

# Run multiple down steps
make migrate-down STEPS=3
```

### Automatic Execution
Migrations run automatically when using:
- `make local-dev` (local development)
- `make dev` (Docker development)
- Docker Compose production deployments

## Migration Files

Located in `migrations/` directory:
- `NNNN_description.up.sql` - Forward migration
- `NNNN_description.down.sql` - Rollback migration

Example: `0001_initial_schema.up.sql`, `0001_initial_schema.down.sql`

## Rules

See [AGENTS.md](./AGENTS.md) for coding conventions.
