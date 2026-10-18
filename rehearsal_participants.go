package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const nextRehearsalParticipantsSQL = `
WITH next_rehearsal AS (
	SELECT g.id, g.band_id
	FROM gigs AS g
	JOIN bands AS b ON b.id = g.band_id
	WHERE b.name = 'Groove Station'
	  AND g.type = 'rehearsal'
	  AND datetime(g.start_at) >= datetime(?)
	ORDER BY datetime(g.start_at), g.id
	LIMIT 1
)
SELECT u.first_name, u.last_name
FROM next_rehearsal AS g
LEFT JOIN participants AS p ON p.gig_id = g.id
LEFT JOIN users AS u ON u.id = p.user_id
  AND EXISTS (SELECT 1 FROM bands_users AS bu WHERE bu.band_id = g.band_id AND bu.user_id = u.id)
ORDER BY u.first_name COLLATE NOCASE, u.last_name COLLATE NOCASE`

func isNextRehearsalParticipantsQuestion(question string) bool {
	if classifyGigQuestion(question) != rehearsalOnly {
		return false
	}
	words := make(map[string]bool)
	for _, word := range messageWordPattern.FindAllString(strings.ToLower(question), -1) {
		for variant := range wordVariants(word) {
			words[variant] = true
		}
	}
	if !anyVariantInSet(words, map[string]bool{
		"prochain": true, "prochaine": true, "suivant": true, "suivante": true, "next": true,
	}) {
		return false
	}
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

func (database *SQLiteDB) nextRehearsalParticipants(ctx context.Context, now time.Time) (bool, []string, error) {
	rows, err := database.db.QueryContext(ctx, nextRehearsalParticipantsSQL, now.UTC().Format(time.RFC3339))
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
	found, names, err := bot.db.nextRehearsalParticipants(ctx, now)
	if err != nil {
		return "", fmt.Errorf("find next rehearsal participants: %w", err)
	}
	if !found {
		return "Aucune répétition à venir n'est programmée.", nil
	}
	if len(names) == 0 {
		return "Personne n'est inscrit à la prochaine répétition pour le moment.", nil
	}
	return "Participants inscrits à la prochaine répétition : " + strings.Join(names, ", ") + ".", nil
}
