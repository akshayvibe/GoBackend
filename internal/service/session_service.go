package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/akshayvibe/GoBackend/internal/config"
	"github.com/akshayvibe/GoBackend/internal/models"
	"github.com/akshayvibe/GoBackend/internal/repository"
	"github.com/google/uuid"
)

// SessionService manages user sessions including creation, validation, and cleanup.
type SessionService struct {
	repo   *repository.UserRepository
	config *config.Config
}

// NewSessionService creates a new SessionService.
func NewSessionService(repo *repository.UserRepository, cfg *config.Config) *SessionService {
	return &SessionService{
		repo:   repo,
		config: cfg,
	}
}

// CreateSession generates a new session for the given user.
// The session is assigned a cryptographically random UUID and expires
// after the configured timeout duration.
func (s *SessionService) CreateSession(ctx context.Context, userID int) (*models.Session, error) {
	session := &models.Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		ExpiresAt: time.Now().Add(s.config.SessionTimeout),
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	slog.Info("session created",
		"session_id", session.ID,
		"user_id", userID,
		"expires_at", session.ExpiresAt.Format(time.RFC3339))
	return session, nil
}

// ValidateSession checks if a session exists and is not expired.
// Returns the session if valid, or nil if expired/not found.
func (s *SessionService) ValidateSession(ctx context.Context, sessionID string) (*models.Session, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate session: %w", err)
	}
	if session == nil {
		return nil, nil
	}

	if session.IsExpired() {
		// Clean up the expired session
		if delErr := s.repo.DeleteSession(ctx, sessionID); delErr != nil {
			slog.Error("failed to delete expired session", "error", delErr)
		}
		slog.Info("session expired", "session_id", sessionID)
		return nil, nil
	}

	return session, nil
}

// DestroySession removes a session from the database (logout).
func (s *SessionService) DestroySession(ctx context.Context, sessionID string) error {
	if err := s.repo.DeleteSession(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to destroy session: %w", err)
	}

	slog.Info("session destroyed (logout)", "session_id", sessionID)
	return nil
}

// CleanupExpired removes all expired sessions from the database.
func (s *SessionService) CleanupExpired(ctx context.Context) (int64, error) {
	return s.repo.DeleteExpiredSessions(ctx)
}
