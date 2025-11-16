# AGENTS.md - AI Agent Context

> **⚠️ IMPORTANT FOR AI AGENTS**
>
> When making changes to this codebase, you MUST update the relevant documentation:
>
> - **AGENTS.md** - Update architecture, design decisions, or code patterns if they change
> - **README.md** - Update usage examples, CLI flags, or features if they change
> - **Makefile comments** - Update if build process changes
> - **Code comments** - Keep inline documentation accurate
>
> Documentation drift leads to confusion. Always keep docs in sync with code changes.

## Project Overview

**rdb2sqlite** is a Go CLI tool that parses Redis RDB (dump) files and imports key metadata into a SQLite database for analysis.

## Purpose

- **Input**: Redis RDB dump file (`dump.rdb`)
- **Output**: SQLite database with key-level metadata
- **Use Case**: Memory analysis, key pattern analysis, optimization

## Architecture

### Core Components

1. **cmd/rdb2sqlite/main.go** - CLI entry point, flag parsing, version info, orchestration
2. **pkg/parser/parser.go** - RDB parsing using `github.com/hdt3213/rdb`
3. **pkg/storage/storage.go** - SQLite operations, batched inserts, schema management
4. **pkg/types/types.go** - Shared data structures (KeyMetadata, Storage)

### Project Structure

```text
rdb2sqlite/
├── cmd/rdb2sqlite/      # Main application binary
│   └── main.go         # Entry point, CLI, and version info
├── pkg/                # Reusable packages
│   ├── parser/         # RDB parsing logic
│   ├── storage/        # SQLite operations
│   └── types/          # Shared data structures
├── AGENTS.md           # This file (AI agent documentation)
├── LICENSE             # MIT License
├── Makefile            # Build automation
├── README.md           # User documentation
└── go.mod              # Go module definition
```

### Key Design Decisions

1. **Streaming Processing**: Never loads entire RDB into memory
2. **Batched Inserts**: Configurable batch size (default: 10,000) for performance
3. **Deferred Indexing**: Creates indexes AFTER data insertion for speed
4. **Progress Tracking**: Uses `github.com/schollz/progressbar/v3` for live updates

### Dependencies

- `github.com/hdt3213/rdb` - RDB parsing and metadata extraction
- `github.com/mattn/go-sqlite3` - SQLite driver (requires CGO)
- `github.com/schollz/progressbar/v3` - Progress bar display

## Database Schema

```sql
CREATE TABLE db_stats (
    database        INTEGER,   -- Redis DB index
    key             TEXT,      -- Key name
    type            TEXT,      -- Data type (string, list, hash, set, zset, stream)
    size            INTEGER,   -- Size in bytes
    size_readable   TEXT,      -- Human-readable size (e.g., "1.5MB")
    element_count   INTEGER,   -- Number of elements in collection
    encoding        TEXT,      -- Redis internal encoding
    expiration      TEXT       -- Timestamp when key expires
);

-- Indexes for query performance
CREATE INDEX idx_db_stats_size ON db_stats(size);
CREATE INDEX idx_db_stats_key ON db_stats(key);
CREATE INDEX idx_db_stats_expiration ON db_stats(expiration);
CREATE INDEX idx_db_stats_type ON db_stats(type);
CREATE INDEX idx_db_stats_expiration_size ON db_stats(expiration, size);
```

## CLI Flags

- `--rdb` (required) - Path to RDB file
- `--db` (required) - Path to SQLite database file
- `--batch-size` (default: 10000) - Batch size for inserts
- `--skip-errors` (default: false) - Continue on errors instead of stopping

## Implementation Details

### Error Handling Modes

1. **Default (Strict)**: Stops immediately on any error
2. **Skip Errors**: Logs errors, increments skip counter, continues parsing

### Performance Optimizations

1. **SQLite Pragmas**:
   - `journal_mode=WAL` - Write-Ahead Logging
   - `synchronous=NORMAL` - Balanced durability/speed

2. **Batching Strategy**:
   - Accumulates rows in memory
   - Commits in single transaction per batch
   - Reduces I/O overhead

3. **Index Timing**:
   - Table created at start (no indexes)
   - Data inserted without index overhead
   - Indexes created once at end

