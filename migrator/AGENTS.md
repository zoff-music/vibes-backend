# Migrator Coding Rules

Non-negotiable conventions. Follow strictly.

## Critical Rules

- **No `New*` constructors** - prefer struct literals.
- **No inline error assignment** - never `if err := ...; err != nil {}`.
- **All errors wrapped** with context: `fmt.Errorf("error doing X: %w", err)`.
- **SQL Files naming**: `NNNN_description.up.sql` / `NNNN_description.down.sql` (e.g. `0001_initial_schema.up.sql`).

## Command Line Interface

The migrator accepts these flags only:
- `-db`: Database path (required) - e.g., `-db ./data/db/vibes.db`
- `-down`: Run down migrations (optional boolean flag)
- `-steps`: Number of migration steps to run (optional, 0 = all)

**Usage Examples:**
```bash
# Run all up migrations
./migrator -db ./data/db/vibes.db

# Run down migrations (1 step)
./migrator -db ./data/db/vibes.db -down -steps 1

# Run specific number of up migrations
./migrator -db ./data/db/vibes.db -steps 3
```

## File Layout

```
migrator/
├── main.go                # Entrypoint with CLI handling
└── migrations/            # SQL migration files
    ├── 0001_initial_schema.up.sql
    ├── 0001_initial_schema.down.sql
    ├── 0002_add_song_votes_and_constraints.up.sql
    ├── 0002_add_song_votes_and_constraints.down.sql
    └── ...
```

## Migration Pattern

- Migrations are applied using custom logic with SQLite.
- Always include both UP and DOWN migrations.
- **Atomic operations**: Each migration file should ideally run in a transaction (unless utility prevents it).
- **Migration discovery**: The migrator searches for migration files in `./migrations` directory.
- **Migration tracking**: Uses `schema_migrations` table to track applied migrations.

## Database Schema Evolution

Current migrations include:

1. **0001_initial_schema**: Core tables (rooms, songs, playback_state, room_users, etc.)
2. **0002_add_song_votes_and_constraints**: Skip voting system
3. **0003_add_room_modes**: Server vs Host mode support
4. **0004_add_participant_status**: Active participant tracking
5. **0005_update_rooms_view**: Room view optimizations
6. **0006_add_external_auth**: OAuth integration tables
7. **0007_add_pending_auth**: OAuth state management
8. **0008_add_tokens_to_external_auth**: Access token storage
9. **0009_add_last_checked_to_access_tokens**: Token refresh tracking
10. **0010_update_room_users_pk**: Primary key updates
11. **0011_consolidate_room_users**: User session consolidation

## Migration Best Practices

- **Backwards Compatibility**: Ensure down migrations can cleanly reverse changes
- **Data Preservation**: Avoid destructive operations without backup strategies
- **Index Management**: Create indexes for performance-critical queries
- **Foreign Key Constraints**: Maintain referential integrity
- **Default Values**: Provide sensible defaults for new columns

## Development Workflow

```bash
# Create new migration files
touch migrations/0012_new_feature.up.sql
touch migrations/0012_new_feature.down.sql

# Test migration up
make migrate-up

# Test migration down
make migrate-down STEPS=1

# Build migrator binary
make build-migrator
```
