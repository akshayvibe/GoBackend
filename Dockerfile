# ============================================================================
# Multi-stage Dockerfile for Go CLI Login System
# Stage 1: Build the Go binary
# Stage 2: Create a minimal runtime image
# ============================================================================

# --- Build Stage ---
FROM golang:1.23-alpine AS builder

# Install build dependencies (git needed for go modules)
RUN apk add --no-cache git

WORKDIR /app

# Copy dependency files first for better Docker layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code
COPY . .

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/cli-login ./cmd/cli/

# --- Runtime Stage ---
FROM alpine:3.19

# Install ca-certificates for any HTTPS calls and tzdata for timezone support
RUN apk add --no-cache ca-certificates tzdata

# Create a non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy the binary from the build stage
COPY --from=builder /app/cli-login .

# Create data directory for SQLite and set ownership
RUN mkdir -p /app/data && chown -R appuser:appgroup /app/data

# Switch to non-root user
USER appuser

# Set environment variables
ENV DB_PATH=/app/data/app.db
ENV SESSION_TIMEOUT_MINUTES=30
ENV MAX_FAILED_ATTEMPTS=5
ENV LOCKOUT_DURATION_MINUTES=15
ENV BCRYPT_COST=12
ENV LOG_LEVEL=info

# The data directory is mounted as a volume for persistence
VOLUME ["/app/data"]

# Run the CLI application with an interactive terminal
ENTRYPOINT ["./cli-login"]
