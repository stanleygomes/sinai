// Package tui — connection_list.go
// Componente de listagem de conexões com design centralizado e customizado.
// Exibe banner ASCII "SINAI" com shadow, montanha decorativa, cards com nome
// em destaque, driver, DSN e data de criação, ordenados alfabeticamente.
package tui

import (
	"fmt"
	"image/color"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stanley/sql-cli/internal/config"
)

// --- Arte ASCII: montanha e banner "SINAI" ---

// mountainArt é uma montanha pequena renderizada acima do banner.
// Cada linha está associada a uma cor de gradiente (pico → base).
var mountainArt = []struct {
	line  string
	color color.Color
}{
	{"                 /\\                 ", lipgloss.Color("#FFFFFF")},
	{"                /  \\  /\\            ", lipgloss.Color("#E0E6FF")},
	{"          /\\   /    \\/  \\           ", lipgloss.Color("#C8D4FF")},
	{"         /  \\ /          \\          ", lipgloss.Color("#B2C2EE")},
	{"        /    /            \\/\\       ", lipgloss.Color("#9BACD4")},
	{"       /    /                \\      ", lipgloss.Color("#8598BA")},
	{"      /    /                  \\     ", lipgloss.Color("#6B7DB3")},
	{"     /    /                    \\    ", lipgloss.Color("#556691")},
	{"    /    /                      \\   ", lipgloss.Color("#4A5980")},
	{"   /    /                        \\  ", lipgloss.Color("#3D4B6E")},
	{"  /    /                          \\ ", lipgloss.Color("#303E5C")},
	{" /____/____________________________\\", lipgloss.Color("#24314A")},
}

// sinaiArt é a arte ASCII do nome do projeto em fonte Big (FIGlet).
// S       I       N         A          I
var sinaiArt = []string{
	` ____    ___   _   _       _       ___ `,
	`/ ___|  |_ _| | \ | |     / \     |_ _|`,
	`\___ \   | |  |  \| |    / _ \     | | `,
	` ___) |  | |  | |\  |   / ___ \    | | `,
	`|____/  |___| |_| \_|  /_/   \_\  |___|`,
}

// --- Estilos ---

const listCardWidth = 64

var (
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Width(listCardWidth).
			Padding(0, 2)

	cardSelectedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Width(listCardWidth).
				Padding(0, 2)

	connNameStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Bold(true)

	connNameSelectedStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	driverBadgeStyle = func(driver string) lipgloss.Style {
		color := lipgloss.Color("#BB9AF7")
		switch strings.ToLower(driver) {
		case "sqlite":
			color = lipgloss.Color("#9ECE6A")
		case "postgres":
			color = lipgloss.Color("#7AA2F7")
		case "mysql":
			color = lipgloss.Color("#E0AF68")
		}
		return lipgloss.NewStyle().Foreground(color).Bold(true)
	}

	dsnStyle = lipgloss.NewStyle().Foreground(colorMuted).Italic(true)

	dateStyle = lipgloss.NewStyle().Foreground(colorMuted)

	listSubtitleStyle = lipgloss.NewStyle().Foreground(colorMuted)

	listHintStyle = lipgloss.NewStyle().Foreground(colorMuted).Italic(true)

	// Estilos do banner ASCII.
	bannerMainStyle   = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	bannerShadowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#2A2F4A"))
	bannerDivStyle    = lipgloss.NewStyle().Foreground(colorBorder)
)

// --- Banner: montanha + SINAI com shadow ---

// renderMountain renderiza a montanha ASCII com gradiente de cor pico→base.
func renderMountain() string {
	var lines []string
	for _, row := range mountainArt {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(row.color).
			Render(row.line))
	}
	return strings.Join(lines, "\n")
}

// renderShadowBanner renderiza o texto ASCII com efeito de shadow.
//
// O shadow é composto caracter por caracter:
//   - Posição [r][c] com char não-espaço → renderizado na cor principal.
//   - Posição [r][c] que é ' ', mas [r-1][c-2] não é ' ' → é sombra da linha
//     de cima, renderizada na cor de shadow.
//
// Isso cria um drop shadow deslocado 1 linha abaixo e 2 colunas à direita.
func renderShadowBanner(art []string) string {
	if len(art) == 0 {
		return ""
	}

	// Largura máxima + margem para o shadow.
	maxCol := 0
	for _, line := range art {
		if n := utf8.RuneCountInString(line); n > maxCol {
			maxCol = n
		}
	}
	totalCols := maxCol + 4 // margem para offset do shadow (+2 cols)
	totalRows := len(art) + 1 // +1 linha para sombra da última linha

	// Converte cada linha em slice de runes com padding de espaços.
	grid := make([][]rune, len(art))
	for i, line := range art {
		runes := []rune(line)
		row := make([]rune, totalCols)
		for j := range row {
			row[j] = ' '
		}
		copy(row, runes)
		grid[i] = row
	}

	// Monta o resultado linha por linha.
	var sb strings.Builder
	for r := 0; r < totalRows; r++ {
		for c := 0; c < totalCols; c++ {
			// Caracter principal na posição [r][c].
			var mc rune = ' '
			if r < len(grid) && c < len(grid[r]) {
				mc = grid[r][c]
			}

			// Shadow: o char [r-1][c-2] do grid principal projeta-se aqui.
			var sc rune = ' '
			sr, scc := r-1, c-2
			if sr >= 0 && sr < len(grid) && scc >= 0 && scc < len(grid[sr]) {
				sc = grid[sr][scc]
			}

			switch {
			case mc != ' ':
				sb.WriteString(bannerMainStyle.Render(string(mc)))
			case sc != ' ':
				sb.WriteString(bannerShadowStyle.Render(string(sc)))
			default:
				sb.WriteRune(' ')
			}
		}
		sb.WriteRune('\n')
	}
	return sb.String()
}

