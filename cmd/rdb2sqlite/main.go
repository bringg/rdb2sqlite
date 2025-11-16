package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"rdb2sqlite/pkg/parser"
	"rdb2sqlite/pkg/storage"
)

// Version information (set by ldflags during build)
var (
	Version   = "1.0.0"
	BuildDate = "dev"
	GitCommit = "unknown"
)

func main() {
	// Define command-line flags
	rdbPath := flag.String("rdb", "", "Path to the RDB file (required)")
	dbPath := flag.String("db", "", "Path to the SQLite database file (required)")
	batchSize := flag.Int("batch-size", 10000, "Batch size for inserts")
	skipErrors := flag.Bool("skip-errors", false, "Continue parsing on errors instead of stopping")
	version := flag.Bool("version", false, "Show version information")

	flag.Parse()

	// Handle version flag
	if *version {
		fmt.Printf("rdb2sqlite version %s\n", Version)
		fmt.Printf("Build date: %s\n", BuildDate)
		fmt.Printf("Git commit: %s\n", GitCommit)
		os.Exit(0)
	}

	// Validate required flags
	if *rdbPath == "" {
		fmt.Fprintf(os.Stderr, "Error: --rdb flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	if *dbPath == "" {
		fmt.Fprintf(os.Stderr, "Error: --db flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Check if RDB file exists
	if _, err := os.Stat(*rdbPath); os.IsNotExist(err) {
		log.Fatalf("Error: RDB file not found: %s\n", *rdbPath)
	}

	// Initialize storage
	log.Printf("Initializing SQLite database: %s\n", *dbPath)
	store, err := storage.New(*dbPath, *batchSize)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v\n", err)
	}
	defer func() {
		if err := storage.Close(store); err != nil {
			log.Printf("Error closing storage: %v\n", err)
		}
	}()

	// Parse RDB file
	log.Printf("Parsing RDB file: %s\n", *rdbPath)
	rdbParser := parser.New(store, *skipErrors)
	if err := rdbParser.Parse(*rdbPath); err != nil {
		log.Fatalf("Failed to parse RDB file: %v\n", err)
	}

	// Report skipped keys if any
	if skipped := store.SkippedKeys; skipped > 0 {
		log.Printf("Warning: Skipped %d keys due to errors\n", skipped)
	}

	// Create indexes after all data is inserted (much faster)
	if err := storage.CreateIndexes(store); err != nil {
		log.Fatalf("Failed to create indexes: %v\n", err)
	}

	// Final report
	log.Printf("Done! Imported %d rows into %s.db_stats\n", store.TotalRows, *dbPath)
}
