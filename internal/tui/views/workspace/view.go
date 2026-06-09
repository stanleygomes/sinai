package workspace

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/stanley/sql-cli/internal/tui/common"
)

// View renders the workspace vertical layout split.
func (m MainViewModel) View() string {
	if m.width == 0 {
		return ""
	}

	leftW, rightW := m.panelWidths()

	leftPanel := m.renderLeftPanel(leftW)
	rightPanel := m.renderRightPanel(rightW)

	workspace := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	statusLine := m.renderInternalStatus()

	return lipgloss.JoinVertical(lipgloss.Left, workspace, statusLine)
}

func (m MainViewModel) renderLeftPanel(w int) string {
	style := common.PanelStyle.Width(w - 2)
	if m.focus == FocusLeft {
		style = common.PanelFocusStyle.Width(w - 2)
	}

	header := lipgloss.NewStyle().
		Foreground(common.ColorPrimary).
		Bold(true).
		MarginBottom(1).
		Render("◧ Tables")

	var body string
	if !m.tablesReady {
		body = common.HelpStyle.Render("Loading...")
	} else if len(m.tables) == 0 {
		body = common.HelpStyle.Render("No tables found.")
	} else {
		var rows strings.Builder
		for i, t := range m.tables {
			prefix := "  "
			lineStyle := lipgloss.NewStyle().Foreground(common.ColorText)
			if i == m.tablesCursor {
				prefix = "▶ "
				lineStyle = lineStyle.Foreground(common.ColorPrimary).Bold(true)
			}
			rows.WriteString(lineStyle.Render(prefix+t) + "\n")
		}
		body = rows.String()
	}

	hint := "\n" + common.HelpStyle.Render("enter: SELECT  ctrl+d: DDL  tab: switch focus")
	content := lipgloss.JoinVertical(lipgloss.Left, header, body, hint)

	return style.Height(m.height - 4).Render(content)
}

func (m MainViewModel) renderRightPanel(w int) string {
	style := common.PanelStyle.Width(w - 2)
	if m.focus == FocusRight {
		style = common.PanelFocusStyle.Width(w - 2)
	}

	tabs := m.renderTabs(w - 4)
	content := m.renderActiveTab(w - 6)

	inner := lipgloss.JoinVertical(lipgloss.Left, tabs, content)
	return style.Height(m.height - 4).Render(inner)
}

func (m MainViewModel) renderTabs(w int) string {
	tabLabels := []string{"① SQL Editor", "② Data", "③ DDL"}
	var rendered []string

	for i, label := range tabLabels {
		if Tab(i) == m.activeTab {
			rendered = append(rendered, lipgloss.NewStyle().
				Foreground(common.ColorPrimary).
				Bold(true).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(common.ColorPrimary).
				Padding(0, 2).
				Render(label))
		} else {
			rendered = append(rendered, lipgloss.NewStyle().
				Foreground(common.ColorMuted).
				Padding(0, 2).
				Render(label))
		}
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Bottom, rendered...)
	hint := common.HelpStyle.Render("ctrl+←/→: switch tab")

	spacer := lipgloss.NewStyle().
		Width(w - lipgloss.Width(tabBar) - lipgloss.Width(hint)).
		Render("")

	return lipgloss.JoinHorizontal(lipgloss.Center, tabBar, spacer, hint)
}

func (m MainViewModel) renderActiveTab(w int) string {
	switch m.activeTab {
	case TabEditor:
		return m.renderEditorTab()
	case TabData:
		return m.renderDataTab()
	case TabDDL:
		return m.renderDDLTab()
	}
	return ""
}

func (m MainViewModel) renderEditorTab() string {
	var b strings.Builder

	hint := common.HelpStyle.Render("F5: execute query")
	if m.isRunning {
		hint = lipgloss.NewStyle().Foreground(common.ColorWarning).Render("⟳ Executing query...")
	}

	b.WriteString(hint + "\n")
	b.WriteString(m.editor.View())
	return b.String()
}

func (m MainViewModel) renderDataTab() string {
	if m.rowCount == 0 && m.queryDuration == 0 {
		return common.HelpStyle.Render("\nExecute a SELECT query to see results here.\n")
	}
	header := common.SuccessStyle.Render(fmt.Sprintf("%d row(s) · %s", m.rowCount, m.queryDuration.Round(time.Millisecond)))
	return lipgloss.JoinVertical(lipgloss.Left, header, m.dataTable.View())
}

func (m MainViewModel) renderDDLTab() string {
	if m.ddlContent == "" {
		return common.HelpStyle.Render("\nSelect a table in the left panel and press ctrl+d.\n")
	}
	return m.ddlViewport.View()
}

func (m MainViewModel) renderInternalStatus() string {
	if m.statusMsg == "" {
		return common.HelpStyle.Width(m.width).Render("  tab: switch panel  |  ctrl+←/→: tabs")
	}
	if m.statusIsError {
		return common.ErrorStyle.Width(m.width).Render("  ✗ " + m.statusMsg)
	}
	return common.SuccessStyle.Width(m.width).Render("  ✓ " + m.statusMsg)
}

func (m MainViewModel) panelWidths() (left, right int) {
	left = m.width / 4
	if left < 20 {
		left = 20
	}
	right = m.width - left
	return
}
