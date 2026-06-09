package menu

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stanley/sql-cli/internal/config"
	"github.com/stanley/sql-cli/internal/tui/common"
	"github.com/stanley/sql-cli/internal/tui/components"
)

const listCardWidth = 64

var (
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(common.ColorBorder).
			Width(listCardWidth).
			Padding(0, 2)

	cardSelectedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(common.ColorPrimary).
				Width(listCardWidth).
				Padding(0, 2)

	connNameStyle = lipgloss.NewStyle().
			Foreground(common.ColorText).
			Bold(true)

	connNameSelectedStyle = lipgloss.NewStyle().
				Foreground(common.ColorPrimary).
				Bold(true)

	driverBadgeStyle = func(driver string) lipgloss.Style {
		clr := common.ColorAccent
		switch strings.ToLower(driver) {
		case "sqlite":
			clr = common.ColorSuccess
		case "postgres":
			clr = common.ColorPrimary
		case "mysql":
			clr = common.ColorWarning
		}
		return lipgloss.NewStyle().Foreground(clr).Bold(true)
	}

	dsnStyle = lipgloss.NewStyle().Foreground(common.ColorMuted).Italic(true)

	dateStyle = lipgloss.NewStyle().Foreground(common.ColorMuted)

	listSubtitleStyle = lipgloss.NewStyle().Foreground(common.ColorMuted)

	listHintStyle = lipgloss.NewStyle().Foreground(common.ColorMuted).Italic(true)
)

// ConnectionListModel displays the selectable card list of saved database connections.
type ConnectionListModel struct {
	connections []config.Connection
	cursor      int
	width       int
	height      int
}

// NewConnectionList initializes a new ConnectionListModel.
func NewConnectionList(conns []config.Connection) ConnectionListModel {
	return ConnectionListModel{connections: sortedConnections(conns)}
}

// SetConnections updates the connection list and resets cursor boundary if needed.
func (m ConnectionListModel) SetConnections(conns []config.Connection) ConnectionListModel {
	sorted := sortedConnections(conns)
	if m.cursor >= len(sorted) {
		m.cursor = max(0, len(sorted)-1)
	}
	m.connections = sorted
	return m
}

// UpdateSize sets terminal dimensions for layout calculations.
func (m ConnectionListModel) UpdateSize(w, h int) ConnectionListModel {
	m.width = w
	m.height = h
	return m
}

// Update handles scroll keys (arrows and j/k).
func (m ConnectionListModel) Update(msg tea.Msg) (ConnectionListModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.connections)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

// Selected returns the currently selected connection, or nil.
func (m ConnectionListModel) Selected() *config.Connection {
	if len(m.connections) == 0 || m.cursor >= len(m.connections) {
		return nil
	}
	c := m.connections[m.cursor]
	return &c
}

// Len returns the connection count.
func (m ConnectionListModel) Len() int { return len(m.connections) }

// View renders the connection manager lists.
func (m ConnectionListModel) View() string {
	if m.width == 0 {
		return ""
	}

	var sections []string

	if m.height > 22 {
		sections = append(sections, components.RenderBanner())
	} else if m.height > 14 {
		div := lipgloss.NewStyle().Foreground(common.ColorBorder).Render(strings.Repeat("─", 44))
		sections = append(sections, div)
	}

	sections = append(sections, m.renderSubtitle())

	if len(m.connections) == 0 {
		sections = append(sections, m.renderEmpty())
	} else {
		sections = append(sections, m.renderCards())
	}

	sections = append(sections, m.renderHints())

	content := lipgloss.JoinVertical(lipgloss.Center, sections...)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

func (m ConnectionListModel) renderSubtitle() string {
	n := len(m.connections)
	var label string
	switch n {
	case 0:
		label = "no saved connections"
	case 1:
		label = "1 connection  ·  n new  ·  e edit  ·  d delete"
	default:
		label = fmt.Sprintf("%d connections  ·  n new  ·  e edit  ·  d delete", n)
	}
	return "\n" + listSubtitleStyle.Render(label) + "\n"
}

func (m ConnectionListModel) renderEmpty() string {
	return lipgloss.NewStyle().
		Foreground(common.ColorMuted).
		Italic(true).
		Width(listCardWidth + 4).
		Align(lipgloss.Center).
		Padding(2, 0).
		Render("No saved connections yet.\nPress  n  to create your first connection.")
}

func (m ConnectionListModel) renderCards() string {
	cardHeight := 5
	visibleMax := (m.height - 20) / cardHeight
	if visibleMax < 2 {
		visibleMax = 2
	}

	start := 0
	if m.cursor >= visibleMax {
		start = m.cursor - visibleMax + 1
	}
	end := start + visibleMax
	if end > len(m.connections) {
		end = len(m.connections)
	}

	var cards []string

	if start > 0 {
		cards = append(cards, lipgloss.NewStyle().
			Foreground(common.ColorMuted).
			Width(listCardWidth+4).
			Align(lipgloss.Center).
			Render(fmt.Sprintf("∧  %d above", start)))
	}

	for i := start; i < end; i++ {
		cards = append(cards, m.renderCard(m.connections[i], i == m.cursor))
	}

	below := len(m.connections) - end
	if below > 0 {
		cards = append(cards, lipgloss.NewStyle().
			Foreground(common.ColorMuted).
			Width(listCardWidth+4).
			Align(lipgloss.Center).
			Render(fmt.Sprintf("∨  %d below", below)))
	}

	return lipgloss.JoinVertical(lipgloss.Center, cards...)
}

func (m ConnectionListModel) renderCard(conn config.Connection, selected bool) string {
	style := cardStyle
	nameStyle := connNameStyle
	icon := "○"
	if selected {
		style = cardSelectedStyle
		nameStyle = connNameSelectedStyle
		icon = "●"
	}

	innerWidth := listCardWidth - 4
	nameText := nameStyle.Render(icon + "  " + conn.Name)
	badgeText := driverBadgeStyle(conn.Driver).Render(strings.ToUpper(conn.Driver))

	nameLen := utf8.RuneCountInString(icon + "  " + conn.Name)
	badgeLen := utf8.RuneCountInString(strings.ToUpper(conn.Driver))
	spacerLen := innerWidth - nameLen - badgeLen
	if spacerLen < 1 {
		spacerLen = 1
	}

	nameLine := nameText + strings.Repeat(" ", spacerLen) + badgeText

	dsn := conn.DSN
	maxDSN := innerWidth - 2
	if utf8.RuneCountInString(dsn) > maxDSN {
		runes := []rune(dsn)
		dsn = string(runes[:maxDSN-1]) + "…"
	}

	return style.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		nameLine,
		dsnStyle.Render("  "+dsn),
		dateStyle.Render("  Added on "+formatDate(conn.CreatedAt)),
	))
}

func (m ConnectionListModel) renderHints() string {
	line := "↑↓ / j k  navigate   ·   enter  connect   ·   n  new   ·   e  edit   ·   d  delete"
	return "\n" + listHintStyle.
		Width(listCardWidth+4).
		Align(lipgloss.Center).
		Render(line)
}

func sortedConnections(conns []config.Connection) []config.Connection {
	sorted := make([]config.Connection, len(conns))
	copy(sorted, conns)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})
	return sorted
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	months := [...]string{
		"jan", "feb", "mar", "apr", "may", "jun",
		"jul", "aug", "sep", "oct", "nov", "dec",
	}
	return fmt.Sprintf("%02d %s %d", t.Day(), months[t.Month()-1], t.Year())
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
