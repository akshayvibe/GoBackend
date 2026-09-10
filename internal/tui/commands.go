package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/akshayvibe/GoBackend/internal/models"
	"github.com/akshayvibe/GoBackend/internal/service"
)

// commandHandler processes a CLI command and returns the output to display.
type commandHandler struct {
	authService    *service.AuthService
	sessionService *service.SessionService
	totpService    *service.TOTPService
}

// newCommandHandler creates a new command handler.
func newCommandHandler(auth *service.AuthService, session *service.SessionService, totp *service.TOTPService) *commandHandler {
	return &commandHandler{
		authService:    auth,
		sessionService: session,
		totpService:    totp,
	}
}

// preLoginCommands lists the commands available before authentication.
var preLoginCommands = []struct {
	name string
	desc string
}{
	{"register", "Create a new user account"},
	{"login", "Login with username and password"},
	{"help", "Show available commands"},
	{"exit", "Quit the program"},
}

// postLoginCommands lists the commands available after authentication.
var postLoginCommands = []struct {
	name string
	desc string
}{
	{"whoami", "Show current user details"},
	{"enable-2fa", "Enable TOTP-based two-factor authentication"},
	{"disable-2fa", "Disable two-factor authentication"},
	{"logout", "End current session"},
	{"help", "Show available commands"},
}

// helpText generates the help output for the current state.
func helpText(loggedIn bool) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(InfoStyle.Render("📋 Available Commands:"))
	sb.WriteString("\n\n")

	cmds := preLoginCommands
	if loggedIn {
		cmds = postLoginCommands
	}

	for _, cmd := range cmds {
		sb.WriteString(fmt.Sprintf("  %s %s\n",
			CommandStyle.Render(cmd.name),
			DescriptionStyle.Render(cmd.desc),
		))
	}
	sb.WriteString("\n")

	return sb.String()
}

// formatUserInfo renders user details in a styled box.
func formatUserInfo(user *models.User, session *models.Session) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(SuccessStyle.Render("✅ Login Successful!"))
	sb.WriteString("\n")

	// Build user info rows
	rows := []struct {
		label string
		value string
	}{
		{"👤 Username", user.Username},
		{"📅 Registered", user.CreatedAt.Local().Format("2006-01-02 15:04:05")},
		{"🔐 2FA Status", formatMFAStatus(user.TOTPEnabled)},
		{"⏰ Session Expires", session.ExpiresAt.Local().Format("2006-01-02 15:04:05")},
	}

	if user.LastLogin != nil {
		rows = append(rows, struct {
			label string
			value string
		}{"🕐 Last Login", user.LastLogin.Local().Format("2006-01-02 15:04:05")})
	}

	var infoLines strings.Builder
	for _, row := range rows {
		infoLines.WriteString(fmt.Sprintf("%s %s\n",
			LabelStyle.Render(row.label+":"),
			ValueStyle.Render(row.value),
		))
	}

	sb.WriteString(UserInfoBoxStyle.Render(infoLines.String()))
	sb.WriteString("\n")

	return sb.String()
}

// formatMFAStatus returns a styled MFA status string.
func formatMFAStatus(enabled bool) string {
	if enabled {
		return SuccessStyle.Render("Enabled ✓")
	}
	return WarningStyle.Render("Disabled ✗")
}

// handleRegister processes the register command flow.
func (h *commandHandler) handleRegister(username, password string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := h.authService.Register(ctx, username, password)
	if err != nil {
		return ErrorStyle.Render(fmt.Sprintf("❌ Registration failed: %s", err.Error()))
	}

	return SuccessStyle.Render(fmt.Sprintf("✅ User '%s' registered successfully! You can now login.", username))
}

// handleLogin processes the login command flow.
// Returns: output message, user (if authenticated), needs TOTP flag.
func (h *commandHandler) handleLogin(username, password string) (string, *models.User, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user, err := h.authService.Login(ctx, username, password)
	if err != nil {
		return ErrorStyle.Render(fmt.Sprintf("❌ Login failed: %s", err.Error())), nil, false
	}

	// Check if TOTP is required
	if user.TOTPEnabled {
		return InfoStyle.Render("🔐 2FA is enabled. Please enter your TOTP code:"), user, true
	}

	return "", user, false
}

