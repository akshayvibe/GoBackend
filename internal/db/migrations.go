package db

import (
	"database/sql"
	"fmt"
	"log/slog"
)

// migrationsSQL contains the SQL statements to create the required schema.
// Using an embedded approach for simplicity since we have a single migration.
const migrationsSQL = `
CREATE TABLE IF NOT EXISTS users (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    username        TEXT UNIQUE NOT NULL,
    password_hash   TEXT NOT NULL,
    totp_secret     TEXT DEFAULT '',
    totp_enabled    INTEGER DEFAULT 0,
    failed_attempts INTEGER DEFAULT 0,
    locked_until    DATETIME,
    created_at      DATETIME DEFAULT (datetime('now')),
    last_login      DATETIME
);

CREATE TABLE IF NOT EXISTS sessions (
    id          TEXT PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at  DATETIME NOT NULL,
    created_at  DATETIME DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
`

// RunMigrations executes the database schema migrations.
// It uses CREATE IF NOT EXISTS so it is safe to run multiple times.
func RunMigrations(db *sql.DB) error {
	slog.Info("running database migrations")

	if _, err := db.Exec(migrationsSQL); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.Info("database migrations completed successfully")
	return nil
}
