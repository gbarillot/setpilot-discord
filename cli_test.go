package main

import (
	"bytes"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func TestQuestionCLIWithoutDiscordOrEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "setpilot.db")
	seed, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = seed.Exec(`
		CREATE TABLE bands (id TEXT, name TEXT);
		CREATE TABLE gigs (id TEXT, band_id TEXT, type TEXT, start_at TEXT);
		CREATE TABLE participants (gig_id TEXT, user_id TEXT);
		CREATE TABLE users (id TEXT, first_name TEXT, last_name TEXT);
		CREATE TABLE bands_users (band_id TEXT, user_id TEXT);
		INSERT INTO bands VALUES ('groove', 'Groove Station');
		INSERT INTO gigs VALUES ('next', 'groove', 'rehearsal', '2026-10-20T14:00:00Z');
		INSERT INTO users VALUES ('alice', 'Alice', 'Dupont'), ('bob', 'Bob', 'Martin');
		INSERT INTO bands_users VALUES ('groove', 'alice'), ('groove', 'bob');
		INSERT INTO participants VALUES ('next', 'bob'), ('next', 'alice');
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := seed.Close(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SET_PILOT_ENV_FILE", filepath.Join(t.TempDir(), "missing.env"))
	var output, diagnostics bytes.Buffer
	err = runQuestionCLI("Quels sont les présents à la prochaine répet?", path, true, "2026-10-19T12:00:00Z", &output, &diagnostics)
	if err != nil {
		t.Fatal(err)
	}
	if want := "Participants inscrits à la prochaine répétition : Alice Dupont, Bob Martin.\n"; output.String() != want {
		t.Errorf("answer = %q, want %q", output.String(), want)
	}
	if !strings.Contains(diagnostics.String(), "Participants returned: 2") {
		t.Errorf("diagnostics = %q, want participant count", diagnostics.String())
	}
}

func TestQuestionCLIRequiresLLMKeyForOtherQuestions(t *testing.T) {
	t.Setenv("SET_PILOT_ENV_FILE", filepath.Join(t.TempDir(), "missing.env"))
	path := filepath.Join(t.TempDir(), "setpilot.db")
	seed, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := seed.Exec("CREATE TABLE bands (id TEXT)"); err != nil {
		t.Fatal(err)
	}
	seed.Close()
	err = runQuestionCLI("Quand est la prochaine répet ?", path, false, "", &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "LLM questions require an .env file") {
		t.Fatalf("error = %v, want missing LLM config", err)
	}
}
