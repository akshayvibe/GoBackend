// Package repository provides data access methods for the database.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/akshayvibe/GoBackend/internal/models"
)

// UserRepository handles all database operations related to users and sessions.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new UserRepository with the given database connection.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser inserts a new user into the database.
func (r *UserRepository) CreateUser(ctx context.Context, username, passwordHash string) (*models.User, error) {
	query := `INSERT INTO users (username, password_hash) VALUES (?, ?) RETURNING id, created_at`

	var user models.User
	user.Username = username
	user.PasswordHash = passwordHash

	err := r.db.QueryRowContext(ctx, query, username, passwordHash).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	slog.Info("user created", "username", username, "id", user.ID)
	return &user, nil
}

// GetUserByUsername retrieves a user by their username.
// Returns nil, nil if the user is not found.
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `SELECT id, username, password_hash, totp_secret, totp_enabled, 
	           failed_attempts, locked_until, created_at, last_login 
	           FROM users WHERE username = ?`

	var user models.User
	var lockedUntil sql.NullTime
	var lastLogin sql.NullTime
	var totpSecret sql.NullString

	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.PasswordHash,
		&totpSecret, &user.TOTPEnabled,
		&user.FailedAttempts, &lockedUntil,
		&user.CreatedAt, &lastLogin,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	if lockedUntil.Valid {
		user.LockedUntil = &lockedUntil.Time
	}
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}
	if totpSecret.Valid {
		user.TOTPSecret = totpSecret.String
	}

	return &user, nil
}

// GetUserByID retrieves a user by their ID.
func (r *UserRepository) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	query := `SELECT id, username, password_hash, totp_secret, totp_enabled, 
	           failed_attempts, locked_until, created_at, last_login 
	           FROM users WHERE id = ?`

	var user models.User
	var lockedUntil sql.NullTime
	var lastLogin sql.NullTime
	var totpSecret sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.PasswordHash,
		&totpSecret, &user.TOTPEnabled,
		&user.FailedAttempts, &lockedUntil,
		&user.CreatedAt, &lastLogin,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	if lockedUntil.Valid {
		user.LockedUntil = &lockedUntil.Time
	}
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}
	if totpSecret.Valid {
		user.TOTPSecret = totpSecret.String
	}

	return &user, nil
}

// UserExists checks if a username is already taken.
func (r *UserRepository) UserExists(ctx context.Context, username string) (bool, error) {
	query := `SELECT COUNT(1) FROM users WHERE username = ?`
	var count int
	err := r.db.QueryRowContext(ctx, query, username).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}
	return count > 0, nil
}

// IncrementFailedAttempts increases the failed login attempt counter for a user.
// If the counter reaches maxAttempts, the account is locked for the specified duration.
func (r *UserRepository) IncrementFailedAttempts(ctx context.Context, userID, maxAttempts int, lockoutDuration time.Duration) error {
	// First increment the counter
	updateQuery := `UPDATE users SET failed_attempts = failed_attempts + 1 WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, updateQuery, userID); err != nil {
		return fmt.Errorf("failed to increment failed attempts: %w", err)
	}

	// Check if we need to lock the account
	var failedAttempts int
	checkQuery := `SELECT failed_attempts FROM users WHERE id = ?`
	if err := r.db.QueryRowContext(ctx, checkQuery, userID).Scan(&failedAttempts); err != nil {
		return fmt.Errorf("failed to check failed attempts: %w", err)
	}

	if failedAttempts >= maxAttempts {
		lockUntil := time.Now().Add(lockoutDuration)
		lockQuery := `UPDATE users SET locked_until = ? WHERE id = ?`
		if _, err := r.db.ExecContext(ctx, lockQuery, lockUntil, userID); err != nil {
			return fmt.Errorf("failed to lock account: %w", err)
		}
		slog.Warn("account locked due to too many failed attempts",
			"user_id", userID, "locked_until", lockUntil)
	}

	return nil
}

// ResetFailedAttempts resets the failed login attempt counter and updates the last login time.
func (r *UserRepository) ResetFailedAttempts(ctx context.Context, userID int) error {
	query := `UPDATE users SET failed_attempts = 0, locked_until = NULL, last_login = datetime('now') WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, query, userID); err != nil {
		return fmt.Errorf("failed to reset failed attempts: %w", err)
	}
	return nil
}

// UpdateTOTPSecret sets the TOTP secret for a user.
func (r *UserRepository) UpdateTOTPSecret(ctx context.Context, userID int, secret string, enabled bool) error {
	query := `UPDATE users SET totp_secret = ?, totp_enabled = ? WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, query, secret, enabled, userID); err != nil {
		return fmt.Errorf("failed to update TOTP secret: %w", err)
	}

	slog.Info("TOTP updated", "user_id", userID, "enabled", enabled)
	return nil
}

// --- Session Operations ---

// CreateSession inserts a new session into the database.
func (r *UserRepository) CreateSession(ctx context.Context, session *models.Session) error {
	query := `INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`
	if _, err := r.db.ExecContext(ctx, query, session.ID, session.UserID, session.ExpiresAt); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	slog.Info("session created", "session_id", session.ID, "user_id", session.UserID,
		"expires_at", session.ExpiresAt)
	return nil
}

// GetSession retrieves a session by its ID.
func (r *UserRepository) GetSession(ctx context.Context, sessionID string) (*models.Session, error) {
	query := `SELECT id, user_id, expires_at, created_at FROM sessions WHERE id = ?`

	var session models.Session
	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(
		&session.ID, &session.UserID, &session.ExpiresAt, &session.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

// DeleteSession removes a session from the database.
func (r *UserRepository) DeleteSession(ctx context.Context, sessionID string) error {
	query := `DELETE FROM sessions WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, query, sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	slog.Info("session deleted", "session_id", sessionID)
	return nil
}

// DeleteExpiredSessions removes all sessions that have passed their expiration time.
func (r *UserRepository) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	query := `DELETE FROM sessions WHERE expires_at < datetime('now')`
	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	count, _ := result.RowsAffected()
	if count > 0 {
		slog.Info("expired sessions cleaned up", "count", count)
	}
	return count, nil
}
