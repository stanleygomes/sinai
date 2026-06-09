package menu

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/stanley/sql-cli/internal/config"
	"github.com/stanley/sql-cli/internal/database"
	"github.com/stanley/sql-cli/internal/tui/common"
)

// Init initializes input blinking.
func (m MenuViewModel) Init() tea.Cmd { return textinput.Blink }

// UpdateSize propagates window size to the connection list sub-component.
func (m MenuViewModel) UpdateSize(w, h int) (MenuViewModel, tea.Cmd) {
	m.width = w
	m.height = h
	m.connList = m.connList.UpdateSize(w, h)
	return m, nil
}

// Update processes events for the menu manager.
func (m MenuViewModel) Update(msg tea.Msg) (MenuViewModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ConnStatusMsg:
		m.statusMsg = msg.Msg
		m.state = stateBrowse
		m.connList = m.connList.SetConnections(m.cfg.Connections)
		return m, nil

	case tea.KeyPressMsg:
		switch m.state {
		case stateBrowse:
			switch {
			case key.Matches(msg, m.keys.Create):
				f := newConnForm()
				f.editID = ""
				m.form = f
				m.state = stateCreate
				return m, textinput.Blink

			case key.Matches(msg, m.keys.Edit):
				if sel := m.connList.Selected(); sel != nil {
					f := newConnForm()
					f.editID = sel.ID
					f.inputs[fieldName].SetValue(sel.Name)
					f.inputs[fieldDriver].SetValue(sel.Driver)
					f.inputs[fieldDSN].SetValue(sel.DSN)
					f.focus(fieldName)
					m.form = f
					m.state = stateEdit
					return m, textinput.Blink
				}

			case key.Matches(msg, m.keys.Delete):
				if m.connList.Selected() != nil {
					m.state = stateDelete
					return m, nil
				}

			case key.Matches(msg, m.keys.Connect):
				if sel := m.connList.Selected(); sel != nil {
					return m, m.connectCmd(*sel)
				}
			}

			var cmd tea.Cmd
			m.connList, cmd = m.connList.Update(msg)
			cmds = append(cmds, cmd)

		case stateCreate, stateEdit:
			switch {
			case key.Matches(msg, m.keys.Cancel):
				m.state = stateBrowse
				return m, nil

			case key.Matches(msg, m.keys.TabNext):
				m.form.nextField()
				return m, textinput.Blink

			case key.Matches(msg, m.keys.TabPrev):
				m.form.prevField()
				return m, textinput.Blink

			case key.Matches(msg, m.keys.Confirm):
				if m.state == stateCreate {
					return m, m.submitCreateCmd()
				}
				return m, m.submitEditCmd()
			}

			var cmd tea.Cmd
			m.form.inputs[m.form.focusIndex], cmd = m.form.inputs[m.form.focusIndex].Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)

		case stateDelete:
			switch msg.String() {
			case "y", "Y", "s", "S":
				return m, m.deleteCmd()
			default:
				m.state = stateBrowse
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m MenuViewModel) connectCmd(conn config.Connection) tea.Cmd {
	return func() tea.Msg {
		db, err := database.Open(conn.Driver, conn.DSN)
		if err != nil {
			return common.ErrMsg{Err: fmt.Errorf("failed to connect: %w", err)}
		}
		return common.ConnectedMsg{DB: db, Conn: conn}
	}
}

func (m MenuViewModel) submitCreateCmd() tea.Cmd {
	if !m.form.isValid() {
		return nil
	}
	cfg := m.cfg
	name := strings.TrimSpace(m.form.inputs[fieldName].Value())
	driver := strings.TrimSpace(m.form.inputs[fieldDriver].Value())
	dsn := strings.TrimSpace(m.form.inputs[fieldDSN].Value())

	return func() tea.Msg {
		conn, err := config.AddConnection(cfg, name, driver, dsn)
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return ConnStatusMsg{Msg: fmt.Sprintf("Connection %q created.", conn.Name)}
	}
}

func (m MenuViewModel) submitEditCmd() tea.Cmd {
	if !m.form.isValid() {
		return nil
	}
	cfg := m.cfg
	id := m.form.editID
	name := strings.TrimSpace(m.form.inputs[fieldName].Value())
	driver := strings.TrimSpace(m.form.inputs[fieldDriver].Value())
	dsn := strings.TrimSpace(m.form.inputs[fieldDSN].Value())

	return func() tea.Msg {
		if err := config.UpdateConnection(cfg, id, name, driver, dsn); err != nil {
			return common.ErrMsg{Err: err}
		}
		return ConnStatusMsg{Msg: fmt.Sprintf("Connection %q updated.", name)}
	}
}

func (m MenuViewModel) deleteCmd() tea.Cmd {
	sel := m.connList.Selected()
	if sel == nil {
		return nil
	}
	cfg := m.cfg
	id := sel.ID
	name := sel.Name

	return func() tea.Msg {
		if err := config.DeleteConnection(cfg, id); err != nil {
			return common.ErrMsg{Err: err}
		}
		return ConnStatusMsg{Msg: fmt.Sprintf("Connection %q removed.", name)}
	}
}

// ConnStatusMsg is emitted by CRUD operations to update MenuViewModel's state.
type ConnStatusMsg struct{ Msg string }
