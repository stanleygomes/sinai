// Package tui implementa a interface de terminal da aplicação usando Bubble Tea.
// O model raiz (RootModel) gerencia a navegação entre telas e o estado global.
package tui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stanley/sql-cli/internal/config"
	"github.com/stanley/sql-cli/internal/database"
)

// --- Mensagens personalizadas do Bubble Tea ---

// ConnectedMsg é enviada quando uma conexão com o banco é estabelecida com sucesso.
type ConnectedMsg struct {
	DB   database.DB
	Conn config.Connection
}

// ErrMsg encapsula um erro ocorrido em qualquer ponto da aplicação.
type ErrMsg struct{ Err error }

func (e ErrMsg) Error() string { return e.Err.Error() }

// NavigateMsg solicita a troca de tela no RootModel.
type NavigateMsg struct{ Screen Screen }

// Screen identifica qual tela está sendo exibida.
type Screen int

const (
	ScreenMenu Screen = iota // Tela de listagem/gerenciamento de conexões.
	ScreenMain               // Workspace principal (editor + tabelas).
)

// --- Estilos globais (lipgloss) ---

// Paleta de cores da aplicação (dark-themed; AdaptiveColor removed in lipgloss v2).
var (
	colorBackground  = lipgloss.Color("#1A1B26")
	colorPrimary     = lipgloss.Color("#7AA2F7") // Azul Tokyo Night
	colorAccent      = lipgloss.Color("#BB9AF7") // Roxo
	colorSuccess     = lipgloss.Color("#9ECE6A") // Verde
	colorWarning     = lipgloss.Color("#E0AF68") // Laranja
	colorDanger      = lipgloss.Color("#F7768E") // Vermelho
	colorMuted       = lipgloss.Color("#565F89")
	colorBorder      = lipgloss.Color("#3B4261")
	colorBorderFocus = lipgloss.Color("#7AA2F7")
	colorText        = lipgloss.Color("#CDD6F4")
)

// Estilos de painel reutilizáveis.
var (
	// PanelStyle é o estilo base para painéis com borda.
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	// PanelFocusStyle é o estilo do painel atualmente focado.
	PanelFocusStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorderFocus).
			Padding(0, 1)

	// StatusBarStyle é o estilo da barra de status inferior.
	StatusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#16161E")).
			Foreground(colorMuted).
			Padding(0, 1)

	// TitleStyle é o estilo do título da aplicação no header.
	TitleStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			Padding(0, 1)

	// HelpStyle é o estilo das dicas de atalhos de teclado.
	HelpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	// ErrorStyle é o estilo de mensagens de erro.
	ErrorStyle = lipgloss.NewStyle().
			Foreground(colorDanger).
			Bold(true)

	// SuccessStyle é o estilo de mensagens de sucesso.
	SuccessStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)
)

// --- Mapa de atalhos globais ---

// KeyMap define os atalhos de teclado disponíveis em toda a aplicação.
type KeyMap struct {
	Quit       key.Binding
	Back       key.Binding
	Help       key.Binding
	FocusLeft  key.Binding
	FocusRight key.Binding
}

// DefaultKeyMap retorna o mapa de atalhos padrão.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "ctrl+q"),
			key.WithHelp("ctrl+c/q", "sair"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "voltar"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "ajuda"),
		),
		FocusLeft: key.NewBinding(
			key.WithKeys("ctrl+left", "ctrl+h"),
			key.WithHelp("ctrl+←", "foco esquerda"),
		),
		FocusRight: key.NewBinding(
			key.WithKeys("ctrl+right", "ctrl+l"),
			key.WithHelp("ctrl+→", "foco direita"),
		),
	}
}

// --- RootModel ---

// RootModel é o modelo raiz do Bubble Tea.
// Ele gerencia qual tela está ativa, dimensões do terminal,
// estado global (config, conexão ativa) e roteia mensagens
// para o sub-modelo correto.
type RootModel struct {
	// keys contém os atalhos de teclado globais.
	keys KeyMap
	// cfg é a configuração da aplicação carregada do disco.
	cfg *config.AppConfig
	// configPath é o caminho do arquivo de configuração (exibido na UI).
	configPath string

	// currentScreen indica qual tela está sendo renderizada.
	currentScreen Screen

	// activeDB é a conexão de banco atualmente aberta (nil se não houver).
	activeDB database.DB
	// activeConn é a metadata da conexão ativa.
	activeConn *config.Connection

	// menuView é o sub-modelo da tela de menu.
	menuView MenuViewModel
	// mainView é o sub-modelo da tela principal (workspace).
	mainView MainViewModel

	// width e height são as dimensões atuais do terminal.
	width  int
	height int

	// statusMsg é a mensagem exibida na barra de status.
	statusMsg string
	// statusIsError indica se statusMsg é uma mensagem de erro.
	statusIsError bool
}

// New cria um RootModel inicializado.
// Recebe a configuração já carregada e o caminho do arquivo de config.
func New(cfg *config.AppConfig, configPath string) RootModel {
	keys := DefaultKeyMap()
	return RootModel{
		keys:          keys,
		cfg:           cfg,
		configPath:    configPath,
		currentScreen: ScreenMenu,
		menuView:      NewMenuView(cfg, configPath),
		mainView:      NewMainView(),
	}
}

