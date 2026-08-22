package main

import "testing"

func TestValidateSelectQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		want    string
		wantErr bool
	}{
		{name: "adds limit", query: "SELECT * FROM bands", want: "SELECT * FROM bands\nLIMIT 50"},
		{name: "keeps limit", query: "select * from bands limit 5", want: "select * from bands limit 5"},
		{name: "strips fence", query: "```sql\nSELECT 1;\n```", want: "SELECT 1\nLIMIT 50"},
		{name: "rejects write", query: "SELECT delete FROM bands", wantErr: true},
		{name: "rejects multiple statements", query: "SELECT 1; SELECT 2", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ValidateSelectQuery(test.query)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("got %q, %v; want %q", got, err, test.want)
			}
		})
	}
}
