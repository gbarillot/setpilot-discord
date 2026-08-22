package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var forbiddenKeywords = map[string]bool{
	"alter": true, "call": true, "copy": true, "create": true, "delete": true,
	"do": true, "drop": true, "grant": true, "insert": true, "revoke": true,
	"truncate": true, "update": true,
}

var sqlTokenPattern = regexp.MustCompile(`\b[a-z_]+\b`)
var limitPattern = regexp.MustCompile(`\blimit\s+\d+\b`)

func ValidateSelectQuery(query string) (string, error) {
	query = strings.TrimSpace(stripCodeFence(query))
	if strings.HasSuffix(query, ";") {
		query = strings.TrimSpace(strings.TrimSuffix(query, ";"))
	}
	if strings.Contains(query, ";") {
		return "", fmt.Errorf("multiple SQL statements are not allowed")
	}
	lowered := strings.ToLower(query)
	if !strings.HasPrefix(lowered, "select ") && !strings.HasPrefix(lowered, "with ") {
		return "", fmt.Errorf("only SELECT queries are allowed")
	}
	var forbidden []string
	for _, token := range sqlTokenPattern.FindAllString(lowered, -1) {
		if forbiddenKeywords[token] {
			forbidden = append(forbidden, token)
		}
	}
	if len(forbidden) > 0 {
		sort.Strings(forbidden)
		return "", fmt.Errorf("forbidden SQL keyword: %s", forbidden[0])
	}
	if !limitPattern.MatchString(lowered) {
		query += "\nLIMIT 50"
	}
	return query, nil
}

func stripCodeFence(text string) string {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "```") {
		return text
	}
	lines := strings.Split(text, "\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "```") {
		lines = lines[1:]
	}
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}
