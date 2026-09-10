// Package config provides application configuration loaded from environment variables.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration values.
type Config struct {
	// DBPath is the filesystem path to the SQLite database file.
	DBPath string

	// SessionTimeout is the duration after which a session expires.
	SessionTimeout time.Duration

	// MaxFailedAttempts is the number of failed login attempts before account lockout.
	MaxFailedAttempts int

	// LockoutDuration is how long an account stays locked after exceeding max failed attempts.
	LockoutDuration time.Duration

	// BcryptCost is the bcrypt hashing cost factor.
	BcryptCost int

	// LogLevel controls the logging verbosity (debug, info, warn, error).
	LogLevel string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		DBPath:            getEnv("DB_PATH", "./data/app.db"),
		SessionTimeout:    time.Duration(getEnvInt("SESSION_TIMEOUT_MINUTES", 30)) * time.Minute,
		MaxFailedAttempts: getEnvInt("MAX_FAILED_ATTEMPTS", 5),
		LockoutDuration:   time.Duration(getEnvInt("LOCKOUT_DURATION_MINUTES", 15)) * time.Minute,
		BcryptCost:        getEnvInt("BCRYPT_COST", 12),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
	}
}

// getEnv retrieves a string environment variable or returns the default value.
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// getEnvInt retrieves an integer environment variable or returns the default value.
func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}
