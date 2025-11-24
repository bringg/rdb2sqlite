# rdb2sqlite

A fast, memory-efficient Go tool that extracts Redis RDB dump metadata into SQLite for analysis.

## About

**rdb2sqlite** parses Redis RDB files and imports key-level metadata (size, type, encoding, expiration) into a SQLite database. This enables efficient memory analysis, key pattern discovery, and optimization without loading entire dumps into memory.

**Key Features:**

- Streaming RDB parser (memory-bounded processing)
- Batched SQLite inserts (configurable batch size)
- Progress bar with live statistics
- Deferred index creation for optimal performance
- Error resilience mode (--skip-errors)

## Usage

### Installation

```bash
make build
# or
go install rdb2sqlite/cmd/rdb2sqlite@latest
```

### Basic Usage

```bash
# Parse RDB and import to SQLite
./rdb2sqlite --rdb /path/to/dump.rdb --db /path/to/analysis.db

# With custom batch size
./rdb2sqlite --rdb dump.rdb --db analysis.db --batch-size 10000

# Skip errors and continue parsing
./rdb2sqlite --rdb dump.rdb --db analysis.db --skip-errors

# Show version
./rdb2sqlite --version
```

### Example

```bash
$ ./rdb2sqlite --rdb /var/lib/redis/dump.rdb --db /tmp/redis-analysis.db
2025/11/13 22:25:38 Initializing SQLite database: /tmp/redis-analysis.db
2025/11/13 22:25:38 Parsing RDB file: /var/lib/redis/dump.rdb
Importing keys (32810000/-, 124015 keys/s) [4m50s]
2025/11/13 22:29:51 Creating indexes on db_stats table...
2025/11/13 22:29:55 Indexes created successfully
2025/11/13 22:29:55 Done! Imported 32810000 rows into /tmp/redis-analysis.db.db_stats
```

### Querying the Database

Once imported, analyze with SQL:

```bash
sqlite3 /tmp/redis-analysis.db
```

```sql
-- Top 100 largest keys
SELECT key, size_readable FROM db_stats ORDER BY size DESC LIMIT 100;

-- Top 100 largest keys without TTL (never expire)
SELECT key, type, size_readable
FROM db_stats
WHERE expiration IS NULL
ORDER BY size DESC
LIMIT 100;

-- Total size of all non-expiring keys
SELECT
    COUNT(*) AS key_count,
    SUM(size) AS total_bytes,
    SUM(size) / 1024.0 / 1024.0 AS total_mb
FROM db_stats
WHERE expiration IS NULL;

-- Top 100 keys with longest expiration (furthest in the future)
-- expiration is stored as ISO-8601 timestamp
SELECT key, type, size_readable, expiration,
       (julianday(expiration) - julianday('now')) * 86400 as seconds_until_expiry
FROM db_stats
WHERE expiration IS NOT NULL
ORDER BY expiration DESC
LIMIT 100;

-- Keys by prefix pattern
SELECT
    substr(key, 1, instr(key, ':') - 1) AS prefix,
    COUNT(*) AS count,
    SUM(size) AS total_size
FROM db_stats
GROUP BY prefix
ORDER BY total_size DESC;

-- Total size of all keys starting with a specific prefix
SELECT
    COUNT(*) AS key_count,
    SUM(size) AS total_bytes,
    SUM(size) / 1024.0 / 1024.0 AS total_mb
FROM db_stats
WHERE key LIKE 'authentication:%';

-- Memory by data type
SELECT type, COUNT(*), SUM(size) AS total_bytes
FROM db_stats
GROUP BY type;
```

## Developing

### Building

```bash
make build
```

### Running Tests

```bash
make test
```

### Make Targets

```bash
make help
```

### Adding Features

See `AGENTS.md` for detailed development guidelines including:

- Architecture overview
- Adding new CLI flags
- Adding metadata fields
- Performance considerations

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run `make fmt && make vet`
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
