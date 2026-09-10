// Package tui provides the terminal user interface using the Bubble Tea framework.
package tui

import "github.com/charmbracelet/lipgloss"

// Color palette for the application.
var (
	primaryColor   = lipgloss.Color("#7C3AED") // Purple
	successColor   = lipgloss.Color("#10B981") // Green
	errorColor     = lipgloss.Color("#EF4444") // Red
	warningColor   = lipgloss.Color("#F59E0B") // Amber
	infoColor      = lipgloss.Color("#3B82F6") // Blue
	mutedColor     = lipgloss.Color("#6B7280") // Gray
	accentColor    = lipgloss.Color("#EC4899") // Pink
	bgColor        = lipgloss.Color("#1F2937") // Dark background
	textColor      = lipgloss.Color("#F9FAFB") // Light text
)

// Styles used throughout the TUI.
var (
	// TitleStyle is used for the application header/banner.
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	// PromptStyle is used for the command prompt indicator.
	PromptStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	// SuccessStyle is used for success messages.
	SuccessStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	// ErrorStyle is used for error messages.
	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	// WarningStyle is used for warning messages.
	WarningStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	// InfoStyle is used for informational messages.
	InfoStyle = lipgloss.NewStyle().
			Foreground(infoColor)

	// MutedStyle is used for less important text.
	MutedStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// AccentStyle is used for highlighted text elements.
	AccentStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	// CommandStyle is used to style command names in help text.
	CommandStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			PaddingRight(2)

	// DescriptionStyle is used for command descriptions.
	DescriptionStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	// UserInfoBoxStyle creates a bordered box for user details.
	UserInfoBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(1, 2).
				MarginTop(1).
				MarginBottom(1)

	// LabelStyle is used for field labels in user info.
	LabelStyle = lipgloss.NewStyle().
			Foreground(infoColor).
			Bold(true).
			Width(22)

	// ValueStyle is used for field values in user info.
	ValueStyle = lipgloss.NewStyle().
			Foreground(textColor)

	// BannerStyle is the large application banner.
	BannerStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	// SeparatorStyle is used for visual separators.
	SeparatorStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// QRStyle is used for the TOTP setup URI.
	QRStyle = lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true)
)
