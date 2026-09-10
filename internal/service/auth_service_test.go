package service

import (
	"testing"
)

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
		{"too long", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true}, // 51 chars
		{"has spaces", "john doe", true},
		{"has special chars", "john@doe", true},
		{"min length", "abc", false},
		{"max length", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false}, // 50 chars
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
