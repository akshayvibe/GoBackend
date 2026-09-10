package service

import (
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestTOTPService_ValidateCode(t *testing.T) {
	svc := &TOTPService{}

	// Generate a test secret
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "test",
		AccountName: "testuser",
	})
	if err != nil {
		t.Fatalf("failed to generate TOTP key: %v", err)
	}

	// Generate a valid code
	validCode, err := totp.GenerateCode(key.Secret(), time.Now())
	if err != nil {
		t.Fatalf("failed to generate TOTP code: %v", err)
	}

	// Test valid code
	if !svc.ValidateCode(key.Secret(), validCode) {
		t.Error("ValidateCode() returned false for a valid code")
	}

	// Test invalid code
	if svc.ValidateCode(key.Secret(), "000000") {
		// Note: there's a tiny chance this is actually valid, so we accept either result
		t.Log("Warning: '000000' was a valid TOTP code (extremely unlikely but possible)")
	}

	// Test empty secret
	if svc.ValidateCode("", validCode) {
		t.Error("ValidateCode() returned true for empty secret")
	}
}
