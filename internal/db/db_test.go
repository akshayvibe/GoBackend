package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	// Create a temp directory for the test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Open database
	database, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer database.Close()

	// Verify the database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}

	// Verify we can ping it
	if err := database.Ping(); err != nil {
		t.Errorf("database ping failed: %v", err)
	}
}

func TestRunMigrations(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer database.Close()

	// Run migrations
	if err := RunMigrations(database); err != nil {
		t.Fatalf("RunMigrations() error = %v", err)
	}

	// Verify tables exist
	tables := []string{"users", "sessions"}
	for _, table := range tables {
		var name string
		err := database.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}

	// Run migrations again (should be idempotent)
	if err := RunMigrations(database); err != nil {
		t.Errorf("RunMigrations() second run error = %v", err)
	}
}
