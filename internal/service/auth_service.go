// Package service provides the business logic for authentication, sessions, and TOTP.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode"

	"github.com/akshayvibe/GoBackend/internal/config"
	"github.com/akshayvibe/GoBackend/internal/models"
	"github.com/akshayvibe/GoBackend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// Sentinel errors for authentication operations.
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("username already taken")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrAccountLocked      = errors.New("account is locked due to too many failed attempts")
	ErrWeakPassword       = errors.New("password must be at least 8 characters with uppercase, lowercase, and digit")
	ErrInvalidUsername     = errors.New("username must be 3-50 alphanumeric characters")
	ErrTOTPRequired       = errors.New("TOTP code required")
)

// AuthService handles user registration and authentication.
type AuthService struct {
	repo   *repository.UserRepository
	config *config.Config
}

// NewAuthService creates a new AuthService.
func NewAuthService(repo *repository.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{
		repo:   repo,
		config: cfg,
	}
}

// Register creates a new user account with the given username and password.
// It validates input, hashes the password with bcrypt, and stores the user.
func (s *AuthService) Register(ctx context.Context, username, password string) (*models.User, error) {
	// Validate username
	if err := validateUsername(username); err != nil {
		return nil, err
	}

	// Validate password strength
	if err := validatePassword(password); err != nil {
		return nil, err
	}

	// Check if username already exists
	exists, err := s.repo.UserExists(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("registration check failed: %w", err)
	}
	if exists {
		return nil, ErrUserAlreadyExists
	}

	// Hash the password with bcrypt
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.config.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create the user in the database
	user, err := s.repo.CreateUser(ctx, username, string(hash))
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	slog.Info("user registered successfully", "username", username)
	return user, nil
}

// Login authenticates a user with the given credentials.
// It returns the user if authentication is successful, or an error describing the failure.
// The caller must handle TOTP verification separately if the user has 2FA enabled.
func (s *AuthService) Login(ctx context.Context, username, password string) (*models.User, error) {
	// Look up the user
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("login lookup failed: %w", err)
	}
	if user == nil {
		slog.Warn("login attempt for non-existent user", "username", username)
		return nil, ErrInvalidCredentials
	}

	// Check account lockout
	if user.IsLocked() {
		remaining := time.Until(*user.LockedUntil).Round(time.Second)
		slog.Warn("login attempt on locked account", "username", username,
			"locked_until", user.LockedUntil, "remaining", remaining)
		return nil, fmt.Errorf("%w: try again in %s", ErrAccountLocked, remaining)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		// Increment failed attempts
		if incErr := s.repo.IncrementFailedAttempts(
			ctx, user.ID, s.config.MaxFailedAttempts, s.config.LockoutDuration,
		); incErr != nil {
			slog.Error("failed to increment failed attempts", "error", incErr)
		}

		remaining := s.config.MaxFailedAttempts - user.FailedAttempts - 1
		if remaining > 0 {
			slog.Warn("failed login attempt", "username", username,
				"remaining_attempts", remaining)
		}
		return nil, ErrInvalidCredentials
	}

	// Password is correct — reset failed attempts and update last login
	if err := s.repo.ResetFailedAttempts(ctx, user.ID); err != nil {
		slog.Error("failed to reset failed attempts", "error", err)
	}

	// Re-fetch user to get updated last_login
	user, err = s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh user data: %w", err)
	}

	slog.Info("user authenticated successfully", "username", username,
		"totp_enabled", user.TOTPEnabled)
	return user, nil
}

// GetUser retrieves a user by ID. Used to refresh user data for display.
func (s *AuthService) GetUser(ctx context.Context, userID int) (*models.User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// validateUsername checks that the username meets requirements.
func validateUsername(username string) error {
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(username) > 50 {
		return ErrInvalidUsername
	}
	for _, r := range username {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			return ErrInvalidUsername
		}
	}
	return nil
}

// validatePassword checks that the password meets minimum security requirements.
func validatePassword(password string) error {
	if len(password) < 8 {
		return ErrWeakPassword
	}

	var hasUpper, hasLower, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return ErrWeakPassword
	}

	return nil
}
