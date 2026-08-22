package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSQLiteDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "setpilot.sqlite3")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	database, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	if _, err := database.db.Exec("CREATE TABLE bands (id INTEGER, name TEXT)"); err == nil {
		t.Fatal("expected read-only database")
	}
	if _, err := database.db.Exec("PRAGMA query_only = OFF"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.db.Exec("CREATE TABLE bands (id INTEGER, name TEXT)"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.db.Exec("INSERT INTO bands VALUES (1, 'Groove Station')"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.db.Exec("PRAGMA query_only = ON"); err != nil {
		t.Fatal(err)
	}

	schema, err := database.SchemaSummary(context.Background())
	if err != nil || schema != "bands(id INTEGER, name TEXT)" {
		t.Fatalf("schema = %q, err = %v", schema, err)
	}
	rows, err := database.FetchRows(context.Background(), "SELECT * FROM bands")
	if err != nil || len(rows) != 1 || rows[0]["name"] != "Groove Station" {
		t.Fatalf("rows = %#v, err = %v", rows, err)
	}
}
