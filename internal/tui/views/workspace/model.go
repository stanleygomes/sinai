// Package workspace implements the main query editor and database viewer screen.
package workspace

import (
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/stanley/sql-cli/internal/config"
	"github.com/stanley/sql-cli/internal/database"
	"github.com/stanley/sql-cli/internal/tui/common"
)

// Tab represents the active tab in the workspace panel.
type Tab int

const (
	// TabEditor is the tab containing the SQL text area editor.
	TabEditor Tab = iota
	// TabData is the tab containing query result data grids.
	TabData
	// TabDDL is the tab showing schema table definitions.
	TabDDL
)

// Focus indicates which panel has the keyboard input focus.
type Focus int

const (
	// FocusLeft means keyboard focus is on the database tables list.
	FocusLeft Focus = iota
	// FocusRight means keyboard focus is on the tabbed workspace views.
	FocusRight
)

type queryResultMsg struct {
	result *database.QueryResult
	err    error
}

type tablesLoadedMsg struct {
	tables []string
	err    error
}

type ddlLoadedMsg struct {
	tableName string
	ddl       string
	err       error
}

type workspaceKeyMap struct {
	RunQuery   key.Binding
	NextTab    key.Binding
	PrevTab    key.Binding
	FocusLeft  key.Binding
	FocusRight key.Binding
	TableDown  key.Binding
	TableUp    key.Binding
	ViewDDL    key.Binding
}

func defaultWorkspaceKeys() workspaceKeyMap {
	return workspaceKeyMap{
		RunQuery: key.NewBinding(
			key.WithKeys("f5"),
			key.WithHelp("F5", "run query"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("ctrl+right", "ctrl+l"),
			key.WithHelp("ctrl+→", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("ctrl+left", "ctrl+h"),
			key.WithHelp("ctrl+←", "prev tab"),
		),
		FocusLeft: key.NewBinding(
			key.WithKeys("ctrl+shift+left"),
			key.WithHelp("ctrl+shift+←", "focus left panel"),
		),
		FocusRight: key.NewBinding(
			key.WithKeys("ctrl+shift+right"),
			key.WithHelp("ctrl+shift+→", "focus right panel"),
		),
		TableDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "next table"),
		),
		TableUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "prev table"),
		),
		ViewDDL: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "view DDL"),
		),
	}
}

// MainViewModel manages the main workspace layout (tables list, SQL editor, data grid, DDL viewer).
type MainViewModel struct {
	db         database.DB
	activeConn config.Connection

	width  int
	height int
	focus  Focus

	tables       []string
	tablesCursor int
	tablesReady  bool

	activeTab Tab

	editor    textarea.Model
	isRunning bool

	dataTable     table.Model
	queryDuration time.Duration
	rowCount      int

	ddlViewport viewport.Model
	ddlContent  string

	statusMsg     string
	statusIsError bool

	keys workspaceKeyMap
}

// NewMainView creates a new MainViewModel.
func NewMainView() MainViewModel {
	ed := textarea.New()
	ed.Placeholder = "Enter your SQL query here...\nPress F5 to execute."
	ed.ShowLineNumbers = true
	ed.CharLimit = 0
	ed.SetWidth(60)
	ed.SetHeight(15)

	vp := viewport.New(viewport.WithWidth(60), viewport.WithHeight(15))
	vp.Style = lipgloss.NewStyle().Padding(1, 2)

	t := table.New(
		table.WithFocused(false),
		table.WithHeight(15),
	)
	ts := table.DefaultStyles()
	ts.Header = ts.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(common.ColorBorder).
		BorderBottom(true).
		Bold(true).
		Foreground(common.ColorPrimary)
	ts.Selected = ts.Selected.
		Foreground(common.ColorText).
		Background(lipgloss.Color("#2E3250")).
		Bold(false)
	t.SetStyles(ts)

	return MainViewModel{
		focus:       FocusRight,
		activeTab:   TabEditor,
		editor:      ed,
		dataTable:   t,
		ddlViewport: vp,
		keys:        defaultWorkspaceKeys(),
	}
}
