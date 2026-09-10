package repository

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/akshayvibe/GoBackend/internal/db"
	"github.com/akshayvibe/GoBackend/internal/models"
)

func setupTestDB(t *testing.T) *UserRepository {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := db.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if err := db.RunMigrations(database); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return NewUserRepository(database)
}

func TestUserRepository_UserOperations(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	// Create user
	user, err := repo.CreateUser(ctx, "testuser", "hashed_pass")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.ID == 0 || user.Username != "testuser" {
		t.Errorf("unexpected user returned: %+v", user)
	}

	// UserExists
	exists, err := repo.UserExists(ctx, "testuser")
	if err != nil || !exists {
		t.Errorf("UserExists returned %v, err %v", exists, err)
	}

	notExists, err := repo.UserExists(ctx, "nonexistent")
	if err != nil || notExists {
		t.Errorf("UserExists returned %v for nonexistent user", notExists)
	}

	// GetUserByUsername
	fetched, err := repo.GetUserByUsername(ctx, "testuser")
	if err != nil || fetched == nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}
	if fetched.ID != user.ID {
		t.Errorf("expected ID %d, got %d", user.ID, fetched.ID)
	}

	// GetUserByID
	fetchedByID, err := repo.GetUserByID(ctx, user.ID)
	if err != nil || fetchedByID == nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if fetchedByID.Username != "testuser" {
		t.Errorf("expected username testuser, got %s", fetchedByID.Username)
	}
}

func TestUserRepository_TOTP(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	user, err := repo.CreateUser(ctx, "totpuser", "hash")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Enable TOTP
	secret := "JBSWY3DPEHPK3PXP"
	if err := repo.UpdateTOTPSecret(ctx, user.ID, secret, true); err != nil {
		t.Fatalf("UpdateTOTPSecret failed: %v", err)
	}

	updated, err := repo.GetUserByID(ctx, user.ID)
	if err != nil || updated == nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if !updated.TOTPEnabled || updated.TOTPSecret != secret {
		t.Errorf("TOTP not set correctly: enabled=%v, secret=%s", updated.TOTPEnabled, updated.TOTPSecret)
	}

	// Disable TOTP
	if err := repo.UpdateTOTPSecret(ctx, user.ID, "", false); err != nil {
		t.Fatalf("UpdateTOTPSecret disable failed: %v", err)
	}

	disabled, err := repo.GetUserByID(ctx, user.ID)
	if err != nil || disabled == nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if disabled.TOTPEnabled || disabled.TOTPSecret != "" {
		t.Errorf("TOTP not disabled correctly: enabled=%v, secret=%s", disabled.TOTPEnabled, disabled.TOTPSecret)
	}
}

func TestUserRepository_FailedAttemptsAndLockout(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	user, err := repo.CreateUser(ctx, "lockuser", "hash")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Increment failed attempts
	maxAttempts := 3
	lockDuration := 15 * time.Minute

	for i := 1; i <= maxAttempts; i++ {
		if err := repo.IncrementFailedAttempts(ctx, user.ID, maxAttempts, lockDuration); err != nil {
			t.Fatalf("IncrementFailedAttempts failed on attempt %d: %v", i, err)
		}
	}

	lockedUser, err := repo.GetUserByID(ctx, user.ID)
	if err != nil || lockedUser == nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}

	if !lockedUser.IsLocked() {
		t.Error("user should be locked after reaching max attempts")
	}

	// Reset failed attempts
	if err := repo.ResetFailedAttempts(ctx, user.ID); err != nil {
		t.Fatalf("ResetFailedAttempts failed: %v", err)
	}

	unlockedUser, err := repo.GetUserByID(ctx, user.ID)
	if err != nil || unlockedUser == nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}

	if unlockedUser.IsLocked() || unlockedUser.FailedAttempts != 0 {
		t.Errorf("user should be unlocked and failed_attempts should be 0: locked=%v attempts=%d", unlockedUser.IsLocked(), unlockedUser.FailedAttempts)
	}
}

func TestUserRepository_SessionOperations(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	user, err := repo.CreateUser(ctx, "sessuser", "hash")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	sess := &models.Session{
		ID:        "session-12345",
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := repo.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	fetchedSess, err := repo.GetSession(ctx, "session-12345")
	if err != nil || fetchedSess == nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if fetchedSess.UserID != user.ID {
		t.Errorf("expected UserID %d, got %d", user.ID, fetchedSess.UserID)
	}

	// Delete session
	if err := repo.DeleteSession(ctx, "session-12345"); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	deletedSess, err := repo.GetSession(ctx, "session-12345")
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}
	if deletedSess != nil {
		t.Error("session should be deleted")
	}

	// Create multiple sessions and test DeleteUserSessions
	s1 := &models.Session{ID: "sess-1", UserID: user.ID, ExpiresAt: time.Now().Add(1 * time.Hour)}
	s2 := &models.Session{ID: "sess-2", UserID: user.ID, ExpiresAt: time.Now().Add(1 * time.Hour)}
	_ = repo.CreateSession(ctx, s1)
	_ = repo.CreateSession(ctx, s2)

	if err := repo.DeleteUserSessions(ctx, user.ID); err != nil {
		t.Fatalf("DeleteUserSessions failed: %v", err)
	}

	s1Check, _ := repo.GetSession(ctx, "sess-1")
	s2Check, _ := repo.GetSession(ctx, "sess-2")
	if s1Check != nil || s2Check != nil {
		t.Error("all user sessions should have been deleted")
	}
}
