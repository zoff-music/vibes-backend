# Add Database Method

Create a new database method with prepared statement.

## Critical Rules

- **No transactions** - each query must be atomic (single statement)
- **No multi-statement operations** - split into separate methods if needed
- **Domain types in `vibe/` directory** - never in handlers or database files

## Requirements

Provide:
- Method name (e.g., "GetRoom", "CreateSong")
- SQL query (single atomic statement)
- Parameters and return type
- Which resource file (rooms.go, songs.go, playback.go, etc.)

## Pattern

1:1 naming convention:
- `prepareXStmt()` → prepares `XStatement`
- `X()` → executes `XStatement`

## Template

File: `backend/client/database/{resource}.go`

```go
// prepareGetXStmt prepares the GetXStatement.
func (c *Client) prepareGetXStmt() error {
    stmt, err := c.DB.Prepare(`
        SELECT column1, column2
        FROM table
        WHERE id = ?
    `)
    if err != nil {
        return fmt.Errorf("error preparing GetXStatement: %w", err)
    }

    c.GetXStatement = stmt

    return nil
}

// GetX retrieves X by id.
func (c *Client) GetX(ctx context.Context, id string, userID string) (*vibe.X, error) {
    span, ctx := opentracing.StartSpanFromContext(ctx, "GetX")
    defer span.Finish()

    cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    row := c.GetXStatement.QueryRowContext(cctx, id)

    var scanned xRow

    err := scanned.scan(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return &vibe.X{}, nil
        }

        return nil, fmt.Errorf("error scanning X row: %w", err)
    }

    result := scanned.toX()

    return &result, nil
}

type xRow struct {
    Column1 sql.NullString
    Column2 sql.NullInt64
}

func (r *xRow) scan(row *sql.Row) error {
    return row.Scan(&r.Column1, &r.Column2)
}

func (r *xRow) toX() vibe.X {
    return vibe.X{
        Column1: r.Column1.String,
        Column2: int(r.Column2.Int64),
    }
}
```

## Database Files

- `database.go` - Client struct, Init, prepared statement fields
- `rooms.go` - Room operations (GetRoom, CreateRoom, UpdateRoomSettings)
- `songs.go` - Queue operations (GetSongs, AddSong, RemoveSong, ReorderSongs)
- `playback.go` - Playback state operations (GetPlaybackState, UpdatePlayback)
- `participants.go` - User session operations (UpdateParticipant, GetActiveParticipants)
- `skip.go` - Skip voting operations (SkipSong, AddSkipVote, HasUserVoted)
- `users.go` - User management operations
- `authorization.go` - OAuth token operations

## Checklist

- [ ] Statement field added to Client struct in database.go
- [ ] prepare method called in Init()
- [ ] Tracing span matches method name
- [ ] Context timeout set (5 seconds)
- [ ] sql.ErrNoRows returns empty struct + nil
- [ ] Domain type in `vibe/{resource}.go`
- [ ] Interface in `vibe/{resource}.go` (if needed)
- [ ] Include userID parameter for user-specific operations
- [ ] Error messages start with "error"
