// Package db provides SQLite database initialization and migration management.
package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // Pure-Go SQLite driver
)

// New creates a new SQLite database connection and ensures the data directory exists.
// It configures WAL mode and foreign keys for optimal performance and data integrity.
func New(dbPath string) (*sql.DB, error) {
	// Ensure the directory for the database file exists with strict 0700 permissions.
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory %q: %w", dir, err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set file permissions to 0600 (owner read/write only) for database security
	_ = os.Chmod(dbPath, 0600)

	// Enable WAL mode for better concurrent read performance.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	// Enable foreign key constraint enforcement.
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Set a reasonable connection pool size for SQLite.
	db.SetMaxOpenConns(1) // SQLite supports only one writer at a time.
	db.SetMaxIdleConns(1)

	slog.Info("database connection established", "path", dbPath)
	return db, nil
}
