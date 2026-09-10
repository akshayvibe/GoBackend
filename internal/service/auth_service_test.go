package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/akshayvibe/GoBackend/internal/config"
	"github.com/akshayvibe/GoBackend/internal/db"
	"github.com/akshayvibe/GoBackend/internal/repository"
	"github.com/pquerna/otp/totp"
)

func setupServices(t *testing.T) (*AuthService, *TOTPService, *SessionService) {
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

	repo := repository.NewUserRepository(database)
	cfg := config.Load()
	cfg.BcryptCost = 4 // Fast bcrypt for tests

	authSvc := NewAuthService(repo, cfg)
	totpSvc := NewTOTPService(repo)
	sessSvc := NewSessionService(repo, cfg)

	return authSvc, totpSvc, sessSvc
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{"valid username", "john_doe", false},
		{"valid with hyphen", "john-doe", false},
		{"valid with digits", "user123", false},
		{"too short", "ab", true},
		{"empty", "", true},
		{"too long", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true},
		{"has spaces", "john doe", true},
		{"has special chars", "john@doe", true},
		{"min length", "abc", false},
		{"max length", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateUsername(%q) error = %v, wantErr %v", tt.username, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid password", "Password1", false},
		{"valid complex", "MyP@ssw0rd!", false},
		{"too short", "Pass1", true},
		{"no uppercase", "password1", true},
		{"no lowercase", "PASSWORD1", true},
		{"no digit", "Password", true},
		{"empty", "", true},
		{"just spaces", "        ", true},
		{"8 chars valid", "Abcdefg1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePassword(%q) error = %v, wantErr %v", tt.password, err, tt.wantErr)
			}
		})
	}
}

func TestAuthService_RegisterAndLoginWith2FA(t *testing.T) {
	authSvc, totpSvc, sessSvc := setupServices(t)
	ctx := context.Background()

	// 1. Register user
	user, err := authSvc.Register(ctx, "secuser", "Secur1tyP@ss")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if user.Username != "secuser" {
		t.Errorf("expected username secuser, got %s", user.Username)
	}

	// 2. Login without 2FA initially
	loggedInUser, err := authSvc.Login(ctx, "secuser", "Secur1tyP@ss")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if loggedInUser.TOTPEnabled {
		t.Error("2FA should be disabled initially")
	}

	// Create session
	sess, err := sessSvc.CreateSession(ctx, loggedInUser.ID)
	if err != nil || sess == nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// 3. Enable 2FA
	key, err := totpSvc.GenerateSecret(user.Username)
	if err != nil {
		t.Fatalf("GenerateSecret failed: %v", err)
	}

	validCode, err := totp.GenerateCode(key.Secret(), time.Now())
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}

	if err := totpSvc.EnableTOTP(ctx, user.ID, key.Secret(), validCode); err != nil {
		t.Fatalf("EnableTOTP failed: %v", err)
	}

	// 4. Verify user data now has 2FA enabled
	updatedUser, err := authSvc.GetUser(ctx, user.ID)
	if err != nil || !updatedUser.TOTPEnabled {
		t.Fatalf("User 2FA flag should be true")
	}

	// 5. Login after 2FA enabled
	user2FA, err := authSvc.Login(ctx, "secuser", "Secur1tyP@ss")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if !user2FA.TOTPEnabled {
		t.Error("User should have 2FA enabled on login")
	}

	// 6. Test 2FA validation (correct code vs invalid code)
	newCode, err := totp.GenerateCode(user2FA.TOTPSecret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}

	if !totpSvc.ValidateCode(user2FA.TOTPSecret, newCode) {
		t.Error("ValidateCode should pass for valid code")
	}

	if totpSvc.ValidateCode(user2FA.TOTPSecret, "000000") {
		t.Error("ValidateCode should fail for wrong code '000000'")
	}
}
