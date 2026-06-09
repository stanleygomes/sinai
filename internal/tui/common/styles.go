// Package common defines shared types, messages, and styling tokens for the TUI.
package common

import "charm.land/lipgloss/v2"

// Color palette (dark-themed).
var (
	ColorBackground  = lipgloss.Color("#1A1B26")
	ColorPrimary     = lipgloss.Color("#7AA2F7") // Tokyo Night Blue
	ColorAccent      = lipgloss.Color("#BB9AF7") // Purple
	ColorSuccess     = lipgloss.Color("#9ECE6A") // Green
	ColorWarning     = lipgloss.Color("#E0AF68") // Orange
	ColorDanger      = lipgloss.Color("#F7768E") // Red
	ColorMuted       = lipgloss.Color("#565F89")
	ColorBorder      = lipgloss.Color("#3B4261")
	ColorBorderFocus = lipgloss.Color("#7AA2F7")
	ColorText        = lipgloss.Color("#CDD6F4")
)

// Reusable Lipgloss styles.
var (
	// PanelStyle is the base style for panels with borders.
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	// PanelFocusStyle is the style for the currently focused panel.
	PanelFocusStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorderFocus).
			Padding(0, 1)

	// StatusBarStyle is the style for the bottom status bar.
	StatusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#16161E")).
			Foreground(ColorMuted).
			Padding(0, 1)

	// TitleStyle is the style for headers.
	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Padding(0, 1)

	// HelpStyle is the style for keyboard shortcut hints.
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)

	// ErrorStyle is the style for error messages.
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)

	// SuccessStyle is the style for success messages.
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)
)
