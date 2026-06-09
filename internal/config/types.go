package config

import "time"

const (
	// ConfigDir is the directory relative to $HOME/.config where data is saved.
	ConfigDir = "sinai"
	// ConfigFile is the name of the JSON configuration file.
	ConfigFile = "config.json"
)

// Connection represents a saved database connection.
type Connection struct {
	// ID is the unique identifier automatically generated (UUID v4).
	ID string `json:"id"`
	// Name is the friendly display name shown in the UI (e.g., "Production", "Local Dev").
	Name string `json:"name"`
	// Driver is the database driver identifier (e.g., "sqlite", "postgres").
	Driver string `json:"driver"`
	// DSN is the Data Source Name — driver-specific connection string.
	// For SQLite: file path (e.g., "/home/user/db.sqlite")
	// For Postgres: standard connection string (e.g., "host=... user=... dbname=...")
	DSN string `json:"dsn"`
	// CreatedAt is the timestamp of when the connection was created.
	CreatedAt time.Time `json:"created_at"`
}

// AppConfig is the root structure of the configuration file.
type AppConfig struct {
	// Version is the configuration schema version, for future migrations.
	Version int `json:"version"`
	// Connections is the list of database connections saved by the user.
	Connections []Connection `json:"connections"`
	// LastConnectionID stores the ID of the last active connection,
	// used to automatically select it when opening the app.
	LastConnectionID string `json:"last_connection_id,omitempty"`
	// Path stores the configuration file path on disk (not serialized to JSON).
	Path string `json:"-"`
}
