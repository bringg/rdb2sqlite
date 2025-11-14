package storage

import (
	"database/sql"
	"fmt"
	"log"

	"rdb2sqlite/pkg/types"

	_ "github.com/mattn/go-sqlite3"
	"github.com/schollz/progressbar/v3"
)

// New creates a new Storage instance and initializes the database
func New(dbPath string, batchSize int) (*types.Storage, error) {
	// Add connection parameters to ensure read-write-create mode
	// This prevents "readonly database" errors
	connStr := dbPath + "?cache=shared&mode=rwc"
	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create a progress bar (max will be updated as we go)
	bar := progressbar.NewOptions(-1,
		progressbar.OptionSetDescription("Importing keys"),
		progressbar.OptionSetWidth(15),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetItsString("keys"),
		progressbar.OptionThrottle(100),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionClearOnFinish(),
	)

	storage := &types.Storage{
		DB:          db,
		BatchSize:   batchSize,
		Batch:       make([]types.KeyMetadata, 0, batchSize),
		TotalRows:   0,
		BatchNumber: 0,
		SkippedKeys: 0,
		ProgressBar: bar,
	}

	// Set SQLite pragmas for better performance
	if err := setPragmas(storage); err != nil {
		db.Close()
		return nil, err
	}

	// Initialize schema
	if err := initSchema(storage); err != nil {
		db.Close()
		return nil, err
	}

	// Check if table already has data (requirement 1c)
	if err := checkExistingData(storage); err != nil {
		db.Close()
		return nil, err
	}

	return storage, nil
}

// setPragmas sets SQLite pragmas for performance
func setPragmas(s *types.Storage) error {
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
	}

	for _, pragma := range pragmas {
		if _, err := s.DB.Exec(pragma); err != nil {
			return fmt.Errorf("failed to set pragma: %w", err)
		}
	}

	return nil
}

// initSchema creates the table (without indexes)
func initSchema(s *types.Storage) error {
	// Create table only - indexes will be created after data insertion
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS db_stats (
		database        INTEGER,
		key             TEXT,
		type            TEXT,
		size            INTEGER,
		size_readable   TEXT,
		element_count   INTEGER,
		encoding        TEXT,
		expiration      TEXT
	);`

	if _, err := s.DB.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

// CreateIndexes creates indexes after all data has been inserted
// This is much faster than creating indexes before inserting data
func CreateIndexes(s *types.Storage) error {
	log.Println("Creating indexes on db_stats table...")

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_db_stats_size ON db_stats(size)",
		"CREATE INDEX IF NOT EXISTS idx_db_stats_key ON db_stats(key)",
		"CREATE INDEX IF NOT EXISTS idx_db_stats_expiration ON db_stats(expiration)",
		"CREATE INDEX IF NOT EXISTS idx_db_stats_type ON db_stats(type)",
		"CREATE INDEX IF NOT EXISTS idx_db_stats_expiration_size ON db_stats(expiration, size)",
	}

	for _, indexSQL := range indexes {
		if _, err := s.DB.Exec(indexSQL); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	log.Println("Indexes created successfully")
	return nil
}

// checkExistingData checks if the db_stats table already contains data
// Returns error if data exists (requirement 1c)
func checkExistingData(s *types.Storage) error {
	var exists int
	err := s.DB.QueryRow("SELECT 1 FROM db_stats LIMIT 1").Scan(&exists)
	if err == sql.ErrNoRows {
		// Table is empty, this is good
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to check existing data: %w", err)
	}

	// If we got here, table has data
	return fmt.Errorf("database already contains data in db_stats table; please use an empty database or delete existing data")
}

// AddKey adds a key to the batch and flushes if batch is full
func AddKey(s *types.Storage, key types.KeyMetadata) error {
	s.Batch = append(s.Batch, key)

	if len(s.Batch) >= s.BatchSize {
		return Flush(s)
	}

	return nil
}

// Flush writes the current batch to the database
func Flush(s *types.Storage) error {
	if len(s.Batch) == 0 {
		return nil
	}

	// Begin transaction
	tx, err := s.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Prepare statement
	stmt, err := tx.Prepare(`
		INSERT INTO db_stats (
			database, key, type, size, size_readable,
			element_count, encoding, expiration
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Execute inserts
	for _, key := range s.Batch {
		var expiration sql.NullTime
		if key.Expiration != nil {
			expiration = sql.NullTime{
				Time:  *key.Expiration,
				Valid: true,
			}
		}

		_, err := stmt.Exec(
			key.Database,
			key.Key,
			key.Type,
			key.Size,
			key.SizeReadable,
			key.ElementCount,
			key.Encoding,
			expiration,
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert row: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Update counters and progress bar
	batchSize := len(s.Batch)
	s.TotalRows += batchSize
	s.BatchNumber++

	// Update progress bar
	s.ProgressBar.Add(batchSize)

	// Clear batch
	s.Batch = s.Batch[:0]

	return nil
}

// FinishProgress completes the progress bar display
func FinishProgress(s *types.Storage) {
	if s.ProgressBar != nil {
		s.ProgressBar.Finish()
	}
}

// LogError writes an error message that doesn't interfere with the progress bar
func LogError(s *types.Storage, message string) {
	if s.ProgressBar != nil {
		s.ProgressBar.Clear()
		log.Println(message)
		return
	}

	log.Println(message)
}

// Close flushes any remaining data and closes the database
func Close(s *types.Storage) error {
	// Flush remaining data
	if err := Flush(s); err != nil {
		return err
	}

	// Close database
	return s.DB.Close()
}
