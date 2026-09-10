// Package main is the entry point for the CLI login system.
package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/akshayvibe/GoBackend/internal/cli"
	"github.com/akshayvibe/GoBackend/internal/config"
	"github.com/akshayvibe/GoBackend/internal/db"
	"github.com/akshayvibe/GoBackend/internal/repository"
	"github.com/akshayvibe/GoBackend/internal/service"
)

func main() {
	// Load configuration from environment variables.
	cfg := config.Load()

	// Configure structured logging.
	setupLogger(cfg.LogLevel)

	slog.Info("starting CLI Login System",
		"db_path", cfg.DBPath,
		"session_timeout", cfg.SessionTimeout,
		"max_failed_attempts", cfg.MaxFailedAttempts,
		"lockout_duration", cfg.LockoutDuration,
	)

	// Initialize the SQLite database.
	database, err := db.New(cfg.DBPath)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// Run database migrations.
	if err := db.RunMigrations(database); err != nil {
		slog.Error("failed to run migrations", "error", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Initialize the repository and service layers.
	repo := repository.NewUserRepository(database)
	authService := service.NewAuthService(repo, cfg)
	sessionService := service.NewSessionService(repo, cfg)
	totpService := service.NewTOTPService(repo)

	// Create and run the CLI application.
	app := cli.New(authService, sessionService, totpService)
	app.Run()
}

// setupLogger configures the structured logger based on the log level.
func setupLogger(level string) {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	// Log to a file so CLI output isn't disrupted.
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// Fall back to stderr if log file can't be opened
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))
		return
	}

	handler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))
}
