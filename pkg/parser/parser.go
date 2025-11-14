package parser

import (
	"fmt"
	"os"

	"rdb2sqlite/pkg/storage"
	"rdb2sqlite/pkg/types"

	"github.com/hdt3213/rdb/bytefmt"
	"github.com/hdt3213/rdb/core"
	"github.com/hdt3213/rdb/model"
)

// Parser handles parsing of RDB files
type Parser struct {
	storage    *types.Storage
	skipErrors bool
}

// New creates a new RDB parser
func New(store *types.Storage, skipErrors bool) *Parser {
	return &Parser{
		storage:    store,
		skipErrors: skipErrors,
	}
}

// Parse parses the RDB file and streams data to storage
// Uses the same approach as helper/memory.go from the rdb library
func (p *Parser) Parse(rdbPath string) error {
	file, err := os.Open(rdbPath)
	if err != nil {
		return fmt.Errorf("failed to open RDB file: %w", err)
	}
	defer file.Close()

	// Use core.NewDecoder for better metadata extraction
	decoder := core.NewDecoder(file)

	// Parse the RDB file with a callback
	err = decoder.Parse(func(o model.RedisObject) bool {
		// Create metadata using the library's built-in methods
		// This is more accurate than manual calculations
		metadata := types.KeyMetadata{
			Database:     o.GetDBIndex(),
			Key:          o.GetKey(),
			Type:         o.GetType(),
			Size:         int64(o.GetSize()),
			SizeReadable: bytefmt.FormatSize(uint64(o.GetSize())),
			ElementCount: o.GetElemCount(),
			Encoding:     o.GetEncoding(),
			Expiration:   o.GetExpiration(),
		}

		// Add to storage
		if err := storage.AddKey(p.storage, metadata); err != nil {
			if p.skipErrors {
				// Log error and skip this key (using progress bar writer to avoid conflicts)
				storage.LogError(p.storage, fmt.Sprintf("Error adding key '%s': %v (skipped)", o.GetKey(), err))
				p.storage.SkippedKeys++
				return true // Continue parsing
			}

			// Stop on error
			storage.LogError(p.storage, fmt.Sprintf("Error adding key '%s': %v", o.GetKey(), err))
			return false
		}

		return true // Continue parsing
	})

	if err != nil {
		return fmt.Errorf("failed to parse RDB file: %w", err)
	}

	// Finish progress bar when parsing is complete
	storage.FinishProgress(p.storage)

	return nil
}
