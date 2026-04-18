package lib

import (
	"testing"
)

func TestParseConventionalCommitBumpType(t *testing.T) {
	tests := []struct {
		title   string
		want    string
		wantErr bool
	}{
		{"feat: add thing", "minor", false},
		{"feat(api): add endpoint", "minor", false},
		{"feat!: breaking change", "major", false},
		{"feat(api)!: breaking change", "major", false},
		{"fix: bug", "patch", false},
		{"fix(auth): null pointer", "patch", false},
		{"chore: cleanup", "patch", false},
		{"docs: update readme", "patch", false},
		{"refactor: extract helper", "patch", false},
		{"refactor!: rename public API", "major", false},
		{"perf: faster query", "patch", false},
		{"ci: update workflow", "patch", false},
		{"FEAT: uppercase type", "minor", false},
		{"just a plain commit", "", true},
		{"WIP something", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got, err := ParseConventionalCommitBumpType(tt.title)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
