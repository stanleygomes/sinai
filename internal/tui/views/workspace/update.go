package workspace

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stanley/sql-cli/internal/config"
	"github.com/stanley/sql-cli/internal/database"
	"github.com/stanley/sql-cli/internal/tui/common"
)

// SetConnection activates a database connection and triggers tables loading.
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

// UpdateSize recalculates panel sizes on window resize.
func (m MainViewModel) UpdateSize(w, h int) (MainViewModel, tea.Cmd) {
	m.width = w
	m.height = h

	_, rightW := m.panelWidths()
	contentH := h - 5

	m.editor.SetWidth(rightW - 6)
	m.editor.SetHeight(contentH - 2)

	m.ddlViewport.SetWidth(rightW - 4)
	m.ddlViewport.SetHeight(contentH - 2)

	m.dataTable.SetHeight(contentH - 3)

	return m, nil
}

// Update handles active view updates and keyboard shortcuts.
func (m MainViewModel) Update(msg tea.Msg) (MainViewModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case msg.String() == "tab":
			if m.focus == FocusLeft {
				m.focus = FocusRight
				m.editor.Focus()
			} else {
				m.focus = FocusLeft
				m.editor.Blur()
			}
			return m, nil

		case key.Matches(msg, m.keys.NextTab) && m.focus == FocusRight:
			m.activeTab = (m.activeTab + 1) % 3
			return m, nil

		case key.Matches(msg, m.keys.PrevTab) && m.focus == FocusRight:
			m.activeTab = (m.activeTab - 1 + 3) % 3
			return m, nil

		case key.Matches(msg, m.keys.RunQuery):
			if !m.isRunning && m.db != nil {
				query := m.getQueryToRun()
				if strings.TrimSpace(query) != "" {
					m.isRunning = true
					m.statusMsg = "Running query..."
					m.statusIsError = false
					return m, m.runQueryCmd(query)
				}
			}
			return m, nil

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

		case key.Matches(msg, m.keys.ViewDDL) && m.focus == FocusLeft:
			if len(m.tables) > 0 {
				tableName := m.tables[m.tablesCursor]
				m.activeTab = TabDDL
				m.focus = FocusRight
				return m, m.loadDDLCmd(tableName)
			}

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
			m.updateDataTable(msg.result)
			m.activeTab = TabData
			m.statusMsg = fmt.Sprintf("%d row(s) · %s", m.rowCount, m.queryDuration.Round(time.Millisecond))
		} else {
			m.statusMsg = fmt.Sprintf("OK · %d row(s) affected · %s",
				msg.result.RowsAffected, msg.result.Duration.Round(time.Millisecond))
		}
		m.statusIsError = false
		return m, nil

	case tablesLoadedMsg:
		if msg.err != nil {
			m.statusMsg = "Failed to load tables: " + msg.err.Error()
			m.statusIsError = true
			return m, nil
		}
		m.tables = msg.tables
		m.tablesReady = true
		return m, nil

	case ddlLoadedMsg:
		if msg.err != nil {
			m.statusMsg = "Failed to load DDL: " + msg.err.Error()
			m.statusIsError = true
			return m, nil
		}
		m.ddlContent = msg.ddl
		m.ddlViewport.SetContent(
			lipgloss.NewStyle().Foreground(common.ColorText).Render(msg.ddl),
		)
		return m, nil
	}

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

func (m MainViewModel) getQueryToRun() string {
	return m.editor.Value()
}

func (m *MainViewModel) updateDataTable(result *database.QueryResult) {
	cols := make([]table.Column, len(result.Columns))
	for i, col := range result.Columns {
		w := len(col.Name) + 2
		if w > 30 {
			w = 30
		}
		cols[i] = table.Column{Title: col.Name, Width: w}
	}

	rows := make([]table.Row, len(result.Rows))
	for i, r := range result.Rows {
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

func (m MainViewModel) runQueryCmd(query string) tea.Cmd {
	db := m.db
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		result, err := db.Query(ctx, query)
		return queryResultMsg{result: result, err: err}
	}
}

func (m MainViewModel) loadTablesCmd() tea.Cmd {
	db := m.db
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		tables, err := db.Tables(ctx)
		return tablesLoadedMsg{tables: tables, err: err}
	}
}

func (m MainViewModel) loadDDLCmd(tableName string) tea.Cmd {
	db := m.db
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ddl, err := db.TableDDL(ctx, tableName)
		return ddlLoadedMsg{tableName: tableName, ddl: ddl, err: err}
	}
}
