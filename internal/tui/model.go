// Package tui implements the terminal user interface using Bubble Tea.
package tui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stanley/sql-cli/internal/config"
	"github.com/stanley/sql-cli/internal/database"
	"github.com/stanley/sql-cli/internal/tui/common"
	"github.com/stanley/sql-cli/internal/tui/views/menu"
	"github.com/stanley/sql-cli/internal/tui/views/workspace"
)

// KeyMap defines the global keyboard shortcuts.
type KeyMap struct {
	Quit       key.Binding
	Back       key.Binding
	Help       key.Binding
	FocusLeft  key.Binding
	FocusRight key.Binding
}

// DefaultKeyMap returns the default keyboard shortcuts.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "ctrl+q"),
			key.WithHelp("ctrl+c/q", "quit"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		FocusLeft: key.NewBinding(
			key.WithKeys("ctrl+left", "ctrl+h"),
			key.WithHelp("ctrl+←", "focus left"),
		),
		FocusRight: key.NewBinding(
			key.WithKeys("ctrl+right", "ctrl+l"),
			key.WithHelp("ctrl+→", "focus right"),
		),
	}
}

// RootModel is the primary TUI container orchestrating navigation and global states.
type RootModel struct {
	keys       KeyMap
	cfg        *config.AppConfig
	configPath string

	currentScreen common.Screen

	activeDB   database.DB
	activeConn *config.Connection

	menuView      menu.MenuViewModel
	workspaceView workspace.MainViewModel

	width  int
	height int

	statusMsg     string
	statusIsError bool
}

// New creates a new RootModel.
func New(cfg *config.AppConfig, configPath string) RootModel {
	keys := DefaultKeyMap()
	return RootModel{
		keys:          keys,
		cfg:           cfg,
		configPath:    configPath,
		currentScreen: common.ScreenMenu,
		menuView:      menu.NewMenuView(cfg, configPath),
		workspaceView: workspace.NewMainView(),
	}
}

// Init initializes the sub-components.
func (m RootModel) Init() tea.Cmd {
	return m.menuView.Init()
}

// Update routes messages to the active view and processes global shortcuts.
func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		var cmd tea.Cmd
		m.menuView, cmd = m.menuView.UpdateSize(msg.Width, m.contentHeight())
		cmds = append(cmds, cmd)
		m.workspaceView, cmd = m.workspaceView.UpdateSize(msg.Width, m.contentHeight())
		cmds = append(cmds, cmd)

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			if m.activeDB != nil {
				_ = m.activeDB.Close()
			}
			return m, tea.Quit

		case key.Matches(msg, m.keys.Back):
			if m.currentScreen == common.ScreenWorkspace {
				if m.activeDB != nil {
					_ = m.activeDB.Close()
					m.activeDB = nil
					m.activeConn = nil
				}
				m.currentScreen = common.ScreenMenu
				m.setStatus("Disconnected.", false)
				return m, nil
			}
		}

	case common.NavigateMsg:
		m.currentScreen = msg.Screen
		return m, nil

	case common.ConnectedMsg:
		m.activeDB = msg.DB
		m.activeConn = &msg.Conn
		m.currentScreen = common.ScreenWorkspace
		m.setStatus(fmt.Sprintf("Connected: %s (%s)", msg.Conn.Name, msg.Conn.Driver), false)

		var cmd tea.Cmd
		m.workspaceView, cmd = m.workspaceView.SetConnection(msg.DB, msg.Conn)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case common.ErrMsg:
		m.setStatus("Error: "+msg.Err.Error(), true)
		return m, nil
	}

	var cmd tea.Cmd
	switch m.currentScreen {
	case common.ScreenMenu:
		m.menuView, cmd = m.menuView.Update(msg)
		m.cfg = m.menuView.Config()
	case common.ScreenWorkspace:
		m.workspaceView, cmd = m.workspaceView.Update(msg)
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View compiles and renders header, active screen, and status bar.
func (m RootModel) View() tea.View {
	if m.width == 0 {
		return tea.NewView("Initializing...")
	}

	header := m.renderHeader()
	content := m.renderCurrentScreen()
	statusBar := m.renderStatusBar()

	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		statusBar,
	))
}

func (m RootModel) renderHeader() string {
	title := common.TitleStyle.Render("◈ sinai")

	var connInfo string
	if m.activeConn != nil {
		connInfo = lipgloss.NewStyle().
			Foreground(common.ColorSuccess).
			Render(fmt.Sprintf("  ●  %s  [%s]", m.activeConn.Name, m.activeConn.Driver))
	}

	hint := common.HelpStyle.Render("ctrl+q quit  |  ? help")
	spacer := lipgloss.NewStyle().
		Width(m.width - lipgloss.Width(title+connInfo) - lipgloss.Width(hint) - 2).
		Render("")

	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		title, connInfo, spacer, hint,
	)

	return lipgloss.NewStyle().
		Width(m.width).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(common.ColorBorder).
		Render(header)
}

func (m RootModel) renderCurrentScreen() string {
	switch m.currentScreen {
	case common.ScreenMenu:
		return m.menuView.View()
	case common.ScreenWorkspace:
		return m.workspaceView.View()
	default:
		return "Unknown screen"
	}
}

func (m RootModel) renderStatusBar() string {
	var msg string
	if m.statusMsg != "" {
		if m.statusIsError {
			msg = common.ErrorStyle.Render("✗ " + m.statusMsg)
		} else {
			msg = common.SuccessStyle.Render("✓ " + m.statusMsg)
		}
	}

	configInfo := lipgloss.NewStyle().
		Foreground(common.ColorMuted).
		Render("config: " + m.configPath)

	spacer := lipgloss.NewStyle().
		Width(m.width - lipgloss.Width(msg) - lipgloss.Width(configInfo) - 2).
		Render("")

	bar := lipgloss.JoinHorizontal(lipgloss.Center, msg, spacer, configInfo)
	return common.StatusBarStyle.Width(m.width).Render(bar)
}

func (m RootModel) contentHeight() int {
	reserved := 3
	h := m.height - reserved
	if h < 0 {
		return 0
	}
	return h
}

func (m *RootModel) setStatus(msg string, isError bool) {
	m.statusMsg = msg
	m.statusIsError = isError
}
