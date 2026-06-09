package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/stanley/sql-cli/internal/config"
	"github.com/stanley/sql-cli/internal/tui"
)

// main is the entry point of the application.
//
// It performs the following tasks:
// - Loads the saved configuration from disk.
// - Initializes the TUI root model with the configuration and path.
// - Starts the Bubble Tea event loop to run the terminal user interface.
func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	rootModel := tui.New(cfg, cfg.Path)
	p := tea.NewProgram(rootModel)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
