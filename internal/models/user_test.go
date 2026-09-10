package models

import (
	"testing"
	"time"
)

func TestUser_IsLocked(t *testing.T) {
	tests := []struct {
		name        string
		lockedUntil *time.Time
		want        bool
	}{
		{
			name:        "not locked (nil)",
			lockedUntil: nil,
			want:        false,
		},
		{
			name:        "locked (future time)",
			lockedUntil: timePtr(time.Now().Add(15 * time.Minute)),
			want:        true,
		},
		{
			name:        "lock expired (past time)",
			lockedUntil: timePtr(time.Now().Add(-1 * time.Minute)),
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{LockedUntil: tt.lockedUntil}
			if got := u.IsLocked(); got != tt.want {
				t.Errorf("User.IsLocked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSession_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "not expired (future)",
			expiresAt: time.Now().Add(30 * time.Minute),
			want:      false,
		},
		{
			name:      "expired (past)",
			expiresAt: time.Now().Add(-1 * time.Minute),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Session{ExpiresAt: tt.expiresAt}
			if got := s.IsExpired(); got != tt.want {
				t.Errorf("Session.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
