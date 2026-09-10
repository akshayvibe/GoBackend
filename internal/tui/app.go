package tui

import (
	"fmt"
	"strings"

	"github.com/akshayvibe/GoBackend/internal/models"
	"github.com/akshayvibe/GoBackend/internal/service"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// inputMode represents the current state of the CLI input.
type inputMode int

const (
	modeCommand          inputMode = iota // Waiting for a command
	modeRegisterUser                      // Entering username for registration
	modeRegisterPass                      // Entering password for registration
	modeLoginUser                         // Entering username for login
	modeLoginPass                         // Entering password for login
	modeTOTPLogin                         // Entering TOTP code during login
	modeTOTPEnable                        // Entering TOTP code to enable 2FA
	modeTOTPDisable                       // Entering TOTP code to disable 2FA
)

// Model is the main Bubble Tea model for the CLI application.
type Model struct {
	textInput textinput.Model
	handler   *commandHandler
	output    []string // Accumulated output lines
	mode      inputMode

	// Authentication state
	loggedIn     bool
	currentUser  *models.User
	session      *models.Session
	pendingUser  string // Username being registered or logging in
	pendingPass  string // Password being registered (held temporarily)
	totpSecret   string // Pending TOTP secret during enable flow
	pendingLogin *models.User // User pending TOTP verification during login

	// History
	history      []string
	historyIndex int

	// Window dimensions
	width  int
	height int

	// Quitting flag
	quitting bool
}

// NewModel creates a new Bubble Tea model.
func NewModel(auth *service.AuthService, session *service.SessionService, totp *service.TOTPService) Model {
	ti := textinput.New()
	ti.Placeholder = "Type a command..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50
	ti.Prompt = "❯ "
	ti.PromptStyle = PromptStyle
	ti.TextStyle = ValueStyle

	m := Model{
		textInput: ti,
		handler:   newCommandHandler(auth, session, totp),
		output:    []string{},
		mode:      modeCommand,
		history:   []string{},
	}

	// Show the welcome banner
	m.addOutput(banner())
	m.addOutput(helpText(false))

	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model — processes keyboard input and updates state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.quitting = true
			return m, tea.Quit

		case tea.KeyUp:
			// Navigate history up
			if len(m.history) > 0 && m.historyIndex > 0 {
				m.historyIndex--
				m.textInput.SetValue(m.history[m.historyIndex])
				m.textInput.CursorEnd()
			}
			return m, nil

		case tea.KeyDown:
			// Navigate history down
			if m.historyIndex < len(m.history)-1 {
				m.historyIndex++
				m.textInput.SetValue(m.history[m.historyIndex])
				m.textInput.CursorEnd()
			} else if m.historyIndex == len(m.history)-1 {
				m.historyIndex = len(m.history)
				m.textInput.SetValue("")
			}
			return m, nil

		case tea.KeyTab:
			// Tab completion
			return m.handleTabCompletion(), nil

		case tea.KeyEnter:
			input := strings.TrimSpace(m.textInput.Value())
			m.textInput.SetValue("")

			if input == "" {
				return m, nil
			}

			// Add to history (only for command mode)
			if m.mode == modeCommand {
				m.history = append(m.history, input)
			}
			m.historyIndex = len(m.history)

			return m.processInput(input)
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// processInput handles the submitted input based on the current mode.
func (m Model) processInput(input string) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeCommand:
		return m.processCommand(input)

	case modeRegisterUser:
		m.pendingUser = input
		m.addOutput(MutedStyle.Render("  Enter password:"))
		m.mode = modeRegisterPass
		m.textInput.EchoMode = textinput.EchoPassword
		m.textInput.EchoCharacter = '•'
		m.textInput.Placeholder = "Password"
		return m, nil

	case modeRegisterPass:
		result := m.handler.handleRegister(m.pendingUser, input)
		m.addOutput(result)
		m.resetToCommand()
		return m, nil

	case modeLoginUser:
		m.pendingUser = input
		m.addOutput(MutedStyle.Render("  Enter password:"))
		m.mode = modeLoginPass
		m.textInput.EchoMode = textinput.EchoPassword
		m.textInput.EchoCharacter = '•'
		m.textInput.Placeholder = "Password"
		return m, nil

	case modeLoginPass:
		msg, user, needsTOTP := m.handler.handleLogin(m.pendingUser, input)

		if user == nil {
			// Login failed
			m.addOutput(msg)
			m.resetToCommand()
			return m, nil
		}

		if needsTOTP {
			// Need TOTP verification
			m.addOutput(msg)
			m.pendingLogin = user
			m.mode = modeTOTPLogin
			m.textInput.EchoMode = textinput.EchoNormal
			m.textInput.Placeholder = "6-digit code"
			return m, nil
		}

		// Login successful without 2FA — create session
		sessionMsg, session := m.handler.createSessionForUser(user)
		if session != nil {
			m.loggedIn = true
			m.currentUser = user
			m.session = session
		}
		m.addOutput(sessionMsg)
		m.resetToCommand()
		return m, nil

	case modeTOTPLogin:
		msg, session := m.handler.handleTOTPVerify(m.pendingLogin, input)
		if session != nil {
			m.loggedIn = true
			m.currentUser = m.pendingLogin
			m.session = session
		}
		m.pendingLogin = nil
		m.addOutput(msg)
		m.resetToCommand()
		return m, nil

	case modeTOTPEnable:
		result := m.handler.handleVerifyAndEnableTOTP(m.currentUser.ID, m.totpSecret, input)
		m.addOutput(result)
		// Re-fetch user to update state
		if !strings.Contains(result, "❌") {
			m.currentUser.TOTPEnabled = true
			m.currentUser.TOTPSecret = m.totpSecret
		}
		m.totpSecret = ""
		m.resetToCommand()
		return m, nil

	case modeTOTPDisable:
		result := m.handler.handleDisableTOTP(m.currentUser, input)
		m.addOutput(result)
		if !strings.Contains(result, "❌") {
			m.currentUser.TOTPEnabled = false
			m.currentUser.TOTPSecret = ""
		}
		m.resetToCommand()
		return m, nil
	}

	return m, nil
}

