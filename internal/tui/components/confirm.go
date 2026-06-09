// Package components provides reusable TUI components.
package components

import (
	"charm.land/lipgloss/v2"
	"github.com/stanley/sql-cli/internal/tui/common"
)

// ConfirmModel is a reusable yes/no confirmation dialog.
type ConfirmModel struct {
	Title   string
	Message string
}

// NewConfirm creates a new confirmation dialog.
func NewConfirm(title, message string) ConfirmModel {
	return ConfirmModel{
		Title:   title,
		Message: message,
	}
}

// View renders the confirmation dialog placed in the center of the terminal.
func (c ConfirmModel) View(width, height int) string {
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		common.ErrorStyle.Render("⚠  "+c.Title),
		"",
		lipgloss.NewStyle().Foreground(common.ColorText).Render(c.Message),
		"",
		common.HelpStyle.Render("y / s  confirm    any other key  cancel"),
	)

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(common.ColorDanger).
		Padding(1, 4).
		Width(50).
		Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal)
}
