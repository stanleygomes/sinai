// Package common defines shared types, messages, and styling tokens for the TUI.
package common

import (
	"github.com/stanley/sql-cli/internal/config"
	"github.com/stanley/sql-cli/internal/database"
)

// Screen represents the active TUI screen.
type Screen int

const (
	// ScreenMenu is the connection manager screen.
	ScreenMenu Screen = iota
	// ScreenWorkspace is the main SQL editor and database tables screen.
	ScreenWorkspace
)

// ConnectedMsg is sent when a database connection is successfully established.
type ConnectedMsg struct {
	DB   database.DB
	Conn config.Connection
}

// ErrMsg wraps any error occurring during application execution.
type ErrMsg struct {
	Err error
}

// Error returns the string representation of the error.
func (e ErrMsg) Error() string { return e.Err.Error() }

// NavigateMsg requests a screen change in the RootModel.
type NavigateMsg struct {
	Screen Screen
}
