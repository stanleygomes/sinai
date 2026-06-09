package database

import "time"

// ColumnMeta describes the metadata of a column in a query result.
type ColumnMeta struct {
	// Name is the column name returned by the driver.
	Name string
	// DatabaseTypeName is the type declared in the DDL (e.g., "TEXT", "INTEGER").
	DatabaseTypeName string
}

// QueryResult encapsulates the result of a SELECT query or other SQL statement.
type QueryResult struct {
	// Columns contains the metadata of the returned columns.
	Columns []ColumnMeta
	// Rows contains the row values as slices of formatted strings.
	// NULL values are represented as "NULL".
	Rows [][]string
	// RowsAffected is populated for DML statements (INSERT/UPDATE/DELETE/etc.).
	RowsAffected int64
	// Duration is the total execution time of the query.
	Duration time.Duration
}
