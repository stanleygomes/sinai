// Package config gerencia a persistência das configurações da aplicação,
// incluindo conexões salvas e histórico de queries, no formato JSON.
// O arquivo de configuração é armazenado em $HOME/.config/sql/config.json,
// seguindo a especificação XDG Base Directory.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

const (
	// ConfigDir é o diretório relativo a $HOME/.config onde os dados são salvos.
	ConfigDir = "sql"
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

// ConfigPath retorna o caminho absoluto do arquivo de configuração.
// Segue a especificação XDG: $XDG_CONFIG_HOME/sql/config.json ou
// $HOME/.config/sql/config.json como fallback.
func ConfigPath() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("não foi possível determinar o diretório home: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, ConfigDir, ConfigFile), nil
}

// Load carrega a configuração do arquivo JSON.
// Se o arquivo não existir, retorna uma AppConfig vazia e válida.
func Load() (*AppConfig, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// Primeira execução: retorna configuração padrão.
		return &AppConfig{Version: 1, Connections: []Connection{}, Path: path}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao ler %s: %w", path, err)
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("erro ao decodificar %s: %w", path, err)
	}
	cfg.Path = path

	return &cfg, nil
}

// Save persiste a configuração no arquivo JSON, criando o diretório se necessário.
func Save(cfg *AppConfig) error {
	path := cfg.Path
	if path == "" {
		var err error
		path, err = ConfigPath()
		if err != nil {
			return err
		}
	}

	// Garante que o diretório de configuração existe.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("erro ao criar diretório %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao serializar configuração: %w", err)
	}

	// Escreve atomicamente via arquivo temporário para evitar corrupção.
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("erro ao escrever arquivo temporário: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("erro ao salvar configuração: %w", err)
	}

	return nil
}

// AddConnection adiciona uma nova conexão à configuração e salva.
// Gera automaticamente um ID único e o timestamp de criação.
func AddConnection(cfg *AppConfig, name, driver, dsn string) (Connection, error) {
	conn := Connection{
		ID:        uuid.NewString(),
		Name:      name,
		Driver:    driver,
		DSN:       dsn,
		CreatedAt: time.Now(),
		History:   []HistoryEntry{},
	}
	cfg.Connections = append(cfg.Connections, conn)
	return conn, Save(cfg)
}

// UpdateConnection atualiza os dados de uma conexão existente pelo ID.
func UpdateConnection(cfg *AppConfig, id, name, driver, dsn string) error {
	for i, c := range cfg.Connections {
		if c.ID == id {
			cfg.Connections[i].Name = name
			cfg.Connections[i].Driver = driver
			cfg.Connections[i].DSN = dsn
			return Save(cfg)
		}
	}
	return fmt.Errorf("conexão com ID %q não encontrada", id)
}

// DeleteConnection remove uma conexão pelo ID e salva.
func DeleteConnection(cfg *AppConfig, id string) error {
	for i, c := range cfg.Connections {
		if c.ID == id {
			cfg.Connections = append(cfg.Connections[:i], cfg.Connections[i+1:]...)
			if cfg.LastConnectionID == id {
				cfg.LastConnectionID = ""
			}
			return Save(cfg)
		}
	}
	return fmt.Errorf("conexão com ID %q não encontrada", id)
}

// AddHistory adiciona uma entrada ao histórico de uma conexão.
// Mantém apenas as últimas maxHistory entradas para evitar crescimento ilimitado.
func AddHistory(cfg *AppConfig, connID string, entry HistoryEntry, maxHistory int) error {
	if maxHistory <= 0 {
		maxHistory = 100 // Padrão razoável.
	}
	entry.ID = uuid.NewString()

	for i, c := range cfg.Connections {
		if c.ID == connID {
			// Insere no início (mais recente primeiro).
			cfg.Connections[i].History = append([]HistoryEntry{entry}, c.History...)
			// Trunca se ultrapassar o limite.
			if len(cfg.Connections[i].History) > maxHistory {
				cfg.Connections[i].History = cfg.Connections[i].History[:maxHistory]
			}
			return Save(cfg)
		}
	}
	return fmt.Errorf("conexão com ID %q não encontrada", connID)
}

// FindConnection retorna um ponteiro para a conexão com o ID informado, ou nil.
func FindConnection(cfg *AppConfig, id string) *Connection {
	for i := range cfg.Connections {
		if cfg.Connections[i].ID == id {
			return &cfg.Connections[i]
		}
	}
	return nil
}
