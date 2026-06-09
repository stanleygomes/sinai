package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/stanley/sql-cli/internal/config"
	"github.com/stanley/sql-cli/internal/tui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao carregar configuração: %v\n", err)
		os.Exit(1)
	}

	rootModel := tui.New(cfg, cfg.Path)
	p := tea.NewProgram(rootModel)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Erro na execução da TUI: %v\n", err)
		os.Exit(1)
	}
}
