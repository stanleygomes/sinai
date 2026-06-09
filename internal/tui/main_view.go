// Package tui — main_view.go
// Implementa o Workspace principal (Tela 2): split de tela horizontal com
// painel de tabelas à esquerda (25%) e painel de editor/resultados à direita (75%).
// O painel direito usa abas: Editor SQL | Visualizador de Dados | DDL Viewer.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stanley/sql-cli/internal/config"
	"github.com/stanley/sql-cli/internal/database"
)

// --- Constantes e tipos da view principal ---

// Tab identifica a aba ativa no painel direito.
type Tab int

const (
	TabEditor Tab = iota // Aba 0: Editor SQL.
	TabData              // Aba 1: Visualizador de Dados.
	TabDDL               // Aba 2: DDL Viewer.
)

// Focus identifica qual painel está com foco de teclado.
type Focus int

const (
	FocusLeft  Focus = iota // Foco no painel de tabelas.
	FocusRight              // Foco no painel editor/resultados.
)

// --- Mensagens internas da view principal ---

// queryResultMsg carrega o resultado de uma query executada em background.
type queryResultMsg struct {
	result *database.QueryResult
	err    error
}

// tablesLoadedMsg carrega a lista de tabelas do banco.
type tablesLoadedMsg struct {
	tables []string
	err    error
}

// ddlLoadedMsg carrega o DDL de uma tabela específica.
type ddlLoadedMsg struct {
	tableName string
	ddl       string
	err       error
}

// --- MainViewModel ---

// MainViewModel é o sub-modelo do workspace principal.
type MainViewModel struct {
	// Conexão ativa com o banco de dados.
	db         database.DB
	activeConn config.Connection

	// Layout.
	width  int
	height int
	focus  Focus

	// Painel esquerdo: lista de tabelas.
	tables      []string
	tablesCursor int
	tablesReady bool

	// Painel direito: abas.
	activeTab Tab

	// Aba 0: Editor SQL.
	editor    textarea.Model
	isRunning bool // true enquanto uma query está sendo executada.

	// Aba 1: Visualizador de Dados.
	dataTable     table.Model
	queryDuration time.Duration
	rowCount      int

	// Aba 2: DDL Viewer.
	ddlViewport viewport.Model
	ddlContent  string

	// Mensagem de status/erro da view.
	statusMsg     string
	statusIsError bool

	keys mainKeyMap
}

// mainKeyMap atalhos do workspace principal.
type mainKeyMap struct {
	RunQuery  key.Binding
	NextTab   key.Binding
	PrevTab   key.Binding
	FocusLeft key.Binding
	FocusRight key.Binding
	TableDown key.Binding
	TableUp   key.Binding
	ViewDDL   key.Binding
}

func defaultMainKeys() mainKeyMap {
	return mainKeyMap{
		RunQuery: key.NewBinding(
			key.WithKeys("f5"),
			key.WithHelp("F5", "executar query"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("ctrl+right", "ctrl+l"),
			key.WithHelp("ctrl+→", "próxima aba"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("ctrl+left", "ctrl+h"),
			key.WithHelp("ctrl+←", "aba anterior"),
		),
		FocusLeft: key.NewBinding(
			key.WithKeys("ctrl+shift+left"),
			key.WithHelp("ctrl+shift+←", "foco painel esquerdo"),
		),
		FocusRight: key.NewBinding(
			key.WithKeys("ctrl+shift+right"),
			key.WithHelp("ctrl+shift+→", "foco painel direito"),
		),
		TableDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "próxima tabela"),
		),
		TableUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "tabela anterior"),
		),
		ViewDDL: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "ver DDL"),
		),
	}
}

// NewMainView cria um MainViewModel inicial (sem conexão ativa).
func NewMainView() MainViewModel {
	// Editor SQL.
	ed := textarea.New()
	ed.Placeholder = "Digite sua query SQL aqui...\nF5 para executar."
	ed.ShowLineNumbers = true
	ed.CharLimit = 0 // Sem limite.
	ed.SetWidth(60)
	ed.SetHeight(15)

	// Viewport para o DDL Viewer.
	vp := viewport.New(viewport.WithWidth(60), viewport.WithHeight(15))
	vp.Style = lipgloss.NewStyle().Padding(1, 2)

	// Tabela de dados (inicialmente vazia).
	t := table.New(
		table.WithFocused(false),
		table.WithHeight(15),
	)
	ts := table.DefaultStyles()
	ts.Header = ts.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorBorder).
		BorderBottom(true).
		Bold(true).
		Foreground(colorPrimary)
	ts.Selected = ts.Selected.
		Foreground(colorText).
		Background(lipgloss.Color("#2E3250")).
		Bold(false)
	t.SetStyles(ts)

	return MainViewModel{
		focus:       FocusRight,
		activeTab:   TabEditor,
		editor:      ed,
		dataTable:   t,
		ddlViewport: vp,
		keys:        defaultMainKeys(),
	}
}