### Progress Bar Integration

- Updates on each batch commit
- Clears before logging errors (avoids UI conflicts)
- Finishes when parsing completes (in `pkg/parser/parser.go`)

## Data Flow

```text
RDB File
   ↓
pkg/parser/parser.go (streaming, using github.com/hdt3213/rdb)
   ↓
pkg/storage/storage.go (batching)
   ↓
SQLite (transactions with WAL mode)
   ↓
Progress Bar (live updates via progressbar/v3)
```

## Common Queries

See `README.md` for analysis query examples:

- Top N largest keys
- Top N largest keys without TTL
- Top N keys with longest TTL
- Keys by prefix pattern
- Keys grouped by prefix
- Memory distribution by type

## Known Limitations

1. **CGO Required**: SQLite driver needs C compiler
2. **Empty DB Only**: Fails if target DB already has data
3. **Size Estimates**: Some sizes are estimates, not exact Redis memory

## Build & Run

```bash
# Build with Makefile (recommended)
make build

# Or build directly with Go
go build -o rdb2sqlite ./cmd/rdb2sqlite

# Run
./rdb2sqlite --rdb dump.rdb --db analysis.db

# With error resilience
./rdb2sqlite --rdb dump.rdb --db analysis.db --skip-errors

# Show version
./rdb2sqlite --version
```

## Testing Considerations

- Test with various RDB versions
- Test with large files (memory constraints)
- Test error scenarios (corrupted RDB)
- Verify size calculations accuracy
- Check performance with different batch sizes

## Future Enhancements (Potential)

- Streaming to other databases (Postgres, MySQL)
- Built-in query commands (no SQL needed)
- Real-time RDB monitoring
- Compression support
- Parallel batch inserts
- Resume/checkpoint capability

## Code Patterns

### Adding New Metadata Fields

1. Update `KeyMetadata` in `pkg/types/types.go`
2. Update schema in `pkg/storage/storage.go` (`initSchema()`)
3. Extract field in `pkg/parser/parser.go` (from `model.RedisObject`)
4. Update insert statement in `pkg/storage/storage.go` (`Flush()`)
5. **MUST UPDATE**: `AGENTS.md` database schema section with the new field
6. **OPTIONAL**: Update `README.md` if the field is user-facing or changes queries

### Adding CLI Flags

1. Define flag in `cmd/rdb2sqlite/main.go` (`flag.String/Int/Bool`)
2. Pass to relevant component (Parser, Storage, etc.)
3. **MUST UPDATE**: `README.md` usage section with the new flag and example
4. Test with `./rdb2sqlite --help` to verify flag appears

## Important Notes for AI Agents

- **Never remove error handling** - strict mode is default for data integrity
- **Preserve streaming architecture** - memory efficiency is critical
- **Maintain progress bar compatibility** - use `LogError()` for logging during parsing
- **Keep indexes deferred** - essential for performance on large datasets
- **Version compatibility** - test with the RDB library version we depend on

## Documentation Maintenance Rules

When you make changes, update documentation immediately. Here's what to update:

### Changes Requiring AGENTS.md Updates

- Adding/removing core components or packages
- Changing architecture or design decisions
- Modifying the data flow
- Changing performance optimization strategies
- Adding new code patterns
- Updating dependencies

**Example**: If you add a new package `pkg/validator/`, update the "Core Components" section.

### Changes Requiring README.md Updates

- Adding/removing CLI flags
- Changing default values
- Adding new features
- Modifying usage examples
- Changing build/install instructions

**Example**: If you add `--max-keys` flag, add it to the "Usage" section with an example.

### Changes Requiring Both

- Major architectural refactoring
- New workflows or processes
- Changes to the project structure
- New make targets

**Example**: If you add `make docker-build`, update both Makefile comments and README.md "Developing" section.

### Pre-Completion Checklist

Before marking any task as complete, verify:

- [ ] All modified code has accurate inline comments
- [ ] CLI flag changes reflected in README.md usage section
- [ ] Architectural changes reflected in AGENTS.md
- [ ] New make targets documented in Makefile and README.md
- [ ] Database schema changes reflected in AGENTS.md schema section
- [ ] New dependencies listed in AGENTS.md dependencies section