// processCommand handles command-mode input.
func (m Model) processCommand(input string) (tea.Model, tea.Cmd) {
	cmd := strings.ToLower(strings.TrimSpace(input))

	if !m.loggedIn {
		switch cmd {
		case "register":
			m.addOutput(InfoStyle.Render("\n📝 Register New Account"))
			m.addOutput(MutedStyle.Render("  Enter username:"))
			m.mode = modeRegisterUser
			m.textInput.Placeholder = "Username"
			return m, nil

		case "login":
			m.addOutput(InfoStyle.Render("\n🔑 Login"))
			m.addOutput(MutedStyle.Render("  Enter username:"))
			m.mode = modeLoginUser
			m.textInput.Placeholder = "Username"
			return m, nil

		case "help":
			m.addOutput(helpText(false))
			return m, nil

		case "exit", "quit":
			m.addOutput(SuccessStyle.Render("👋 Goodbye!"))
			m.quitting = true
			return m, tea.Quit

		default:
			m.addOutput(ErrorStyle.Render(fmt.Sprintf("❌ Unknown command: '%s'. Type 'help' for available commands.", cmd)))
			return m, nil
		}
	}

	// Logged-in commands
	switch cmd {
	case "whoami":
		m.addOutput(m.handler.handleWhoami(m.currentUser, m.session))
		return m, nil

	case "enable-2fa":
		msg, secret := m.handler.handleEnableTOTP(m.currentUser)
		m.addOutput(msg)
		if secret != "" {
			m.totpSecret = secret
			m.mode = modeTOTPEnable
			m.textInput.Placeholder = "6-digit code"
		}
		return m, nil

	case "disable-2fa":
		if !m.currentUser.TOTPEnabled {
			m.addOutput(ErrorStyle.Render("❌ 2FA is already disabled."))
			return m, nil
		}
		m.addOutput(InfoStyle.Render("🔐 Enter your current TOTP code to disable 2FA:"))
		m.mode = modeTOTPDisable
		m.textInput.Placeholder = "6-digit code"
		return m, nil

	case "logout":
		result := m.handler.handleLogout(m.session.ID)
		m.addOutput(result)
		m.loggedIn = false
		m.currentUser = nil
		m.session = nil
		m.addOutput(helpText(false))
		return m, nil

	case "help":
		m.addOutput(helpText(true))
		return m, nil

	default:
		m.addOutput(ErrorStyle.Render(fmt.Sprintf("❌ Unknown command: '%s'. Type 'help' for available commands.", cmd)))
		return m, nil
	}
}

