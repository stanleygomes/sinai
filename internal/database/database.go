// Package database defines the interface for database access and
// provides implementations for supported database drivers.
package database

import (
	"context"
	"fmt"
	"strings"
)

// DB is the interface that every database implementation must satisfy.
type DB interface {
	// Query executes an SQL statement and returns the query result.
	// Works for both query (SELECT) and exec (DML) statements.
	Query(ctx context.Context, sql string) (*QueryResult, error)

	// Tables returns the names of all tables in the current/default schema.
	Tables(ctx context.Context) ([]string, error)

	// TableDDL returns the DDL schema statement used to create a table (CREATE TABLE ...).
	TableDDL(ctx context.Context, tableName string) (string, error)

	// Ping checks if the database connection is active and healthy.
	Ping(ctx context.Context) error

	// Close terminates the connection to the database.
	Close() error

	// DriverName returns the name of the database driver (e.g., "sqlite", "postgres").
	DriverName() string
}

// Open is the generic entry point that instantiates the correct DB client
// based on the driver name.
func Open(driver, dsn string) (DB, error) {
	switch strings.ToLower(driver) {
	case "sqlite":
		return OpenSQLite(dsn)
	default:
		return nil, fmt.Errorf("driver %q not supported", driver)
	}
}
