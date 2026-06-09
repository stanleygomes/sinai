// Package tui — menu_view.go
// Orquestra a tela de gerenciamento de conexões (Tela 1).
// Delega a renderização da lista ao ConnectionListModel (connection_list.go)
// e gerencia os formulários de criação/edição e o modal de exclusão.
package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stanley/sql-cli/internal/config"
	"github.com/stanley/sql-cli/internal/database"
)

// menuState controla o estado interno da tela de menu.
type menuState int

const (
	menuStateBrowse menuState = iota // Navegando na lista de conexões.
	menuStateCreate                  // Formulário de criação de conexão.
	menuStateEdit                    // Formulário de edição de conexão.
	menuStateDelete                  // Confirmação de exclusão.
)

// --- Formulário de conexão ---

// formField identifica os campos do formulário.
type formField int

const (
	fieldName   formField = iota
	fieldDriver           // "sqlite", "postgres", ...
	fieldDSN              // caminho do arquivo ou connection string
	fieldCount            // sentinela: número total de campos
)

// connForm agrupa os textinputs do formulário de conexão.
type connForm struct {
	inputs     [fieldCount]textinput.Model
	focusIndex formField
	editID     string // ID da conexão sendo editada (vazio = nova criação)
}

// newConnForm inicializa o formulário com placeholders e estilos.
func newConnForm() connForm {
	f := connForm{}

	nameInput := textinput.New()
	nameInput.Placeholder = "Ex: Dev Local, Produção..."
	nameInput.CharLimit = 64
	nameInput.Prompt = ""

	driverInput := textinput.New()
	driverInput.Placeholder = "sqlite | postgres"
	driverInput.CharLimit = 32
	driverInput.SetValue("sqlite")
	driverInput.Prompt = ""

	dsnInput := textinput.New()
	dsnInput.Placeholder = "/caminho/para/banco.db  ou  host=... user=... dbname=..."
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
			s.Focused.Prompt = lipgloss.NewStyle().Foreground(colorPrimary)
			s.Focused.Text = lipgloss.NewStyle().Foreground(colorText)
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

// --- Mapa de atalhos ---

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
		Create:  key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "nova")),
		Edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "editar")),
		Delete:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "deletar")),
		Connect: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "conectar")),
		Confirm: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirmar")),
		Cancel:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancelar")),
		TabNext: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "próximo campo")),
		TabPrev: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "campo anterior")),
	}
}

// --- MenuViewModel ---

// MenuViewModel é o sub-modelo da tela de menu.
// Orquestra as transições entre browse, criação, edição e exclusão.
type MenuViewModel struct {
	cfg        *config.AppConfig
	configPath string
	state      menuState

	// connList é o componente de listagem com design customizado.
	connList ConnectionListModel

	form      connForm
	width     int
	height    int
	statusMsg string
	keys      menuKeyMap
}

// NewMenuView cria o MenuViewModel com a configuração já carregada.
func NewMenuView(cfg *config.AppConfig, configPath string) MenuViewModel {
	return MenuViewModel{
		cfg:        cfg,
		configPath: configPath,
		state:      menuStateBrowse,
		connList:   NewConnectionList(cfg.Connections),
		form:       newConnForm(),
		keys:       defaultMenuKeys(),
	}
}

// Config expõe o ponteiro de configuração para o RootModel sincronizar.
func (m MenuViewModel) Config() *config.AppConfig { return m.cfg }

// Init arranca o cursor piscante dos textinputs.
func (m MenuViewModel) Init() tea.Cmd { return textinput.Blink }

// UpdateSize propaga dimensões para os sub-componentes.
func (m MenuViewModel) UpdateSize(w, h int) (MenuViewModel, tea.Cmd) {
	m.width = w
	m.height = h
	m.connList = m.connList.UpdateSize(w, h)
	return m, nil
}