// SetConnection configura a conexão ativa e dispara o carregamento das tabelas.
func (m MainViewModel) SetConnection(db database.DB, conn config.Connection) (MainViewModel, tea.Cmd) {
	m.db = db
	m.activeConn = conn
	m.tablesReady = false
	m.tables = nil
	m.tablesCursor = 0
	m.statusMsg = ""
	m.editor.Focus()
	return m, m.loadTablesCmd()
}

// UpdateSize atualiza as dimensões do workspace e dos componentes internos.
func (m MainViewModel) UpdateSize(w, h int) (MainViewModel, tea.Cmd) {
	m.width = w
	m.height = h

	_, rightW := m.panelWidths()
	contentH := h - 5 // Desconta cabeçalho de abas + bordas.

	// Redimensiona o editor.
	m.editor.SetWidth(rightW - 6)
	m.editor.SetHeight(contentH - 2)

	// Redimensiona o viewport do DDL.
	m.ddlViewport.SetWidth(rightW - 4)
	m.ddlViewport.SetHeight(contentH - 2)

	// Redimensiona a tabela de dados.
	m.dataTable.SetHeight(contentH - 3)

	return m, nil
}

// Update processa mensagens no workspace.
func (m MainViewModel) Update(msg tea.Msg) (MainViewModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {

		// Alternar foco entre painéis com Tab.
		case msg.String() == "tab":
			if m.focus == FocusLeft {
				m.focus = FocusRight
				m.editor.Focus()
			} else {
				m.focus = FocusLeft
				m.editor.Blur()
			}
			return m, nil

		// Navegar nas abas (foco direito).
		case key.Matches(msg, m.keys.NextTab) && m.focus == FocusRight:
			m.activeTab = (m.activeTab + 1) % 3
			return m, nil

		case key.Matches(msg, m.keys.PrevTab) && m.focus == FocusRight:
			m.activeTab = (m.activeTab - 1 + 3) % 3
			return m, nil

		// Executar query (F5).
		case key.Matches(msg, m.keys.RunQuery):
			if !m.isRunning && m.db != nil {
				query := m.getQueryToRun()
				if strings.TrimSpace(query) != "" {
					m.isRunning = true
					m.statusMsg = "Executando..."
					m.statusIsError = false
					return m, m.runQueryCmd(query)
				}
			}
			return m, nil

		// Navegar na lista de tabelas (foco esquerdo).
		case key.Matches(msg, m.keys.TableDown) && m.focus == FocusLeft:
			if m.tablesCursor < len(m.tables)-1 {
				m.tablesCursor++
			}
			return m, nil

		case key.Matches(msg, m.keys.TableUp) && m.focus == FocusLeft:
			if m.tablesCursor > 0 {
				m.tablesCursor--
			}
			return m, nil

		// Ver DDL da tabela selecionada (foco esquerdo).
		case key.Matches(msg, m.keys.ViewDDL) && m.focus == FocusLeft:
			if len(m.tables) > 0 {
				tableName := m.tables[m.tablesCursor]
				m.activeTab = TabDDL
				m.focus = FocusRight
				return m, m.loadDDLCmd(tableName)
			}

		// Enter no painel esquerdo: preenche SELECT na aba editor.
		case msg.String() == "enter" && m.focus == FocusLeft:
			if len(m.tables) > 0 {
				tableName := m.tables[m.tablesCursor]
				query := fmt.Sprintf("SELECT * FROM %s LIMIT 100;", tableName)
				m.editor.SetValue(query)
				m.activeTab = TabEditor
				m.focus = FocusRight
				m.editor.Focus()
			}
			return m, nil
		}

	// Resultado de query retornado do background.
	case queryResultMsg:
		m.isRunning = false
		if msg.err != nil {
			m.statusMsg = msg.err.Error()
			m.statusIsError = true
			return m, nil
		}
		m.queryDuration = msg.result.Duration
		m.rowCount = len(msg.result.Rows)

		if len(msg.result.Columns) > 0 {
			// Popula a tabela de dados e navega para a aba de resultados.
			m.updateDataTable(msg.result)
			m.activeTab = TabData
			m.statusMsg = fmt.Sprintf("%d linha(s) · %s", m.rowCount, m.queryDuration.Round(time.Millisecond))
		} else {
			m.statusMsg = fmt.Sprintf("OK · %d linha(s) afetada(s) · %s",
				msg.result.RowsAffected, msg.result.Duration.Round(time.Millisecond))
		}
		m.statusIsError = false
		return m, nil

	// Lista de tabelas carregada.
	case tablesLoadedMsg:
		if msg.err != nil {
			m.statusMsg = "Erro ao carregar tabelas: " + msg.err.Error()
			m.statusIsError = true
			return m, nil
		}
		m.tables = msg.tables
		m.tablesReady = true
		return m, nil

	// DDL carregado.
	case ddlLoadedMsg:
		if msg.err != nil {
			m.statusMsg = "Erro ao carregar DDL: " + msg.err.Error()
			m.statusIsError = true
			return m, nil
		}
		m.ddlContent = msg.ddl
		m.ddlViewport.SetContent(
			lipgloss.NewStyle().Foreground(colorText).Render(msg.ddl),
		)
		return m, nil
	}

	// Propaga mensagens para os componentes ativos.
	if m.focus == FocusRight {
		switch m.activeTab {
		case TabEditor:
			var cmd tea.Cmd
			m.editor, cmd = m.editor.Update(msg)
			cmds = append(cmds, cmd)
		case TabData:
			var cmd tea.Cmd
			m.dataTable, cmd = m.dataTable.Update(msg)
			cmds = append(cmds, cmd)
		case TabDDL:
			var cmd tea.Cmd
			m.ddlViewport, cmd = m.ddlViewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View renderiza o workspace completo: painel esquerdo + painel direito.
func (m MainViewModel) View() string {
	if m.width == 0 {
		return ""
	}

	leftW, rightW := m.panelWidths()

	leftPanel := m.renderLeftPanel(leftW)
	rightPanel := m.renderRightPanel(rightW)

	workspace := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	// Barra de status interna da view.
	statusLine := m.renderInternalStatus()

	return lipgloss.JoinVertical(lipgloss.Left, workspace, statusLine)
}

// --- Renderização dos painéis ---

// renderLeftPanel renderiza o painel de tabelas (25% da largura).
func (m MainViewModel) renderLeftPanel(w int) string {
	style := PanelStyle.Width(w - 2)
	if m.focus == FocusLeft {
		style = PanelFocusStyle.Width(w - 2)
	}

	header := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		MarginBottom(1).
		Render("◧ Tabelas")

	var body string
	if !m.tablesReady {
		body = HelpStyle.Render("Carregando...")
	} else if len(m.tables) == 0 {
		body = HelpStyle.Render("Nenhuma tabela encontrada.")
	} else {
		var rows strings.Builder
		for i, t := range m.tables {
			prefix := "  "
			lineStyle := lipgloss.NewStyle().Foreground(colorText)
			if i == m.tablesCursor {
				prefix = "▶ "
				lineStyle = lineStyle.Foreground(colorPrimary).Bold(true)
			}
			rows.WriteString(lineStyle.Render(prefix+t) + "\n")
		}
		body = rows.String()
	}

	hint := "\n" + HelpStyle.Render("enter: SELECT  ctrl+d: DDL  tab: trocar foco")
	content := lipgloss.JoinVertical(lipgloss.Left, header, body, hint)

	return style.Height(m.height - 4).Render(content)
}

// renderRightPanel renderiza o painel de editor/resultados com abas (75%).
func (m MainViewModel) renderRightPanel(w int) string {
	style := PanelStyle.Width(w - 2)
	if m.focus == FocusRight {
		style = PanelFocusStyle.Width(w - 2)
	}

	tabs := m.renderTabs(w - 4)
	content := m.renderActiveTab(w - 6)

	inner := lipgloss.JoinVertical(lipgloss.Left, tabs, content)
	return style.Height(m.height - 4).Render(inner)
}

// renderTabs renderiza a barra de abas superior do painel direito.
func (m MainViewModel) renderTabs(w int) string {
	tabLabels := []string{"① Editor SQL", "② Dados", "③ DDL"}
	var rendered []string

	for i, label := range tabLabels {
		if Tab(i) == m.activeTab {
			rendered = append(rendered, lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 2).
				Render(label))
		} else {
			rendered = append(rendered, lipgloss.NewStyle().
				Foreground(colorMuted).
				Padding(0, 2).
				Render(label))
		}
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Bottom, rendered...)
	hint := HelpStyle.Render("ctrl+←/→: trocar aba")

	spacer := lipgloss.NewStyle().
		Width(w - lipgloss.Width(tabBar) - lipgloss.Width(hint)).
		Render("")

	return lipgloss.JoinHorizontal(lipgloss.Center, tabBar, spacer, hint)
}

// renderActiveTab renderiza o conteúdo da aba ativa.
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

// renderEditorTab renderiza o editor SQL.
func (m MainViewModel) renderEditorTab() string {
	var b strings.Builder

	hint := HelpStyle.Render("F5: executar  |  selecione texto para executar trecho")
	if m.isRunning {
		hint = lipgloss.NewStyle().Foreground(colorWarning).Render("⟳ Executando query...")
	}

	b.WriteString(hint + "\n")
	b.WriteString(m.editor.View())
	return b.String()
}

// renderDataTab renderiza a tabela de resultados.
func (m MainViewModel) renderDataTab() string {
	if m.rowCount == 0 && m.queryDuration == 0 {
		return HelpStyle.Render("\nExecute uma query SELECT para ver os resultados aqui.\n")
	}
	header := SuccessStyle.Render(fmt.Sprintf("%d linha(s) · %s", m.rowCount, m.queryDuration.Round(time.Millisecond)))
	return lipgloss.JoinVertical(lipgloss.Left, header, m.dataTable.View())
}

// renderDDLTab renderiza o DDL viewer.
func (m MainViewModel) renderDDLTab() string {
	if m.ddlContent == "" {
		return HelpStyle.Render("\nSelecione uma tabela no painel esquerdo e pressione ctrl+d.\n")
	}
	return m.ddlViewport.View()
}

// renderInternalStatus renderiza a linha de status da view principal.
func (m MainViewModel) renderInternalStatus() string {
	if m.statusMsg == "" {
		return HelpStyle.Width(m.width).Render("  tab: trocar painel  |  ctrl+←/→: abas")
	}
	if m.statusIsError {
		return ErrorStyle.Width(m.width).Render("  ✗ " + m.statusMsg)
	}
	return SuccessStyle.Width(m.width).Render("  ✓ " + m.statusMsg)
}

// --- Cálculo do layout ---

// panelWidths calcula as larguras dos painéis esquerdo e direito.
// Esquerdo: ~25% | Direito: ~75%
func (m MainViewModel) panelWidths() (left, right int) {
	left = m.width / 4
	if left < 20 {
		left = 20
	}
	right = m.width - left
	return
}

// --- Helpers de componentes ---

// getQueryToRun retorna a query a executar.
// Se houver texto selecionado no editor, retorna apenas a seleção;
// caso contrário, retorna todo o conteúdo do editor.
// Nota: o componente textarea do bubbles não expõe seleção de texto nativamente.
// A lógica de seleção pode ser implementada futuramente com um editor customizado.
// Por ora, retorna sempre o conteúdo completo.
func (m MainViewModel) getQueryToRun() string {
	return m.editor.Value()
}

// updateDataTable popula o componente de tabela com os resultados da query.
func (m *MainViewModel) updateDataTable(result *database.QueryResult) {
	// Define colunas dinamicamente.
	cols := make([]table.Column, len(result.Columns))
	for i, col := range result.Columns {
		// Largura mínima baseada no nome da coluna, máximo de 30 chars.
		w := len(col.Name) + 2
		if w > 30 {
			w = 30
		}
		cols[i] = table.Column{Title: col.Name, Width: w}
	}

	// Define linhas.
	rows := make([]table.Row, len(result.Rows))
	for i, r := range result.Rows {
		// Trunca valores longos para exibição na tabela.
		cells := make([]string, len(r))
		for j, cell := range r {
			if len(cell) > 30 {
				cells[j] = cell[:27] + "..."
			} else {
				cells[j] = cell
			}
		}
		rows[i] = table.Row(cells)
	}

	m.dataTable.SetColumns(cols)
	m.dataTable.SetRows(rows)
	m.dataTable.SetCursor(0)

	// Reajusta largura das colunas baseado nos dados.
	for i, col := range cols {
		maxW := col.Width
		for _, row := range rows {
			if i < len(row) && len(row[i]) > maxW {
				maxW = len(row[i])
			}
		}
		if maxW > 40 {
			maxW = 40
		}
		cols[i].Width = maxW
	}
	m.dataTable.SetColumns(cols)
}

// --- Comandos em background ---

// runQueryCmd executa a query de forma assíncrona.
func (m MainViewModel) runQueryCmd(query string) tea.Cmd {
	db := m.db
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		result, err := db.Query(ctx, query)
		return queryResultMsg{result: result, err: err}
	}
}

// loadTablesCmd carrega a lista de tabelas do banco em background.
func (m MainViewModel) loadTablesCmd() tea.Cmd {
	db := m.db
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		tables, err := db.Tables(ctx)
		return tablesLoadedMsg{tables: tables, err: err}
	}
}

// loadDDLCmd carrega o DDL de uma tabela em background.
func (m MainViewModel) loadDDLCmd(tableName string) tea.Cmd {
	db := m.db
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ddl, err := db.TableDDL(ctx, tableName)
		return ddlLoadedMsg{tableName: tableName, ddl: ddl, err: err}
	}
}