// handleTOTPVerify verifies a TOTP code during login.
func (h *commandHandler) handleTOTPVerify(user *models.User, code string) (string, *models.Session) {
	if !h.totpService.ValidateCode(user.TOTPSecret, code) {
		return ErrorStyle.Render("❌ Invalid TOTP code. Login aborted."), nil
	}

	// Create session
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := h.sessionService.CreateSession(ctx, user.ID)
	if err != nil {
		return ErrorStyle.Render(fmt.Sprintf("❌ Failed to create session: %s", err.Error())), nil
	}

	return formatUserInfo(user, session), session
}

// createSessionForUser creates a session and returns the formatted user info.
func (h *commandHandler) createSessionForUser(user *models.User) (string, *models.Session) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := h.sessionService.CreateSession(ctx, user.ID)
	if err != nil {
		return ErrorStyle.Render(fmt.Sprintf("❌ Failed to create session: %s", err.Error())), nil
	}

	return formatUserInfo(user, session), session
}

// handleWhoami returns the current user's details.
func (h *commandHandler) handleWhoami(user *models.User, session *models.Session) string {
	// Re-fetch user to get latest data
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	freshUser, err := h.authService.GetUser(ctx, user.ID)
	if err != nil || freshUser == nil {
		return formatUserInfo(user, session)
	}

	return formatUserInfo(freshUser, session)
}

// handleEnableTOTP starts the 2FA enable flow.
// Returns the output message and the TOTP secret (to be verified in the next step).
func (h *commandHandler) handleEnableTOTP(user *models.User) (string, string) {
	if user.TOTPEnabled {
		return ErrorStyle.Render("❌ " + service.ErrTOTPAlreadyEnabled.Error()), ""
	}

	key, err := h.totpService.GenerateSecret(user.Username)
	if err != nil {
		return ErrorStyle.Render(fmt.Sprintf("❌ Failed to generate 2FA secret: %s", err.Error())), ""
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(InfoStyle.Render("🔐 Setting up Two-Factor Authentication"))
	sb.WriteString("\n\n")
	sb.WriteString("  Add this account to Google Authenticator or any TOTP app:\n\n")
	sb.WriteString(fmt.Sprintf("  %s %s\n\n",
		LabelStyle.Render("Secret:"),
		QRStyle.Render(key.Secret()),
	))
	sb.WriteString(fmt.Sprintf("  %s %s\n\n",
		LabelStyle.Render("URI:"),
		MutedStyle.Render(key.URL()),
	))
	sb.WriteString(InfoStyle.Render("  Enter the 6-digit code from your authenticator app to verify:"))
	sb.WriteString("\n")

	return sb.String(), key.Secret()
}

// handleVerifyAndEnableTOTP verifies a TOTP code and enables 2FA.
func (h *commandHandler) handleVerifyAndEnableTOTP(userID int, secret, code string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := h.totpService.EnableTOTP(ctx, userID, secret, code); err != nil {
		return ErrorStyle.Render(fmt.Sprintf("❌ Failed to enable 2FA: %s", err.Error()))
	}

	return SuccessStyle.Render("✅ Two-Factor Authentication has been enabled successfully!")
}

// handleDisableTOTP verifies a TOTP code and disables 2FA.
func (h *commandHandler) handleDisableTOTP(user *models.User, code string) string {
	if !user.TOTPEnabled {
		return ErrorStyle.Render("❌ " + service.ErrTOTPAlreadyDisabled.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := h.totpService.DisableTOTP(ctx, user.ID, user.TOTPSecret, code); err != nil {
		return ErrorStyle.Render(fmt.Sprintf("❌ Failed to disable 2FA: %s", err.Error()))
	}

	return SuccessStyle.Render("✅ Two-Factor Authentication has been disabled.")
}

// handleLogout destroys the session and logs out.
func (h *commandHandler) handleLogout(sessionID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := h.sessionService.DestroySession(ctx, sessionID); err != nil {
		return ErrorStyle.Render(fmt.Sprintf("❌ Logout failed: %s", err.Error()))
	}

	return SuccessStyle.Render("👋 Logged out successfully.")
}

// availableCommands returns command names for the current state (used for tab completion).
func availableCommands(loggedIn bool) []string {
	if loggedIn {
		return []string{"whoami", "enable-2fa", "disable-2fa", "logout", "help"}
	}
	return []string{"register", "login", "help", "exit"}
}
