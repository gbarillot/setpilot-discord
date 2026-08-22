package main

import "testing"

func TestIsWhitelistedMessage(t *testing.T) {
	whitelist := parseStringSet("concert,date,durée")
	for _, test := range []struct {
		message string
		want    bool
	}{
		{message: "Quelle est la durée ?", want: true},
		{message: "Quels concerts sont prévus ?", want: true},
		{message: "Bonjour", want: false},
	} {
		if got := isWhitelistedMessage(test.message, whitelist); got != test.want {
			t.Errorf("isWhitelistedMessage(%q) = %v, want %v", test.message, got, test.want)
		}
	}
}
