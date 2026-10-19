package main

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// SQLite understands RFC3339 timestamps but not time.Time.String() values such
// as "2026-10-19 15:29:00 +0000 UTC" written by older Fiber versions. Keep
// valid SQLite timestamps intact (including their offset) and normalize only
// the legacy values' first 19 UTC date/time characters.
const normalizedGigStartAtSQL = `COALESCE(
	datetime(g.start_at),
	datetime(replace(substr(g.start_at, 1, 19), ' ', 'T') || 'Z')
)`

const nextRehearsalParticipantsSQL = `
WITH rehearsals AS (
	SELECT g.id, g.band_id, ` + normalizedGigStartAtSQL + ` AS start_at_utc
	FROM gigs AS g
	JOIN bands AS b ON b.id = g.band_id
	WHERE b.name = 'Groove Station'
	  AND g.type = 'rehearsal'

), next_rehearsal AS (
	SELECT id, band_id
	FROM rehearsals
	WHERE start_at_utc >= datetime(?)
	ORDER BY start_at_utc, id
	LIMIT 1
)
SELECT u.first_name, u.last_name
FROM next_rehearsal AS g
LEFT JOIN participants AS p ON p.gig_id = g.id
LEFT JOIN users AS u ON u.id = p.user_id
  AND EXISTS (SELECT 1 FROM bands_users AS bu WHERE bu.band_id = g.band_id AND bu.user_id = u.id)
	ORDER BY u.first_name COLLATE NOCASE, u.last_name COLLATE NOCASE`

const rehearsalParticipantsOnDateSQL = `
WITH rehearsals AS (
	SELECT g.id, g.band_id, ` + normalizedGigStartAtSQL + ` AS start_at_utc
	FROM gigs AS g
	JOIN bands AS b ON b.id = g.band_id
	WHERE b.name = 'Groove Station'
	  AND g.type = 'rehearsal'
), matching_rehearsals AS (
	SELECT id, band_id
	FROM rehearsals
	WHERE date(start_at_utc) = date(?)
)
SELECT u.first_name, u.last_name
FROM matching_rehearsals AS g
LEFT JOIN participants AS p ON p.gig_id = g.id
LEFT JOIN users AS u ON u.id = p.user_id
  AND EXISTS (SELECT 1 FROM bands_users AS bu WHERE bu.band_id = g.band_id AND bu.user_id = u.id)
ORDER BY u.first_name COLLATE NOCASE, u.last_name COLLATE NOCASE`

var questionISODatePattern = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`)

func isNextRehearsalParticipantsQuestion(question string) bool {
	if classifyGigQuestion(question) != rehearsalOnly {
		return false
	}
	if _, ok := rehearsalParticipantsDate(question); ok {
		return false
	}
	return asksForNextRehearsal(question) && asksForParticipants(question)
}

func questionWords(question string) map[string]bool {
	words := make(map[string]bool)
	for _, word := range messageWordPattern.FindAllString(strings.ToLower(question), -1) {
		for variant := range wordVariants(word) {
			words[variant] = true
		}
	}
	return words
}

func asksForNextRehearsal(question string) bool {
	return anyVariantInSet(questionWords(question), map[string]bool{
		"prochain": true, "prochaine": true, "suivant": true, "suivante": true, "next": true,
	})
}

func asksForParticipants(question string) bool {
	words := questionWords(question)
	if anyVariantInSet(words, map[string]bool{
		"participant": true, "présent": true, "présente": true, "inscrit": true, "inscrite": true,
	}) {
		return true
	}
	return words["qui"] && anyVariantInSet(words, map[string]bool{
		"sera": true, "vient": true, "viendra": true, "participe": true,
		"assistera": true, "là": true,
	})
}

func rehearsalParticipantsDate(question string) (time.Time, bool) {
	if classifyGigQuestion(question) != rehearsalOnly || !asksForParticipants(question) {
		return time.Time{}, false
	}
	dateText := questionISODatePattern.FindString(question)
	if dateText == "" {
		return time.Time{}, false
	}
	date, err := time.Parse("2006-01-02", dateText)
	return date, err == nil
}

func (database *SQLiteDB) nextRehearsalParticipants(ctx context.Context, now time.Time) (bool, []string, error) {
	rows, err := database.db.QueryContext(ctx, nextRehearsalParticipantsSQL, now.UTC().Format(time.RFC3339))
	return scanRehearsalParticipantRows(rows, err)
}

func (database *SQLiteDB) rehearsalParticipantsOnDate(ctx context.Context, date time.Time) (bool, []string, error) {
	rows, err := database.db.QueryContext(ctx, rehearsalParticipantsOnDateSQL, date.Format("2006-01-02"))
	return scanRehearsalParticipantRows(rows, err)
}

func scanRehearsalParticipantRows(rows *sql.Rows, err error) (bool, []string, error) {
	if err != nil {
		return false, nil, err
	}
	defer rows.Close()

	found := false
	var names []string
	for rows.Next() {
		found = true
		var firstName, lastName sql.NullString
		if err := rows.Scan(&firstName, &lastName); err != nil {
			return false, nil, err
		}
		if firstName.Valid {
			names = append(names, strings.TrimSpace(firstName.String+" "+lastName.String))
		}
	}
	return found, names, rows.Err()
}

func (bot *Bot) answerNextRehearsalParticipants(ctx context.Context, now time.Time) (string, error) {
	answer, _, err := bot.answerNextRehearsalParticipantsWithCount(ctx, now)
	return answer, err
}

func (bot *Bot) answerNextRehearsalParticipantsWithCount(ctx context.Context, now time.Time) (string, int, error) {
	found, names, err := bot.db.nextRehearsalParticipants(ctx, now)
	if err != nil {
		return "", 0, fmt.Errorf("find next rehearsal participants: %w", err)
	}
	if !found {
		return "Aucune répétition à venir n'est programmée.", 0, nil
	}
	if len(names) == 0 {
		return "Personne n'est inscrit à la prochaine répétition pour le moment.", 0, nil
	}
	return "Participants inscrits à la prochaine répétition : " + strings.Join(names, ", ") + ".", len(names), nil
}

func (bot *Bot) answerRehearsalParticipantsOnDate(ctx context.Context, date time.Time) (string, int, error) {
	found, names, err := bot.db.rehearsalParticipantsOnDate(ctx, date)
	if err != nil {
		return "", 0, fmt.Errorf("find rehearsal participants on %s: %w", date.Format("2006-01-02"), err)
	}
	formattedDate := frenchDate(date)
	if !found {
		return "Aucune répétition n'est programmée le " + formattedDate + ".", 0, nil
	}
	if len(names) == 0 {
		return "Personne n'est inscrit à la répétition du " + formattedDate + " pour le moment.", 0, nil
	}
	return "Participants inscrits à la répétition du " + formattedDate + " : " + strings.Join(names, ", ") + ".", len(names), nil
}

func frenchDate(date time.Time) string {
	months := [...]string{"", "janvier", "février", "mars", "avril", "mai", "juin", "juillet", "août", "septembre", "octobre", "novembre", "décembre"}
	return fmt.Sprintf("%d %s %d", date.Day(), months[date.Month()], date.Year())
}