// renderBanner junta montanha + SINAI com shadow + divisor.
func renderBanner() string {
	mountain := renderMountain()

	divider := bannerDivStyle.Render(strings.Repeat("─", 44))

	return lipgloss.JoinVertical(
		lipgloss.Center,
		mountain,
		divider,
	)
}

// --- ConnectionListModel ---

// ConnectionListModel gerencia navegação e renderização da lista de conexões.
type ConnectionListModel struct {
	connections []config.Connection
	cursor      int
	width       int
	height      int
}

func NewConnectionList(conns []config.Connection) ConnectionListModel {
	return ConnectionListModel{connections: sortedConnections(conns)}
}

func (m ConnectionListModel) SetConnections(conns []config.Connection) ConnectionListModel {
	sorted := sortedConnections(conns)
	if m.cursor >= len(sorted) {
		m.cursor = max(0, len(sorted)-1)
	}
	m.connections = sorted
	return m
}

func (m ConnectionListModel) UpdateSize(w, h int) ConnectionListModel {
	m.width = w
	m.height = h
	return m
}

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

func (m ConnectionListModel) Selected() *config.Connection {
	if len(m.connections) == 0 || m.cursor >= len(m.connections) {
		return nil
	}
	c := m.connections[m.cursor]
	return &c
}

func (m ConnectionListModel) Len() int { return len(m.connections) }

// View renderiza a tela completa.
func (m ConnectionListModel) View() string {
	if m.width == 0 {
		return ""
	}

	var sections []string

	// Mostra banner apenas quando há espaço vertical suficiente.
	// Montanha (6) + sinai (6) + divider (1) + padding ≈ 14 linhas.
	if m.height > 22 {
		sections = append(sections, renderBanner())
	} else if m.height > 14 {
		// Sem montanha, só SINAI + divider.
		sinai := strings.TrimRight(renderShadowBanner(sinaiArt), "\n")
		div := bannerDivStyle.Render(strings.Repeat("─", 44))
		sections = append(sections, lipgloss.JoinVertical(lipgloss.Center, sinai, div))
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

// --- Renderização interna ---

func (m ConnectionListModel) renderSubtitle() string {
	n := len(m.connections)
	var label string
	switch n {
	case 0:
		label = "nenhuma conexão cadastrada"
	case 1:
		label = "1 conexão  ·  n nova  ·  e editar  ·  d deletar"
	default:
		label = fmt.Sprintf("%d conexões  ·  n nova  ·  e editar  ·  d deletar", n)
	}
	return "\n" + listSubtitleStyle.Render(label) + "\n"
}

func (m ConnectionListModel) renderEmpty() string {
	return lipgloss.NewStyle().
		Foreground(colorMuted).
		Italic(true).
		Width(listCardWidth + 4).
		Align(lipgloss.Center).
		Padding(2, 0).
		Render("Nenhuma conexão salva ainda.\nPressione  n  para criar a primeira.")
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
			Foreground(colorMuted).
			Width(listCardWidth+4).
			Align(lipgloss.Center).
			Render(fmt.Sprintf("∧  %d acima", start)))
	}

	for i := start; i < end; i++ {
		cards = append(cards, m.renderCard(m.connections[i], i == m.cursor))
	}

	below := len(m.connections) - end
	if below > 0 {
		cards = append(cards, lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(listCardWidth+4).
			Align(lipgloss.Center).
			Render(fmt.Sprintf("∨  %d abaixo", below)))
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
		dateStyle.Render("  Adicionado em "+formatDate(conn.CreatedAt)),
	))
}

func (m ConnectionListModel) renderHints() string {
	line := "↑↓ / j k  navegar   ·   enter  conectar   ·   n  nova   ·   e  editar   ·   d  deletar"
	return "\n" + listHintStyle.
		Width(listCardWidth+4).
		Align(lipgloss.Center).
		Render(line)
}

// --- Utilitários ---

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
		"jan", "fev", "mar", "abr", "mai", "jun",
		"jul", "ago", "set", "out", "nov", "dez",
	}
	return fmt.Sprintf("%02d %s %d", t.Day(), months[t.Month()-1], t.Year())
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
