package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/akshayvibe/GoBackend/internal/repository"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTP configuration constants.
const (
	totpIssuer = "GoBackend-CLI"
)

// Sentinel errors for TOTP operations.
var (
	ErrTOTPAlreadyEnabled  = errors.New("2FA is already enabled")
	ErrTOTPAlreadyDisabled = errors.New("2FA is already disabled")
	ErrInvalidTOTPCode     = errors.New("invalid TOTP code")
)

// TOTPService handles TOTP-based two-factor authentication.
type TOTPService struct {
	repo *repository.UserRepository
}

// NewTOTPService creates a new TOTPService.
func NewTOTPService(repo *repository.UserRepository) *TOTPService {
	return &TOTPService{repo: repo}
}

// GenerateSecret creates a new TOTP secret for the given user.
// Returns the OTP key which contains the secret and the provisioning URI
// for scanning with Google Authenticator.
func (s *TOTPService) GenerateSecret(username string) (*otp.Key, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      totpIssuer,
		AccountName: username,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	slog.Info("TOTP secret generated", "username", username)
	return key, nil
}

// EnableTOTP stores the TOTP secret and enables 2FA for the user after verifying
// the provided code. This ensures the user has correctly set up their authenticator app.
func (s *TOTPService) EnableTOTP(ctx context.Context, userID int, secret, code string) error {
	// Validate the code against the secret to ensure proper setup
	if !totp.Validate(code, secret) {
		return ErrInvalidTOTPCode
	}

	// Store the secret and enable TOTP
	if err := s.repo.UpdateTOTPSecret(ctx, userID, secret, true); err != nil {
		return fmt.Errorf("failed to enable TOTP: %w", err)
	}

	// Invalidate active sessions as a security best practice upon security level changes
	_ = s.repo.DeleteUserSessions(ctx, userID)

	slog.Info("TOTP enabled for user", "user_id", userID)
	return nil
}

// DisableTOTP removes the TOTP secret and disables 2FA for the user.
// Requires a valid TOTP code for security verification.
func (s *TOTPService) DisableTOTP(ctx context.Context, userID int, secret, code string) error {
	// Verify the current TOTP code before disabling
	if !totp.Validate(code, secret) {
		return ErrInvalidTOTPCode
	}

	// Clear the secret and disable TOTP
	if err := s.repo.UpdateTOTPSecret(ctx, userID, "", false); err != nil {
		return fmt.Errorf("failed to disable TOTP: %w", err)
	}

	// Invalidate active sessions as a security best practice upon security level changes
	_ = s.repo.DeleteUserSessions(ctx, userID)

	slog.Info("TOTP disabled for user", "user_id", userID)
	return nil
}

// ValidateCode checks whether a TOTP code is valid for the given secret.
func (s *TOTPService) ValidateCode(secret, code string) bool {
	return totp.Validate(code, secret)
}
