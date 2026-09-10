// Package models defines the data structures used throughout the application.
package models

import "time"

// User represents a registered user in the system.
type User struct {
	ID             int        `json:"id"`
	Username       string     `json:"username"`
	PasswordHash   string     `json:"-"` // Never serialized
	TOTPSecret     string     `json:"-"` // Never serialized
	TOTPEnabled    bool       `json:"totp_enabled"`
	FailedAttempts int        `json:"failed_attempts"`
	LockedUntil    *time.Time `json:"locked_until,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	LastLogin      *time.Time `json:"last_login,omitempty"`
}

// IsLocked checks whether the user account is currently locked.
func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

// Session represents an active user session.
type Session struct {
	ID        string    `json:"id"`
	UserID    int       `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// IsExpired checks whether the session has passed its expiration time.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
