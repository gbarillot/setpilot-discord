package main

import "testing"

func TestIsWhitelistedMessage(t *testing.T) {
	values, err := readEnvFile(".env.example")
	if err != nil {
		t.Fatal(err)
	}
	whitelist := parseStringSet(values["WHITE_LIST_WORDS"])
	for _, test := range []struct {
		message string
		want    bool
	}{
		{message: "Quelle est la date ?", want: true},
		{message: "Quels concerts sont prévus ?", want: true},
		{message: "Bonjour", want: false},
		{message: "Quand est la répétition ?", want: true},
		{message: "Quand est la répetition ?", want: true},
		{message: "Quand est la repetition ?", want: true},
		{message: "C'est quand la repet' ?", want: true},
		{message: "C'est quand la répet' ?", want: true},
	} {
		if got := isWhitelistedMessage(test.message, whitelist); got != test.want {
			t.Errorf("isWhitelistedMessage(%q) = %v, want %v", test.message, got, test.want)
		}
	}
}

func TestApostropheWhitelistVariants(t *testing.T) {
	for _, word := range []string{"repet'", "répet'"} {
		if !isWhitelistedMessage(word, parseStringSet(word)) {
			t.Errorf("apostrophe whitelist entry %q did not match the message", word)
		}
	}
}