// handleTabCompletion provides command autocompletion.
func (m Model) handleTabCompletion() Model {
	if m.mode != modeCommand {
		return m
	}

	input := strings.ToLower(strings.TrimSpace(m.textInput.Value()))
	if input == "" {
		return m
	}

	cmds := availableCommands(m.loggedIn)
	var matches []string
	for _, c := range cmds {
		if strings.HasPrefix(c, input) {
			matches = append(matches, c)
		}
	}

	if len(matches) == 1 {
		m.textInput.SetValue(matches[0])
		m.textInput.CursorEnd()
	} else if len(matches) > 1 {
		m.addOutput(MutedStyle.Render("  " + strings.Join(matches, "  ")))
	}

	return m
}

// resetToCommand resets the input mode back to command entry.
func (m *Model) resetToCommand() {
	m.mode = modeCommand
	m.textInput.EchoMode = textinput.EchoNormal
	m.textInput.Placeholder = "Type a command..."
	m.pendingUser = ""
	m.pendingPass = ""
}

// addOutput appends a line to the output buffer.
func (m *Model) addOutput(s string) {
	m.output = append(m.output, s)
}

// View implements tea.Model — renders the current UI state.
func (m Model) View() string {
	if m.quitting {
		return SuccessStyle.Render("👋 Goodbye!") + "\n"
	}

	var sb strings.Builder

	// Render output history (show last N lines to fit the terminal)
	maxLines := m.height - 4 // Leave room for input + status bar
	if maxLines < 10 {
		maxLines = 30
	}

	startIdx := 0
	if len(m.output) > maxLines {
		startIdx = len(m.output) - maxLines
	}

	for _, line := range m.output[startIdx:] {
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	// Status bar
	if m.loggedIn {
		status := fmt.Sprintf(" 🟢 %s ", m.currentUser.Username)
		sb.WriteString(SeparatorStyle.Render(strings.Repeat("─", 50)))
		sb.WriteString("\n")
		sb.WriteString(SuccessStyle.Render(status))
		sb.WriteString("\n")
	} else {
		sb.WriteString(SeparatorStyle.Render(strings.Repeat("─", 50)))
		sb.WriteString("\n")
	}

	// Input prompt
	sb.WriteString(m.textInput.View())
	sb.WriteString("\n")

	return sb.String()
}

// banner returns the application startup banner.
func banner() string {
	b := `
   ██████╗  ██████╗     ██████╗██╗     ██╗
  ██╔════╝ ██╔═══██╗   ██╔════╝██║     ██║
  ██║  ███╗██║   ██║   ██║     ██║     ██║
  ██║   ██║██║   ██║   ██║     ██║     ██║
  ╚██████╔╝╚██████╔╝   ╚██████╗███████╗██║
   ╚═════╝  ╚═════╝     ╚═════╝╚══════╝╚═╝
   Secure Login System with 2FA`

	return BannerStyle.Render(b) + "\n"
}
