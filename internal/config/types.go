package config

import "time"

const (
	// ConfigDir é o diretório relativo a $HOME/.config onde os dados são salvos.
	ConfigDir = "sinai"
	// ConfigFile é o nome do arquivo de configuração JSON.
	ConfigFile = "config.json"
)

// HistoryEntry representa uma entrada no histórico de queries de uma conexão.
type HistoryEntry struct {
	// ID único da entrada no histórico.
	ID string `json:"id"`
	// Query é o texto SQL executado.
	Query string `json:"query"`
	// ExecutedAt é o timestamp de quando a query foi executada.
	ExecutedAt time.Time `json:"executed_at"`
	// DurationMs é o tempo de execução em milissegundos.
	DurationMs int64 `json:"duration_ms"`
	// Error armazena a mensagem de erro, caso a query tenha falhado.
	Error string `json:"error,omitempty"`
}

// Connection representa uma conexão de banco de dados salva.
type Connection struct {
	// ID é o identificador único gerado automaticamente (UUID v4).
	ID string `json:"id"`
	// Name é o nome amigável exibido na interface (ex: "Produção", "Dev Local").
	Name string `json:"name"`
	// Driver é o identificador do driver de banco (ex: "sqlite", "postgres").
	Driver string `json:"driver"`
	// DSN é a Data Source Name — string de conexão específica do driver.
	// Para SQLite: caminho do arquivo (ex: "/home/user/db.sqlite")
	// Para Postgres: string de conexão padrão (ex: "host=... user=... dbname=...")
	DSN string `json:"dsn"`
	// CreatedAt é o timestamp de quando a conexão foi criada.
	CreatedAt time.Time `json:"created_at"`
	// History contém as últimas queries executadas nesta conexão.
	// É mantido em ordem cronológica decrescente (mais recente primeiro).
	History []HistoryEntry `json:"history,omitempty"`
}

// AppConfig é a estrutura raiz do arquivo de configuração.
type AppConfig struct {
	// Version é a versão do schema de configuração, para migrações futuras.
	Version int `json:"version"`
	// Connections é a lista de conexões salvas pelo usuário.
	Connections []Connection `json:"connections"`
	// LastConnectionID guarda o ID da última conexão usada,
	// para selecioná-la automaticamente ao abrir o app.
	LastConnectionID string `json:"last_connection_id,omitempty"`
	// Path armazena o caminho do arquivo de configuração no disco (não persistido no JSON).
	Path string `json:"-"`
}
