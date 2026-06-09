// Package menu implements the connection manager screen.
package menu

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"github.com/stanley/sql-cli/internal/config"
	"github.com/stanley/sql-cli/internal/tui/common"
)

type menuState int

const (
	stateBrowse menuState = iota
	stateCreate
	stateEdit
	stateDelete
)

type formField int

const (
	fieldName formField = iota
	fieldDriver
	fieldDSN
	fieldCount
)

type connForm struct {
	inputs     [fieldCount]textinput.Model
	focusIndex formField
	editID     string // Empty means new connection
}

func newConnForm() connForm {
	f := connForm{}

	nameInput := textinput.New()
	nameInput.Placeholder = "e.g., Local Dev, Production..."
	nameInput.CharLimit = 64
	nameInput.Prompt = ""

	driverInput := textinput.New()
	driverInput.Placeholder = "sqlite | postgres"
	driverInput.CharLimit = 32
	driverInput.SetValue("sqlite")
	driverInput.Prompt = ""

	dsnInput := textinput.New()
	dsnInput.Placeholder = "/path/to/database.db  or  host=... user=... dbname=..."
	dsnInput.CharLimit = 256
	dsnInput.Prompt = ""

	f.inputs[fieldName] = nameInput
	f.inputs[fieldDriver] = driverInput
	f.inputs[fieldDSN] = dsnInput
	f.focus(fieldName)
	return f
}

func (f *connForm) focus(field formField) {
	f.focusIndex = field
	for i := range f.inputs {
		if formField(i) == field {
			f.inputs[i].Focus()
			s := f.inputs[i].Styles()
			s.Focused.Prompt = lipgloss.NewStyle().Foreground(common.ColorPrimary)
			s.Focused.Text = lipgloss.NewStyle().Foreground(common.ColorText)
			f.inputs[i].SetStyles(s)
		} else {
			f.inputs[i].Blur()
		}
	}
}

func (f *connForm) nextField() { f.focus((f.focusIndex + 1) % fieldCount) }
func (f *connForm) prevField() { f.focus((f.focusIndex - 1 + fieldCount) % fieldCount) }

func (f *connForm) isValid() bool {
	return strings.TrimSpace(f.inputs[fieldName].Value()) != "" &&
		strings.TrimSpace(f.inputs[fieldDSN].Value()) != ""
}

type menuKeyMap struct {
	Create  key.Binding
	Edit    key.Binding
	Delete  key.Binding
	Connect key.Binding
	Confirm key.Binding
	Cancel  key.Binding
	TabNext key.Binding
	TabPrev key.Binding
}

func defaultMenuKeys() menuKeyMap {
	return menuKeyMap{
		Create:  key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
		Edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Delete:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Connect: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "connect")),
		Confirm: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
		Cancel:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		TabNext: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field")),
		TabPrev: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "previous field")),
	}
}

// MenuViewModel manages connection list browsing, creation, editing, and deletion.
type MenuViewModel struct {
	cfg        *config.AppConfig
	configPath string
	state      menuState

	// connList is the custom connection list display sub-component.
	connList ConnectionListModel

	form      connForm
	width     int
	height    int
	statusMsg string
	keys      menuKeyMap
}

// NewMenuView creates a new MenuViewModel.
func NewMenuView(cfg *config.AppConfig, configPath string) MenuViewModel {
	return MenuViewModel{
		cfg:        cfg,
		configPath: configPath,
		state:      stateBrowse,
		connList:   NewConnectionList(cfg.Connections),
		form:       newConnForm(),
		keys:       defaultMenuKeys(),
	}
}

// Config returns the root configuration.
func (m MenuViewModel) Config() *config.AppConfig { return m.cfg }
