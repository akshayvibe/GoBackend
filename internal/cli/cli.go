// Package cli provides a lightweight interactive command-line interface.
// It uses bufio for input, golang.org/x/term for hidden password input,
// and ANSI escape codes for colored output.
package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/akshayvibe/GoBackend/internal/models"
	"github.com/akshayvibe/GoBackend/internal/service"
	"github.com/mdp/qrterminal/v3"
	"golang.org/x/term"
)

// ANSI color codes for terminal output.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

// App holds the CLI application state and service dependencies.
type App struct {
	scanner        *bufio.Scanner
	authService    *service.AuthService
	sessionService *service.SessionService
	totpService    *service.TOTPService

	// Session state
	loggedIn    bool
	currentUser *models.User
	session     *models.Session
}

// New creates a new CLI application.
func New(auth *service.AuthService, session *service.SessionService, totp *service.TOTPService) *App {
	return &App{
		scanner:        bufio.NewScanner(os.Stdin),
		authService:    auth,
		sessionService: session,
		totpService:    totp,
	}
}

// Run starts the interactive CLI loop.
func (a *App) Run() {
	a.printBanner()
	a.printHelp()

	for {
		// Check session expiry before each command
		if a.loggedIn && a.session != nil && a.session.IsExpired() {
			a.printWarning("Session expired. You have been logged out.")
			a.loggedIn = false
			a.currentUser = nil
			a.session = nil
		}

		a.printPrompt()

		if !a.scanner.Scan() {
			break // EOF or error
		}

		input := sanitizeInput(a.scanner.Text())
		if input == "" {
			continue
		}

		cmd := strings.ToLower(input)

		if !a.loggedIn {
			switch cmd {
			case "register":
				a.handleRegister()
			case "login":
				a.handleLogin()
			case "help":
				a.printHelp()
			case "exit", "quit":
				a.printSuccess("👋 Goodbye!")
				return
			default:
				a.printError("Unknown command: '%s'. Type 'help' for available commands.", cmd)
			}
		} else {
			switch cmd {
			case "whoami":
				a.handleWhoami()
			case "enable-2fa":
				a.handleEnable2FA()
			case "disable-2fa":
				a.handleDisable2FA()
			case "logout":
				a.handleLogout()
			case "help":
				a.printHelp()
			default:
				a.printError("Unknown command: '%s'. Type 'help' for available commands.", cmd)
			}
		}
	}
}

// --- Command Handlers ---

