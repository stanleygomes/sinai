// Package env manages loading local environment variables from .env files.
package env

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Load reads the .env file in the current directory and injects its variables into the process environment
// if they are not already set in the operating system environment.
func Load() {
	file, err := os.Open(".env")
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if isIgnoredLine(line) {
			continue
		}

		key, val, ok := parseLine(line)
		if !ok {
			continue
		}

		if key != "" && os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading .env file: %v\n", err)
	}
}

// isIgnoredLine checks if the line is empty or a comment.
func isIgnoredLine(line string) bool {
	return line == "" || strings.HasPrefix(line, "#")
}

// parseLine splits a key=value line and returns the clean key, clean value, and true if the line is valid.
func parseLine(line string) (string, string, bool) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := strings.TrimSpace(parts[0])
	val := strings.TrimSpace(parts[1])
	return key, cleanQuotes(val), true
}

// cleanQuotes removes single or double quotes around the value, if they exist.
func cleanQuotes(val string) string {
	if (strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"")) ||
		(strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'")) {
		return val[1 : len(val)-1]
	}
	return val
}
