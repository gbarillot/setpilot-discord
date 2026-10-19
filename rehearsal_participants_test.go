package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNextRehearsalParticipantsQuestion(t *testing.T) {
	for _, test := range []struct {
		question string
		want     bool
	}{
		{"Quels sont les présents à la prochaine répet?", true},
		{"Qui sera là à la prochaine répet ?", true},
		{"Qui vient à la prochaine répétition ?", true},
		{"Quels participants à la prochaine répetition ?", true},
		{"Qui sera présent à la prochaine rehearsal ?", true},
		{"Quand est la prochaine répet ?", false},
		{"Qui sera là au prochain concert ?", false},
		{"Qui participe à toutes les répétitions ?", false},
		{"Quels concerts et répétitions sont prévus ?", false},
		{"Quels sont les présents à la prochaine répétition du 2026-07-12 ?", false},
	} {
		t.Run(test.question, func(t *testing.T) {
			if got := isNextRehearsalParticipantsQuestion(test.question); got != test.want {
				t.Errorf("isNextRehearsalParticipantsQuestion(%q) = %v, want %v", test.question, got, test.want)
			}
		})
	}
}

func TestNextRehearsalParticipants(t *testing.T) {
	path := filepath.Join(t.TempDir(), "setpilot.sqlite3")
	seed, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = seed.Exec(`
		CREATE TABLE bands (id TEXT PRIMARY KEY, name TEXT);
		CREATE TABLE gigs (id TEXT PRIMARY KEY, band_id TEXT, type TEXT, start_at TEXT);
		CREATE TABLE participants (gig_id TEXT, user_id TEXT);
		CREATE TABLE users (id TEXT PRIMARY KEY, first_name TEXT, last_name TEXT);
		CREATE TABLE bands_users (band_id TEXT, user_id TEXT);
		INSERT INTO bands VALUES ('groove', 'Groove Station'), ('other', 'Other Band');
		INSERT INTO gigs VALUES
			('old', 'groove', 'rehearsal', '2026-07-11T12:00:00Z'),
			('concert', 'groove', 'gig', '2026-07-12T13:00:00Z'),
			('other', 'other', 'rehearsal', '2026-07-12T13:00:00Z'),
			('unscheduled', 'groove', 'rehearsal', NULL),
			('next', 'groove', 'rehearsal', '2026-07-12 14:00:00 +0000 UTC'),
			('later', 'groove', 'rehearsal', '2026-07-13T14:00:00Z');
		INSERT INTO users VALUES ('alice', 'Alice', 'Dupont'), ('bob', 'Bob', NULL), ('charlie', 'Charlie', 'Martin'), ('outsider', 'Outsider', 'Other');
		INSERT INTO bands_users VALUES ('groove', 'alice'), ('groove', 'bob'), ('groove', 'charlie');
		INSERT INTO participants VALUES
			('next', 'bob'), ('next', 'alice'), ('next', 'charlie'), ('next', 'outsider'),
			('later', 'outsider'), ('old', 'outsider'), ('concert', 'outsider');
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := seed.Close(); err != nil {
		t.Fatal(err)
	}
	database, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	bot := &Bot{db: database}
	ctx := context.Background()

	for _, test := range []struct {
		name string
		now  time.Time
		want string
	}{
		{"upcoming", time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC), "Participants inscrits à la prochaine répétition : Alice, Bob, Charlie."},
		{"no signups", time.Date(2026, 7, 12, 15, 0, 0, 0, time.UTC), "Personne n'est inscrit à la prochaine répétition pour le moment."},
		{"no upcoming", time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC), "Aucune répétition à venir n'est programmée."},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := bot.answerNextRehearsalParticipants(ctx, test.now)
			if err != nil || got != test.want {
				t.Fatalf("answer = %q, error = %v; want %q", got, err, test.want)
			}
		})
	}

	// The lookup must not rely on the LLM or expose unconfirmed members.
	answer, err := bot.answerNextRehearsalParticipants(ctx, time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC))
	if err != nil || strings.Contains(answer, "Outsider") {
		t.Fatalf("unexpected participant: answer = %q, error = %v", answer, err)
	}
}

func TestRehearsalParticipantsOnDateQuestion(t *testing.T) {
	for _, test := range []struct {
		question string
		want     string
		ok       bool
	}{
		{"Quels sont les présents à la répetition du 2026-10-27 ?", "2026-10-27", true},
		{"Qui est présent à la répétition du 2026-02-30 ?", "", false},
		{"Quels sont les présents au concert du 2026-10-27 ?", "", false},
		{"Quand est la répétition du 2026-10-27 ?", "", false},
	} {
		t.Run(test.question, func(t *testing.T) {
			got, ok := rehearsalParticipantsDate(test.question)
			if ok != test.ok || (ok && got.Format("2006-01-02") != test.want) {
				t.Errorf("rehearsalParticipantsDate(%q) = (%s, %v), want (%s, %v)", test.question, got.Format("2006-01-02"), ok, test.want, test.ok)
			}
		})
	}
}

func TestAnswerRehearsalParticipantsOnDate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "setpilot.sqlite3")
	seed, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = seed.Exec(`
		CREATE TABLE bands (id TEXT PRIMARY KEY, name TEXT);
		CREATE TABLE gigs (id TEXT PRIMARY KEY, band_id TEXT, type TEXT, start_at TEXT);
		CREATE TABLE participants (gig_id TEXT, user_id TEXT);
		CREATE TABLE users (id TEXT PRIMARY KEY, first_name TEXT, last_name TEXT);
		CREATE TABLE bands_users (band_id TEXT, user_id TEXT);
		INSERT INTO bands VALUES ('groove', 'Groove Station');
		INSERT INTO gigs VALUES ('target', 'groove', 'rehearsal', '2026-10-27 15:29:00 +0000 UTC');
		INSERT INTO users VALUES ('alice', 'Alice', 'Dupont');
		INSERT INTO bands_users VALUES ('groove', 'alice');
		INSERT INTO participants VALUES ('target', 'alice');
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := seed.Close(); err != nil {
		t.Fatal(err)
	}
	database, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	bot := &Bot{db: database}
	answer, details, err := bot.answerQuestion(context.Background(), "Quels sont les présents à la répetition du 2026-10-27 ?", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if answer != "Participants inscrits à la répétition du 27 octobre 2026 : Alice." {
		t.Errorf("answer = %q", answer)
	}
	if details.path != "rehearsal participants by date" || details.participants != 1 {
		t.Errorf("details = %+v", details)
	}
}
