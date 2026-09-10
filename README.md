# 🔐 Go CLI Login System with Optional 2FA

A secure, containerized command-line login system built in Go. Features user registration, authentication, optional TOTP-based two-factor authentication (Google Authenticator compatible), session management, and account lockout protection.

Uses a lightweight CLI with hidden password input, terminal QR code generation, and SQLite for persistence.

---

## ✨ Features

- **User Registration** — Create accounts with validated username and password (with confirmation)
- **Secure Authentication** — Login with bcrypt-hashed passwords
- **TOTP 2FA** — Optional Google Authenticator compatible two-factor authentication
- **QR Code Generation** — Scannable QR code displayed in terminal for easy 2FA setup
- **Session Management** — UUID-based sessions with configurable timeout
- **Account Lockout** — Automatic lockout after configurable failed login attempts
- **Hidden Password Input** — Passwords are never displayed on screen (uses `golang.org/x/term`)
- **Colored Output** — ANSI-styled terminal output for clear feedback
- **Containerized** — Docker + Docker Compose for easy deployment
- **Data Persistence** — SQLite database persists across container restarts

---

## 📁 Project Structure

```
.
├── cmd/
│   └── cli/
│       └── main.go                 # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go              # Environment-based configuration
│   ├── cli/
│   │   └── cli.go                 # Interactive CLI (input, commands, output)
│   ├── db/
│   │   ├── db.go                  # SQLite connection setup
│   │   ├── db_test.go             # Database tests
│   │   └── migrations.go          # Schema migrations
│   ├── models/
│   │   ├── user.go                # User & Session data models
│   │   └── user_test.go           # Model tests
│   ├── repository/
│   │   └── user_repository.go     # Database access layer
│   ├── service/
│   │   ├── auth_service.go        # Authentication logic
│   │   ├── auth_service_test.go   # Auth tests
│   │   ├── session_service.go     # Session management
│   │   ├── totp_service.go        # TOTP 2FA logic
│   │   └── totp_service_test.go   # TOTP tests
├── data/                           # SQLite database (auto-created)
├── Dockerfile                      # Multi-stage build
├── docker-compose.yml             # Container orchestration
├── .env.example                   # Environment variable template
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

---

## 🚀 Getting Started

### Prerequisites

- [Docker](https://www.docker.com/get-started) & Docker Compose
- OR [Go 1.21+](https://go.dev/dl/) for local development

### Run with Docker (Recommended)

```bash
# Clone the repository
git clone https://github.com/akshayvibe/GoBackend.git
cd GoBackend

# Build and run
docker compose run --rm cli-app
```

The SQLite database is stored in a Docker named volume (`app-data`) and persists across container restarts.

### Run Locally

```bash
# Clone the repository
git clone https://github.com/akshayvibe/GoBackend.git
cd GoBackend

# Install dependencies
go mod download

# Run the application
go run ./cmd/cli/

# Or build and run the binary
go build -o cli-login ./cmd/cli/
./cli-login
```

---

## 📋 Available Commands

### Before Login

| Command    | Description                              |
|------------|------------------------------------------|
| `register` | Create a new user account                |
| `login`    | Login with username and password         |
| `help`     | Show available commands                  |
| `exit`     | Quit the program                         |

### After Login

| Command       | Description                                |
|---------------|--------------------------------------------|
| `whoami`      | Show current user details                  |
| `enable-2fa`  | Enable TOTP-based 2FA (displays QR code)   |
| `disable-2fa` | Disable two-factor authentication          |
| `logout`      | End current session                        |
| `help`        | Show available commands                    |

---

## 🔐 Security Features

### Password Requirements
- Minimum 8 characters
- Must contain at least one uppercase letter
- Must contain at least one lowercase letter
- Must contain at least one digit

### Password Storage
- Bcrypt hashing with configurable cost factor (default: 12)
- Passwords are never logged, displayed, or stored in command history

### Account Lockout
- Locks after 5 failed login attempts (configurable)
- Lockout duration: 15 minutes (configurable)
- Counter resets on successful login

### Session Management
- Cryptographically secure UUID-based session tokens
- Configurable timeout (default: 30 minutes)
- Sessions are stored in the database and validated on each command

### Two-Factor Authentication
- TOTP-based (Time-based One-Time Password)
- Compatible with Google Authenticator, Authy, and similar apps
- **QR code displayed in terminal** for easy scanning
- Verification required before enabling (ensures proper setup)
- Verification required before disabling (prevents unauthorized changes)

---

## ⚙️ Configuration

Configuration is done via environment variables. See [`.env.example`](.env.example) for all options:

| Variable                   | Default       | Description                          |
|----------------------------|---------------|--------------------------------------|
| `DB_PATH`                  | `./data/app.db` | Path to SQLite database file       |
| `SESSION_TIMEOUT_MINUTES`  | `30`          | Session expiration time              |
| `MAX_FAILED_ATTEMPTS`      | `5`           | Failed logins before lockout         |
| `LOCKOUT_DURATION_MINUTES` | `15`          | Duration of account lockout          |
| `BCRYPT_COST`              | `12`          | Bcrypt hashing cost factor           |
| `LOG_LEVEL`                | `info`        | Logging level (debug/info/warn/error)|

---

## 🧪 Running Tests

```bash
# Run all tests
go test ./... -v

