package main

import (
	"fmt"
	"regexp"
	"strings"
)

type gigTypeIntent string

const (
	gigOnly       gigTypeIntent = "gig"
	rehearsalOnly gigTypeIntent = "rehearsal"
	bothGigTypes  gigTypeIntent = "both"
)

var gigTablePattern = regexp.MustCompile(`(?i)\b(?:from|join)\s+(?:main\.)?(?:gigs\b|"gigs")`)
var gigTypePredicatePattern = regexp.MustCompile(`(?i)(?:\b[a-z_][a-z_0-9]*\.)?\btype\s*=\s*'(gig|rehearsal)'|'(gig|rehearsal)'\s*=\s*(?:\b[a-z_][a-z_0-9]*\.)?\btype`)

func classifyGigQuestion(question string) gigTypeIntent {
	var gig, rehearsal bool
	for _, word := range messageWordPattern.FindAllString(strings.ToLower(question), -1) {
		variants := wordVariants(word)
		if anyVariantInSet(variants, map[string]bool{"concert": true, "gig": true}) {
			gig = true
		}
		if anyVariantInSet(variants, map[string]bool{
			"répétition": true, "répetition": true, "repetition": true,
			"répet": true, "repet": true, "rehearsal": true,
		}) || strings.HasPrefix(word, "répét") || strings.HasPrefix(word, "répèt") || strings.HasPrefix(word, "répet") || strings.HasPrefix(word, "repet") {
			rehearsal = true
		}
	}
	if gig && rehearsal {
		return bothGigTypes
	}
	if rehearsal {
		return rehearsalOnly
	}
	return gigOnly
}

// Check that generated SQL explicitly filters the gigs table when the question
// asks for only one kind of event. This is a sanity check, not a SQL parser or
// an authorization boundary.
func validateGigTypeQuery(query string, intent gigTypeIntent) error {
	if intent == bothGigTypes || !gigTablePattern.MatchString(query) {
		return nil
	}
	matches := gigTypePredicatePattern.FindAllStringSubmatch(query, -1)
	if len(matches) == 0 {
		return fmt.Errorf("query on gigs is missing a type filter for %s", intent)
	}
	for _, match := range matches {
		value := match[1]
		if value == "" {
			value = match[2]
		}
		if !strings.EqualFold(value, string(intent)) {
			return fmt.Errorf("query on gigs includes the wrong type filter: %s", value)
		}
	}
	return nil
}