func (a *App) handleRegister() {
	fmt.Println()
	a.printInfo("📝 Register New Account")

	username := a.readLine("  Username: ")
	if username == "" {
		a.printError("Username cannot be empty.")
		return
	}

	password := a.readPassword("  Password: ")
	if password == "" {
		a.printError("Password cannot be empty.")
		return
	}

	confirmPassword := a.readPassword("  Confirm Password: ")
	if password != confirmPassword {
		a.printError("Passwords do not match.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := a.authService.Register(ctx, username, password)
	if err != nil {
		a.printError("Registration failed: %s", err.Error())
		return
	}

	a.printSuccess("✅ User '%s' registered successfully! You can now login.", username)
}

func (a *App) handleLogin() {
	fmt.Println()
	a.printInfo("🔑 Login")

	username := a.readLine("  Username: ")
	if username == "" {
		a.printError("Username cannot be empty.")
		return
	}

	password := a.readPassword("  Password: ")
	if password == "" {
		a.printError("Password cannot be empty.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user, err := a.authService.Login(ctx, username, password)
	if err != nil {
		a.printError("Login failed: %s", err.Error())
		return
	}

	// Check if TOTP is required
	if user.TOTPEnabled {
		a.printInfo("🔐 2FA is enabled. Enter your 6-digit TOTP code:")
		code := a.readLine("  Code: ")

		if code == "" || !a.totpService.ValidateCode(user.TOTPSecret, code) {
			_ = a.authService.RecordFailedAttempt(ctx, user.ID)
			a.printError("Invalid TOTP code. Login aborted.")
			return
		}
	}

	// Create session
	session, err := a.sessionService.CreateSession(ctx, user.ID)
	if err != nil {
		a.printError("Failed to create session: %s", err.Error())
		return
	}

	a.loggedIn = true
	a.currentUser = user
	a.session = session

	a.printSuccess("✅ Login Successful!")
	a.printUserInfo(user, session)
}

func (a *App) handleWhoami() {
	// Re-fetch user data for latest info
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user, err := a.authService.GetUser(ctx, a.currentUser.ID)
	if err != nil || user == nil {
		user = a.currentUser
	}

	a.printUserInfo(user, a.session)
}

func (a *App) handleEnable2FA() {
	if a.currentUser.TOTPEnabled {
		a.printError("2FA is already enabled.")
		return
	}

	key, err := a.totpService.GenerateSecret(a.currentUser.Username)
	if err != nil {
		a.printError("Failed to generate 2FA secret: %s", err.Error())
		return
	}

	fmt.Println()
	a.printInfo("🔐 Setting up Two-Factor Authentication")
	fmt.Println()

	// Display QR code in terminal
	fmt.Println("  Scan this QR code with Google Authenticator or any TOTP app:")
	fmt.Println()
	qrterminal.GenerateWithConfig(key.URL(), qrterminal.Config{
		Level:     qrterminal.M,
		Writer:    os.Stdout,
		BlackChar: qrterminal.BLACK,
		WhiteChar: qrterminal.WHITE,
		QuietZone: 2,
	})
	fmt.Println()

	fmt.Printf("  %sSecret (manual entry):%s %s%s%s\n", colorCyan, colorReset, colorBold, key.Secret(), colorReset)
	fmt.Println()

	a.printInfo("  Enter the 6-digit code from your authenticator app to verify:")
	code := a.readLine("  Code: ")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.totpService.EnableTOTP(ctx, a.currentUser.ID, key.Secret(), code); err != nil {
		a.printError("Failed to enable 2FA: %s", err.Error())
		return
	}

	a.currentUser.TOTPEnabled = true
	a.currentUser.TOTPSecret = key.Secret()
	a.printSuccess("✅ Two-Factor Authentication has been enabled successfully!")
}

func (a *App) handleDisable2FA() {
	if !a.currentUser.TOTPEnabled {
		a.printError("2FA is already disabled.")
		return
	}

	a.printInfo("🔐 Enter your current TOTP code to disable 2FA:")
	code := a.readLine("  Code: ")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.totpService.DisableTOTP(ctx, a.currentUser.ID, a.currentUser.TOTPSecret, code); err != nil {
		a.printError("Failed to disable 2FA: %s", err.Error())
		return
	}

	a.currentUser.TOTPEnabled = false
	a.currentUser.TOTPSecret = ""
	a.printSuccess("✅ Two-Factor Authentication has been disabled.")
}

func (a *App) handleLogout() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.sessionService.DestroySession(ctx, a.session.ID); err != nil {
		a.printError("Logout failed: %s", err.Error())
		return
	}

	a.loggedIn = false
	a.currentUser = nil
	a.session = nil

	a.printSuccess("👋 Logged out successfully.")
	fmt.Println()
	a.printHelp()
}

// --- Input Helpers ---

// readLine reads a line of visible text from stdin.
func (a *App) readLine(prompt string) string {
	fmt.Print(prompt)
	if a.scanner.Scan() {
		return sanitizeInput(a.scanner.Text())
	}
	return ""
}

// readPassword reads a password from the terminal with input hidden.
// Falls back to visible input if terminal raw mode is unavailable.
func (a *App) readPassword(prompt string) string {
	fmt.Print(prompt)

	// Use x/term for hidden password input
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		password, err := term.ReadPassword(fd)
		fmt.Println() // Move to next line after hidden input
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(password))
	}

	// Fallback for non-terminal (e.g., piped input in Docker)
	if a.scanner.Scan() {
		return strings.TrimSpace(a.scanner.Text())
	}
	return ""
}

// --- Output Helpers ---

func (a *App) printBanner() {
	banner := `
   ██████╗  ██████╗     ██████╗██╗     ██╗
  ██╔════╝ ██╔═══██╗   ██╔════╝██║     ██║
  ██║  ███╗██║   ██║   ██║     ██║     ██║
  ██║   ██║██║   ██║   ██║     ██║     ██║
  ╚██████╔╝╚██████╔╝   ╚██████╗███████╗██║
   ╚═════╝  ╚═════╝     ╚═════╝╚══════╝╚═╝
   Secure Login System with 2FA
`
	fmt.Printf("%s%s%s\n", colorPurple, banner, colorReset)
}

func (a *App) printPrompt() {
	if a.loggedIn {
		fmt.Printf("%s%s[%s]%s ❯ ", colorGreen, colorBold, a.currentUser.Username, colorReset)
	} else {
		fmt.Printf("%s%s❯%s ", colorPurple, colorBold, colorReset)
	}
}

func (a *App) printHelp() {
	fmt.Printf("\n%s📋 Available Commands:%s\n\n", colorCyan, colorReset)

	if !a.loggedIn {
		fmt.Printf("  %s%sregister%s   Create a new user account\n", colorPurple, colorBold, colorReset)
		fmt.Printf("  %s%slogin%s      Login with username and password\n", colorPurple, colorBold, colorReset)
		fmt.Printf("  %s%shelp%s       Show available commands\n", colorPurple, colorBold, colorReset)
		fmt.Printf("  %s%sexit%s       Quit the program\n", colorPurple, colorBold, colorReset)
	} else {
		fmt.Printf("  %s%swhoami%s       Show current user details\n", colorPurple, colorBold, colorReset)
		fmt.Printf("  %s%senable-2fa%s   Enable TOTP-based two-factor authentication\n", colorPurple, colorBold, colorReset)
		fmt.Printf("  %s%sdisable-2fa%s  Disable two-factor authentication\n", colorPurple, colorBold, colorReset)
		fmt.Printf("  %s%slogout%s       End current session\n", colorPurple, colorBold, colorReset)
		fmt.Printf("  %s%shelp%s         Show available commands\n", colorPurple, colorBold, colorReset)
	}
	fmt.Println()
}

func (a *App) printUserInfo(user *models.User, session *models.Session) {
	fmt.Println()
	fmt.Printf("  ╭────────────────────────────────────────────╮\n")
	fmt.Printf("  │  %s👤 Username:%s        %-22s │\n", colorCyan, colorReset, user.Username)
	fmt.Printf("  │  %s📅 Registered:%s      %-22s │\n", colorCyan, colorReset, user.CreatedAt.Local().Format("2006-01-02 15:04:05"))

	mfaStatus := fmt.Sprintf("%s✗ Disabled%s", colorYellow, colorReset)
	if user.TOTPEnabled {
		mfaStatus = fmt.Sprintf("%s✓ Enabled%s", colorGreen, colorReset)
	}
	fmt.Printf("  │  %s🔐 2FA Status:%s      %-31s │\n", colorCyan, colorReset, mfaStatus)
	fmt.Printf("  │  %s⏰ Session Expires:%s  %-22s │\n", colorCyan, colorReset, session.ExpiresAt.Local().Format("2006-01-02 15:04:05"))

	if user.LastLogin != nil {
		fmt.Printf("  │  %s🕐 Last Login:%s      %-22s │\n", colorCyan, colorReset, user.LastLogin.Local().Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("  ╰────────────────────────────────────────────╯\n")
	fmt.Println()
}

func (a *App) printSuccess(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s\n", colorGreen, msg, colorReset)
}

func (a *App) printError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s❌ %s%s\n", colorRed, msg, colorReset)
}

func (a *App) printInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s\n", colorCyan, msg, colorReset)
}

func (a *App) printWarning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s⚠️  %s%s\n", colorYellow, msg, colorReset)
}

// sanitizeInput strips non-printable control characters (such as arrow key escape codes) from user input.
func sanitizeInput(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if unicode.IsPrint(r) {
			sb.WriteRune(r)
		}
	}
	return strings.TrimSpace(sb.String())
}
