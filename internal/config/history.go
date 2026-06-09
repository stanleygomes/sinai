package config

import (
	"fmt"

	"github.com/google/uuid"
)

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
