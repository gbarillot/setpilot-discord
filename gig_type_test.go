package main

import (
	"strings"
	"testing"
)

func TestClassifyGigQuestion(t *testing.T) {
	for _, test := range []struct {
		question string
		want     gigTypeIntent
	}{
		{"Quels sont les prochains concerts ?", gigOnly},
		{"Quels sont les gigs ?", gigOnly},
		{"Quelle est la prochaine répétition ?", rehearsalOnly},
		{"Quelles répétitions sont prévues ?", rehearsalOnly},
		{"Quand est la répetition ?", rehearsalOnly},
		{"Quand est la repetition ?", rehearsalOnly},
		{"C'est quand la répet' ?", rehearsalOnly},
		{"C'est quand la repet' ?", rehearsalOnly},
		{"Qui répète mardi ?", rehearsalOnly},
		{"Qui est présent à la répét du 2026-10-27 ?", rehearsalOnly},
		{"When is the next rehearsal?", rehearsalOnly},
		{"Quels concerts et répétitions sont prévus ?", bothGigTypes},
		{"Quelle est la prochaine date ?", gigOnly},
	} {
		t.Run(test.question, func(t *testing.T) {
			if got := classifyGigQuestion(test.question); got != test.want {
				t.Errorf("classifyGigQuestion(%q) = %q, want %q", test.question, got, test.want)
			}
		})
	}
}

func TestValidateGigTypeQuery(t *testing.T) {
	for _, test := range []struct {
		name   string
		query  string
		intent gigTypeIntent
		wantOK bool
	}{
		{"gig", "SELECT * FROM gigs WHERE gigs.type = 'gig'", gigOnly, true},
		{"rehearsal", "SELECT * FROM gigs AS g WHERE g.type = 'rehearsal'", rehearsalOnly, true},
		{"quoted table", `SELECT * FROM "gigs" WHERE type = 'gig'`, gigOnly, true},
		{"missing filter", "SELECT * FROM gigs", gigOnly, false},
		{"wrong filter", "SELECT * FROM gigs WHERE type = 'rehearsal'", gigOnly, false},
		{"wrong type alongside right", "SELECT * FROM gigs WHERE type = 'gig' OR type = 'rehearsal'", gigOnly, false},
		{"both types", "SELECT type, count(*) FROM gigs GROUP BY type", bothGigTypes, true},
		{"no gigs table", "SELECT * FROM setlists", gigOnly, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateGigTypeQuery(test.query, test.intent); (err == nil) != test.wantOK {
				t.Errorf("validateGigTypeQuery(%q, %q) error = %v, wantOK %v", test.query, test.intent, err, test.wantOK)
			}
		})
	}
}

func TestSQLPromptIncludesGigIntent(t *testing.T) {
	for _, intent := range []gigTypeIntent{gigOnly, rehearsalOnly, bothGigTypes} {
		if !strings.Contains(sqlSystemPrompt(intent), `requested gig type is "`+string(intent)+`"`) {
			t.Errorf("SQL prompt does not specify %q intent", intent)
		}
	}
}
