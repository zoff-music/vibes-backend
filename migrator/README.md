# Vibez Migrator

Database migration tool for managing Postgres schema changes.

## Usage

### Manual Execution

```bash
export DATABASE_URL=postgres://user:password@localhost:5432/vibes?sslmode=disable
go run ./cmd/migrator/main.go
```

### With Makefile

```bash
make migrator
```

### Integration Test

```bash
make integrationtest
```

This starts a local Postgres container, runs all up migrations, then runs down migrations.

### Database Docs

```bash
make docs
```

This starts local Postgres, runs migrations, and generates tbls documentation in `docs/db/`.

## Migration Files

Postgres migrations live in `migrator/postgres/`:

- `NNNN_description.up.sql` - forward migration
- `NNNN_description.down.sql` - rollback migration

The migrator also supports `./postgres` when run from inside the `migrator` directory.

## Rules

See [AGENTS.md](./AGENTS.md) for coding conventions.
