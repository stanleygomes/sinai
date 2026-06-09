package config

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

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

// FindConnection retorna um ponteiro para a conexão com o ID informado, ou nil.
func FindConnection(cfg *AppConfig, id string) *Connection {
	for i := range cfg.Connections {
		if cfg.Connections[i].ID == id {
			return &cfg.Connections[i]
		}
	}
	return nil
}
