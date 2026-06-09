// Package database defines the interface for database access and
// provides implementations for supported database drivers.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	// Pure Go SQLite driver (no CGO required).
	_ "modernc.org/sqlite"
)

// sqliteClient implements the DB interface for modernc/sqlite driver.
type sqliteClient struct {
	db  *sql.DB
	dsn string
}

// OpenSQLite opens (or creates) a SQLite database at the specified DSN path.
// The DSN should be the file path (e.g., "/home/user/data.db") or
// ":memory:" for an in-memory database.
func OpenSQLite(dsn string) (DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite (%s): %w", dsn, err)
	}

	// SQLite does not support concurrent write operations across connections.
	db.SetMaxOpenConns(1)

	// Enable WAL journal mode and foreign keys enforcement.
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA foreign_keys=ON;",
		"PRAGMA busy_timeout=5000;",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to apply pragma %q: %w", pragma, err)
		}
	}

	return &sqliteClient{db: db, dsn: dsn}, nil
}

// DriverName returns the driver identifier.
func (c *sqliteClient) DriverName() string { return "sqlite" }

// Ping checks if the database is accessible.
func (c *sqliteClient) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// Close closes the database connection.
func (c *sqliteClient) Close() error {
	return c.db.Close()
}

// Query executes any SQL statement and returns normalized query results.
// For SELECTs and row-returning statements, it populates Columns and Rows.
// For DML (INSERT/UPDATE/DELETE/etc.), it populates RowsAffected.
func (c *sqliteClient) Query(ctx context.Context, query string) (*QueryResult, error) {
	result := &QueryResult{}
	start := time.Now()

	stmt := strings.TrimSpace(query)
	upperStmt := strings.ToUpper(stmt)

	isDML := strings.HasPrefix(upperStmt, "INSERT") ||
		strings.HasPrefix(upperStmt, "UPDATE") ||
		strings.HasPrefix(upperStmt, "DELETE") ||
		strings.HasPrefix(upperStmt, "CREATE") ||
		strings.HasPrefix(upperStmt, "DROP") ||
		strings.HasPrefix(upperStmt, "ALTER")

	if isDML {
		res, err := c.db.ExecContext(ctx, query)
		result.Duration = time.Since(start)
		if err != nil {
			return nil, fmt.Errorf("failed to execute statement: %w", err)
		}
		result.RowsAffected, _ = res.RowsAffected()
		return result, nil
	}

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to read column metadata: %w", err)
	}
	result.Columns = make([]ColumnMeta, len(colTypes))
	for i, ct := range colTypes {
		result.Columns[i] = ColumnMeta{
			Name:             ct.Name(),
			DatabaseTypeName: ct.DatabaseTypeName(),
		}
	}

	scanBuf := make([]any, len(colTypes))
	scanPtrs := make([]any, len(colTypes))
	for i := range scanBuf {
		scanPtrs[i] = &scanBuf[i]
	}

	for rows.Next() {
		if err := rows.Scan(scanPtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		row := make([]string, len(scanBuf))
		for i, v := range scanBuf {
			row[i] = valueToString(v)
		}
		result.Rows = append(result.Rows, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	result.Duration = time.Since(start)
	return result, nil
}

// Tables returns all tables and views from SQLite's master schema.
func (c *sqliteClient) Tables(ctx context.Context) ([]string, error) {
	const q = `
		SELECT name FROM sqlite_master
		WHERE type IN ('table', 'view')
		  AND name NOT LIKE 'sqlite_%'
		ORDER BY name;
	`
	rows, err := c.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

// TableDDL returns the table creation schema statement querying sqlite_master.
func (c *sqliteClient) TableDDL(ctx context.Context, tableName string) (string, error) {
	const q = `
		SELECT sql FROM sqlite_master
		WHERE name = ? AND type IN ('table', 'view');
	`
	var ddl string
	err := c.db.QueryRowContext(ctx, q, tableName).Scan(&ddl)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("table %q not found", tableName)
	}
	if err != nil {
		return "", fmt.Errorf("failed to retrieve DDL for %q: %w", tableName, err)
	}
	return ddl, nil
}

// valueToString converts values returned by the driver into a human-readable string.
func valueToString(v any) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case []byte:
		return string(val)
	case string:
		return val
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case time.Time:
		return val.Format(time.RFC3339)
	default:
		return fmt.Sprintf("%v", val)
	}
}
