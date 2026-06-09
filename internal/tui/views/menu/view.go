package menu

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/stanley/sql-cli/internal/tui/common"
	"github.com/stanley/sql-cli/internal/tui/components"
)

// View renders the connection manager screen depending on its internal state.
func (m MenuViewModel) View() string {
	switch m.state {
	case stateCreate:
		return m.renderForm("  New Connection")
	case stateEdit:
		return m.renderForm("  Edit Connection")
	case stateDelete:
		return m.renderDeleteConfirm()
	default:
		return m.connList.View()
	}
}

func (m MenuViewModel) renderForm(title string) string {
	labels := []string{"Connection Name", "Driver", "DSN / Path"}

	var rows strings.Builder
	for i, inp := range m.form.inputs {
		label := lipgloss.NewStyle().
			Foreground(common.ColorMuted).
			Render(labels[i])

		focused := formField(i) == m.form.focusIndex
		borderColor := common.ColorBorder
		if focused {
			borderColor = common.ColorBorderFocus
		}
		inputBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Width(52).
			Padding(0, 1).
			Render(inp.View())

		rows.WriteString(label + "\n" + inputBox + "\n\n")
	}

	var validMsg string
	if !m.form.isValid() {
		validMsg = "\n" + common.ErrorStyle.Render("  Name and DSN are required.")
	}

	hint := common.HelpStyle.Render("tab · shift+tab  navigate fields    enter  save    esc  cancel")

	inner := lipgloss.JoinVertical(
		lipgloss.Left,
		common.TitleStyle.Render(title),
		"",
		rows.String(),
		hint,
		validMsg,
	)

	formBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(common.ColorPrimary).
		Padding(1, 3).
		Width(64).
		Render(inner)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, formBox)
}

func (m MenuViewModel) renderDeleteConfirm() string {
	sel := m.connList.Selected()
	if sel == nil {
		return ""
	}
	c := components.NewConfirm("Confirm Deletion", fmt.Sprintf("Do you want to remove %q?", sel.Name))
	return c.View(m.width, m.height)
}
