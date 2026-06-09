// Package config manages the persistence of application settings,
// including saved connections, in JSON format.
// The configuration file is stored in $HOME/.config/sinai/config.json,
// following the XDG Base Directory specification.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ConfigPath returns the absolute path of the configuration file.
// Follows the XDG specification: $XDG_CONFIG_HOME/sinai/config.json or
// $HOME/.config/sinai/config.json as fallback.
func ConfigPath() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not determine home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}

	return filepath.Join(base, ConfigDir, ConfigFile), nil
}

// Load loads the configuration from the JSON file.
// If the file does not exist, it returns a valid, empty AppConfig.
func Load() (*AppConfig, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// First run: return default configuration.
		return &AppConfig{Version: 1, Connections: []Connection{}, Path: path}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to decode %s: %w", path, err)
	}
	cfg.Path = path

	return &cfg, nil
}

// Save persists the configuration to the JSON file, creating the directory if needed.
func Save(cfg *AppConfig) error {
	path := cfg.Path
	if path == "" {
		var err error
		path, err = ConfigPath()
		if err != nil {
			return err
		}
	}

	// Ensure the configuration directory exists.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize configuration: %w", err)
	}

	// Write atomically via temporary file to prevent corruption.
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
}