# Run specific package tests
go test ./internal/service/ -v
go test ./internal/models/ -v
go test ./internal/db/ -v
```

---

## 📦 Database Schema

```sql
CREATE TABLE users (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    username        TEXT UNIQUE NOT NULL,
    password_hash   TEXT NOT NULL,
    totp_secret     TEXT DEFAULT '',
    totp_enabled    INTEGER DEFAULT 0,
    failed_attempts INTEGER DEFAULT 0,
    locked_until    DATETIME,
    created_at      DATETIME DEFAULT (datetime('now')),
    last_login      DATETIME
);

CREATE TABLE sessions (
    id          TEXT PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at  DATETIME NOT NULL,
    created_at  DATETIME DEFAULT (datetime('now'))
);
```

---

## 🏗️ Architecture

The project follows a clean layered architecture:

```
┌─────────────────────────────────────────┐
│           CLI (bufio + x/term)          │  ← User interaction
├─────────────────────────────────────────┤
│            Service Layer                │  ← Business logic
│  (Auth, Session, TOTP)                  │
├─────────────────────────────────────────┤
│           Repository Layer              │  ← Data access
├─────────────────────────────────────────┤
│        Database (SQLite)                │  ← Persistence
└─────────────────────────────────────────┘
```

- **CLI Layer** — Handles user input (with hidden passwords), command routing, and colored output
- **Service Layer** — Contains all business logic (authentication, sessions, TOTP)
- **Repository Layer** — Abstracts database queries with parameterized SQL
- **Database Layer** — SQLite with WAL mode, foreign keys, and auto-migrations

---

## 📝 Usage Example

```
$ go run ./cmd/cli/

   ██████╗  ██████╗     ██████╗██╗     ██╗
  ██╔════╝ ██╔═══██╗   ██╔════╝██║     ██║
  ██║  ███╗██║   ██║   ██║     ██║     ██║
  ██║   ██║██║   ██║   ██║     ██║     ██║
  ╚██████╔╝╚██████╔╝   ╚██████╗███████╗██║
   ╚═════╝  ╚═════╝     ╚═════╝╚══════╝╚═╝
   Secure Login System with 2FA

❯ register
  Username: john_doe
  Password: ••••••••
  Confirm Password: ••••••••
✅ User 'john_doe' registered successfully! You can now login.

❯ login
  Username: john_doe
  Password: ••••••••
✅ Login Successful!
  ╭────────────────────────────────────────────╮
  │  👤 Username:        john_doe              │
  │  📅 Registered:      2024-01-15 10:30:00   │
  │  🔐 2FA Status:      ✗ Disabled            │
  │  ⏰ Session Expires:  2024-01-15 11:00:00   │
  ╰────────────────────────────────────────────╯

[john_doe] ❯ enable-2fa
  Scan this QR code with Google Authenticator:
  ██████████████████████████████
  ██ ▄▄▄▄▄ █ ▀█ █▀█ ▄▄▄▄▄ ██
  ██ █   █ █▀██ ▄▄█ █   █ ██
  ...
  Secret (manual entry): JBSWY3DPEHPK3PXP
  Code: 123456
✅ Two-Factor Authentication has been enabled successfully!

[john_doe] ❯ logout
👋 Logged out successfully.
```

---

## 📄 License

This project is part of a backend development assessment.
