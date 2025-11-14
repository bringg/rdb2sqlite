package types

import (
	"database/sql"
	"time"

	"github.com/schollz/progressbar/v3"
)

// KeyMetadata represents the metadata for a single Redis key
type KeyMetadata struct {
	Database     int        // Redis DB index
	Key          string     // Key name
	Type         string     // Redis data type
	Size         int64      // Size in bytes
	SizeReadable string     // Human-readable size
	ElementCount int        // Number of elements for collections
	Encoding     string     // Encoding information
	Expiration   *time.Time // Absolute expiration time (stored as ISO-8601 TEXT), nil if no expiration
}

// Storage handles SQLite operations and batching
type Storage struct {
	DB          *sql.DB
	BatchSize   int
	Batch       []KeyMetadata
	TotalRows   int
	BatchNumber int
	SkippedKeys int
	ProgressBar *progressbar.ProgressBar
}
