// Package database define a interface de acesso ao banco de dados e
// fornece implementações para os drivers suportados.
// A interface DB abstrai o driver subjacente, permitindo trocar
// SQLite por Postgres (ou qualquer driver database/sql) sem alterar
// o restante da aplicação.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	// Driver SQLite puro Go (sem CGO).
	_ "modernc.org/sqlite"
)

// ColumnMeta descreve metadados de uma coluna no resultado de uma query.
type ColumnMeta struct {
	// Name é o nome da coluna retornado pelo driver.
	Name string
	// DatabaseTypeName é o tipo declarado na DDL (ex: "TEXT", "INTEGER").
	DatabaseTypeName string
}

// QueryResult encapsula o resultado de uma query SELECT.
type QueryResult struct {
	// Columns contém os metadados das colunas retornadas.
	Columns []ColumnMeta
	// Rows contém as linhas como slices de strings formatadas para exibição.
	// Valores NULL são representados como "NULL".
	Rows [][]string
	// RowsAffected é preenchido para queries DML (INSERT/UPDATE/DELETE).
	RowsAffected int64
	// Duration é o tempo total de execução da query.
	Duration time.Duration
}

// DB é a interface que toda implementação de banco de dados deve satisfazer.
// Ela se mantém intencionalmente pequena — novas capacidades (transações,
// prepared statements) podem ser adicionadas à medida que o projeto cresce.
type DB interface {
	// Query executa uma instrução SQL e retorna os resultados.
	// Funciona tanto para SELECTs quanto para DML.
	Query(ctx context.Context, sql string) (*QueryResult, error)

	// Tables retorna os nomes de todas as tabelas do schema atual/padrão.
	Tables(ctx context.Context) ([]string, error)

	// TableDDL retorna o DDL de criação de uma tabela (CREATE TABLE ...).
	TableDDL(ctx context.Context, tableName string) (string, error)

	// Ping verifica se a conexão está ativa.
	Ping(ctx context.Context) error

	// Close encerra a conexão com o banco.
	Close() error

	// DriverName retorna o nome do driver (ex: "sqlite", "postgres").
	DriverName() string
}

// sqliteClient implementa DB para o driver modernc.org/sqlite.
type sqliteClient struct {
	db     *sql.DB
	dsn    string
}

// OpenSQLite abre (ou cria) um banco SQLite no caminho especificado pelo DSN.
// O DSN deve ser o caminho do arquivo (ex: "/home/user/data.db") ou
// ":memory:" para banco em memória.
func OpenSQLite(dsn string) (DB, error) {
	// O driver modernc/sqlite registra-se como "sqlite".
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir sqlite (%s): %w", dsn, err)
	}

	// SQLite não suporta concorrência com múltiplas conexões de escrita.
	db.SetMaxOpenConns(1)

	// Habilita WAL mode e chaves estrangeiras via PRAGMA.
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA foreign_keys=ON;",
		"PRAGMA busy_timeout=5000;",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("erro ao aplicar pragma %q: %w", pragma, err)
		}
	}

	return &sqliteClient{db: db, dsn: dsn}, nil
}

// Open é o ponto de entrada genérico que seleciona o cliente correto
// com base no nome do driver. Adicione novos cases aqui para suportar
// outros bancos de dados no futuro.
func Open(driver, dsn string) (DB, error) {
	switch strings.ToLower(driver) {
	case "sqlite":
		return OpenSQLite(dsn)
	// Exemplo de extensão futura:
	// case "postgres":
	//     return OpenPostgres(dsn)
	default:
		return nil, fmt.Errorf("driver %q não suportado", driver)
	}
}

// DriverName retorna o identificador do driver.
func (c *sqliteClient) DriverName() string { return "sqlite" }

// Ping verifica se o banco está acessível.
func (c *sqliteClient) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// Close encerra a conexão.
func (c *sqliteClient) Close() error {
	return c.db.Close()
}

// Query executa qualquer instrução SQL e retorna os resultados normalizados.
// Para SELECTs, popula Columns e Rows.
// Para DML (INSERT/UPDATE/DELETE), popula RowsAffected.
func (c *sqliteClient) Query(ctx context.Context, query string) (*QueryResult, error) {
	result := &QueryResult{}
	start := time.Now()

	// Detecta o tipo de statement pela primeira palavra para escolher o path correto.
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
			return nil, fmt.Errorf("erro ao executar statement: %w", err)
		}
		result.RowsAffected, _ = res.RowsAffected()
		return result, nil
	}

	// Path de leitura: SELECT e outros statements que retornam linhas.
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("erro ao executar query: %w", err)
	}
	defer rows.Close()

	// Lê metadados das colunas.
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("erro ao ler metadados das colunas: %w", err)
	}
	result.Columns = make([]ColumnMeta, len(colTypes))
	for i, ct := range colTypes {
		result.Columns[i] = ColumnMeta{
			Name:             ct.Name(),
			DatabaseTypeName: ct.DatabaseTypeName(),
		}
	}

	// Lê todas as linhas.
	scanBuf := make([]any, len(colTypes))
	scanPtrs := make([]any, len(colTypes))
	for i := range scanBuf {
		scanPtrs[i] = &scanBuf[i]
	}

	for rows.Next() {
		if err := rows.Scan(scanPtrs...); err != nil {
			return nil, fmt.Errorf("erro ao escanear linha: %w", err)
		}
		row := make([]string, len(scanBuf))
		for i, v := range scanBuf {
			row[i] = valueToString(v)
		}
		result.Rows = append(result.Rows, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro durante iteração das linhas: %w", err)
	}

	result.Duration = time.Since(start)
	return result, nil
}

// Tables retorna todas as tabelas e views do schema principal do SQLite.
func (c *sqliteClient) Tables(ctx context.Context) ([]string, error) {
	const q = `
		SELECT name FROM sqlite_master
		WHERE type IN ('table', 'view')
		  AND name NOT LIKE 'sqlite_%'
		ORDER BY name;
	`
	rows, err := c.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar tabelas: %w", err)
	}
	defer rows.Close()

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

// TableDDL retorna o DDL de criação da tabela consultando sqlite_master.
func (c *sqliteClient) TableDDL(ctx context.Context, tableName string) (string, error) {
	const q = `
		SELECT sql FROM sqlite_master
		WHERE name = ? AND type IN ('table', 'view');
	`
	var ddl string
	err := c.db.QueryRowContext(ctx, q, tableName).Scan(&ddl)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("tabela %q não encontrada", tableName)
	}
	if err != nil {
		return "", fmt.Errorf("erro ao buscar DDL de %q: %w", tableName, err)
	}
	return ddl, nil
}

// valueToString converte um valor retornado pelo driver para string legível.
// Trata os tipos mais comuns retornados pelo driver SQLite (modernc).
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