// Update processa mensagens na tela de menu.
func (m MenuViewModel) Update(msg tea.Msg) (MenuViewModel, tea.Cmd) {
	var cmds []tea.Cmd

	// --- Handlers de mensagens assíncronas (retornadas por tea.Cmd) ---
	switch msg := msg.(type) {

	// Resultado de operações CRUD de conexão.
	case ConnStatusMsg:
		m.statusMsg = msg.Msg
		m.state = menuStateBrowse
		m.connList = m.connList.SetConnections(m.cfg.Connections)
		return m, nil

	case tea.KeyPressMsg:
		switch m.state {

		// ── Estado: navegando na lista ──────────────────────────────────
		case menuStateBrowse:
			switch {
			case key.Matches(msg, m.keys.Create):
				f := newConnForm()
				f.editID = ""
				m.form = f
				m.state = menuStateCreate
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
					m.state = menuStateEdit
					return m, textinput.Blink
				}

			case key.Matches(msg, m.keys.Delete):
				if m.connList.Selected() != nil {
					m.state = menuStateDelete
					return m, nil
				}

			case key.Matches(msg, m.keys.Connect):
				if sel := m.connList.Selected(); sel != nil {
					return m, m.connectCmd(*sel)
				}
			}

			// Delega navegação (↑↓ j k) ao componente de lista.
			var cmd tea.Cmd
			m.connList, cmd = m.connList.Update(msg)
			cmds = append(cmds, cmd)

		// ── Estado: formulário de criação / edição ──────────────────────
		case menuStateCreate, menuStateEdit:
			switch {
			case key.Matches(msg, m.keys.Cancel):
				m.state = menuStateBrowse
				return m, nil

			case key.Matches(msg, m.keys.TabNext):
				m.form.nextField()
				return m, textinput.Blink

			case key.Matches(msg, m.keys.TabPrev):
				m.form.prevField()
				return m, textinput.Blink

			case key.Matches(msg, m.keys.Confirm):
				if m.state == menuStateCreate {
					return m, m.submitCreateCmd()
				}
				return m, m.submitEditCmd()
			}

			// Propaga a tecla para o input com foco.
			var cmd tea.Cmd
			m.form.inputs[m.form.focusIndex], cmd = m.form.inputs[m.form.focusIndex].Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)

		// ── Estado: confirmação de exclusão ─────────────────────────────
		case menuStateDelete:
			switch msg.String() {
			case "y", "Y", "s", "S":
				return m, m.deleteCmd()
			default:
				m.state = menuStateBrowse
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// View renderiza a tela de menu conforme o estado atual.
func (m MenuViewModel) View() string {
	switch m.state {
	case menuStateCreate:
		return m.renderForm("  Nova Conexão")
	case menuStateEdit:
		return m.renderForm("  Editar Conexão")
	case menuStateDelete:
		return m.renderDeleteConfirm()
	default:
		return m.connList.View()
	}
}

// --- Renderização dos formulários ---

func (m MenuViewModel) renderForm(title string) string {
	labels := []string{"Nome da Conexão", "Driver", "DSN / Caminho"}

	var rows strings.Builder
	for i, inp := range m.form.inputs {
		label := lipgloss.NewStyle().
			Foreground(colorMuted).
			Render(labels[i])

		focused := formField(i) == m.form.focusIndex
		borderColor := colorBorder
		if focused {
			borderColor = colorBorderFocus
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
		validMsg = "\n" + ErrorStyle.Render("  Nome e DSN são obrigatórios.")
	}

	hint := HelpStyle.Render("tab · shift+tab  navegar campos    enter  salvar    esc  cancelar")

	inner := lipgloss.JoinVertical(
		lipgloss.Left,
		TitleStyle.Render(title),
		"",
		rows.String(),
		hint,
		validMsg,
	)

	formBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
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

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		ErrorStyle.Render("⚠  Confirmar exclusão"),
		"",
		lipgloss.NewStyle().Foreground(colorText).Render(
			fmt.Sprintf("Deseja remover %q?", sel.Name),
		),
		"",
		HelpStyle.Render("s / y  confirmar    qualquer outra tecla  cancelar"),
	)

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorDanger).
		Padding(1, 4).
		Width(50).
		Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

// --- Comandos (tea.Cmd) ---

// connectCmd abre a conexão com o banco em background.
func (m MenuViewModel) connectCmd(conn config.Connection) tea.Cmd {
	return func() tea.Msg {
		db, err := database.Open(conn.Driver, conn.DSN)
		if err != nil {
			return ErrMsg{Err: fmt.Errorf("falha ao conectar: %w", err)}
		}
		return ConnectedMsg{DB: db, Conn: conn}
	}
}

// submitCreateCmd persiste a nova conexão e emite ConnStatusMsg.
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
			return ErrMsg{Err: err}
		}
		return ConnStatusMsg{Msg: fmt.Sprintf("Conexão %q criada.", conn.Name)}
	}
}

// submitEditCmd persiste as alterações e emite ConnStatusMsg.
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
			return ErrMsg{Err: err}
		}
		return ConnStatusMsg{Msg: fmt.Sprintf("Conexão %q atualizada.", name)}
	}
}

// deleteCmd remove a conexão selecionada e emite ConnStatusMsg.
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
			return ErrMsg{Err: err}
		}
		return ConnStatusMsg{Msg: fmt.Sprintf("Conexão %q removida.", name)}
	}
}

// ConnStatusMsg é emitida por operações CRUD para atualizar o estado do modelo.
type ConnStatusMsg struct{ Msg string }
