package config

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AddConnection adds a new connection to the configuration and saves it.
// Automatically generates a unique ID and the creation timestamp.
func AddConnection(cfg *AppConfig, name, driver, dsn string) (Connection, error) {
	conn := Connection{
		ID:        uuid.NewString(),
		Name:      name,
		Driver:    driver,
		DSN:       dsn,
		CreatedAt: time.Now(),
	}
	cfg.Connections = append(cfg.Connections, conn)
	return conn, Save(cfg)
}

// UpdateConnection updates the details of an existing connection by ID.
func UpdateConnection(cfg *AppConfig, id, name, driver, dsn string) error {
	for i, c := range cfg.Connections {
		if c.ID == id {
			cfg.Connections[i].Name = name
			cfg.Connections[i].Driver = driver
			cfg.Connections[i].DSN = dsn
			return Save(cfg)
		}
	}
	return fmt.Errorf("connection with ID %q not found", id)
}

// DeleteConnection removes a connection by ID and saves the configuration.
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
	return fmt.Errorf("connection with ID %q not found", id)
}

// FindConnection returns a pointer to the connection with the given ID, or nil.
func FindConnection(cfg *AppConfig, id string) *Connection {
	for i := range cfg.Connections {
		if cfg.Connections[i].ID == id {
			return &cfg.Connections[i]
		}
	}
	return nil
}