// Init implementa tea.Model. Inicia comandos iniciais se necessário.
func (m RootModel) Init() tea.Cmd {
	return m.menuView.Init()
}

// Update implementa tea.Model. Roteia mensagens para o sub-modelo correto.
func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	// Atualização de dimensões do terminal.
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Propaga o resize para os sub-modelos.
		var cmd tea.Cmd
		m.menuView, cmd = m.menuView.UpdateSize(msg.Width, m.contentHeight())
		cmds = append(cmds, cmd)
		m.mainView, cmd = m.mainView.UpdateSize(msg.Width, m.contentHeight())
		cmds = append(cmds, cmd)

	// Atalhos globais (processados antes dos sub-modelos).
	// In bubbletea v2, use tea.KeyPressMsg for key press events.
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			if m.activeDB != nil {
				_ = m.activeDB.Close()
			}
			return m, tea.Quit

		case key.Matches(msg, m.keys.Back):
			if m.currentScreen == ScreenMain {
				// Ao voltar do workspace, fecha a conexão ativa.
				if m.activeDB != nil {
					_ = m.activeDB.Close()
					m.activeDB = nil
					m.activeConn = nil
				}
				m.currentScreen = ScreenMenu
				m.setStatus("Desconectado.", false)
				return m, nil
			}
		}

	// Navegação entre telas.
	case NavigateMsg:
		m.currentScreen = msg.Screen
		return m, nil

	// Conexão estabelecida com sucesso.
	case ConnectedMsg:
		m.activeDB = msg.DB
		m.activeConn = &msg.Conn
		m.currentScreen = ScreenMain
		m.setStatus(fmt.Sprintf("Conectado: %s (%s)", msg.Conn.Name, msg.Conn.Driver), false)
		// Inicializa o workspace com a nova conexão.
		var cmd tea.Cmd
		m.mainView, cmd = m.mainView.SetConnection(msg.DB, msg.Conn)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	// Tratamento de erros globais.
	case ErrMsg:
		m.setStatus("Erro: "+msg.Err.Error(), true)
		return m, nil
	}

	// Delega o Update para o sub-modelo da tela ativa.
	var cmd tea.Cmd
	switch m.currentScreen {
	case ScreenMenu:
		m.menuView, cmd = m.menuView.Update(msg)
		// Propaga atualizações de config para o modelo raiz.
		m.cfg = m.menuView.Config()
	case ScreenMain:
		m.mainView, cmd = m.mainView.Update(msg)
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View implementa tea.Model. Compõe o layout final da tela.
func (m RootModel) View() tea.View {
	if m.width == 0 {
		return tea.NewView("Inicializando...")
	}

	// Compõe: header + conteúdo da tela ativa + barra de status.
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

// renderHeader renderiza o cabeçalho superior da aplicação.
func (m RootModel) renderHeader() string {
	title := TitleStyle.Render("◈ sql-cli")

	var connInfo string
	if m.activeConn != nil {
		connInfo = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Render(fmt.Sprintf("  ●  %s  [%s]", m.activeConn.Name, m.activeConn.Driver))
	}

	// Preenche o restante da linha com espaços para alinhar à direita.
	hint := HelpStyle.Render("ctrl+q sair  |  ? ajuda")
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
		BorderForeground(colorBorder).
		Render(header)
}

// renderCurrentScreen renderiza a tela ativa com a altura disponível.
func (m RootModel) renderCurrentScreen() string {
	switch m.currentScreen {
	case ScreenMenu:
		return m.menuView.View()
	case ScreenMain:
		return m.mainView.View()
	default:
		return "Tela desconhecida"
	}
}

// renderStatusBar renderiza a barra de status inferior.
func (m RootModel) renderStatusBar() string {
	var msg string
	if m.statusMsg != "" {
		if m.statusIsError {
			msg = ErrorStyle.Render("✗ " + m.statusMsg)
		} else {
			msg = SuccessStyle.Render("✓ " + m.statusMsg)
		}
	}

	// Exibe sempre o caminho do arquivo de configuração no canto direito.
	configInfo := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render("config: " + m.configPath)

	spacer := lipgloss.NewStyle().
		Width(m.width - lipgloss.Width(msg) - lipgloss.Width(configInfo) - 2).
		Render("")

	bar := lipgloss.JoinHorizontal(lipgloss.Center, msg, spacer, configInfo)
	return StatusBarStyle.Width(m.width).Render(bar)
}

// contentHeight calcula a altura disponível para o conteúdo,
// descontando o header (2 linhas) e a status bar (1 linha).
func (m RootModel) contentHeight() int {
	reserved := 3 // header (2) + statusbar (1)
	h := m.height - reserved
	if h < 0 {
		return 0
	}
	return h
}

// setStatus define a mensagem da barra de status.
func (m *RootModel) setStatus(msg string, isError bool) {
	m.statusMsg = msg
	m.statusIsError = isError
}
