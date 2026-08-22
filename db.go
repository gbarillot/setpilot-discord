package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

const tablesSQL = `
SELECT name
FROM sqlite_master
WHERE type = 'table' AND name LIKE 'setpilot_%'
ORDER BY name`

type SQLiteDB struct {
	db *sql.DB
}

func OpenSQLite(path string) (*SQLiteDB, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("SQLite database %s is not accessible: %w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("SQLite database path %s is a directory", path)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open SQLite database %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA query_only = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set SQLite read-only mode for %s: %w", path, err)
	}
	return &SQLiteDB{db: db}, nil
}

func (database *SQLiteDB) Close() error {
	return database.db.Close()
}

func (database *SQLiteDB) SchemaSummary(ctx context.Context) (string, error) {
	rows, err := database.db.QueryContext(ctx, tablesSQL)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return "", err
		}
		tableNames = append(tableNames, tableName)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	rows.Close()

	var summary []string
	for _, tableName := range tableNames {
		columns, err := database.tableColumns(ctx, tableName)
		if err != nil {
			return "", err
		}
		summary = append(summary, fmt.Sprintf("%s(%s)", tableName, strings.Join(columns, ", ")))
	}
	return strings.Join(summary, "\n"), nil
}

func (database *SQLiteDB) tableColumns(ctx context.Context, tableName string) ([]string, error) {
	identifier := strings.ReplaceAll(tableName, `"`, `""`)
	rows, err := database.db.QueryContext(ctx, `PRAGMA table_info("`+identifier+`")`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, primaryKey int
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			return nil, err
		}
		columns = append(columns, fmt.Sprintf("%s %s", name, dataType))
	}
	return columns, rows.Err()
}

func (database *SQLiteDB) FetchRows(ctx context.Context, query string) ([]map[string]any, error) {
	rows, err := database.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var result []map[string]any
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for index := range values {
			pointers[index] = &values[index]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(columns))
		for index, value := range values {
			if bytes, ok := value.([]byte); ok {
				row[columns[index]] = string(bytes)
			} else {
				row[columns[index]] = value
			}
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
