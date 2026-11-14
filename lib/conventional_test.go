package lib

import (
	"errors"
	"strings"
	"testing"
)

// TestParseConventionalCommitBumpType verifies that conventional commit subject
// lines are correctly mapped to major, minor, or patch bump types, and that
// non-conventional subjects return an error.
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

// TestMaxBumpType verifies that the highest-order bump across a commit range
// wins, that non-conventional subjects are skipped and reported, and that
// empty or fully non-conventional ranges are errors.
func TestMaxBumpType(t *testing.T) {
	tests := []struct {
		name        string
		titles      []string
		want        string
		wantSkipped []string
		wantErr     bool
	}{
		// Issue #14: newest-first git log order, chore at HEAD must not mask feat.
		{"issue 14 feat then chore", []string{"chore: readme update", "feat: example"}, "minor", nil, false},
		{"all patch", []string{"fix: a", "chore: b", "docs: c"}, "patch", nil, false},
		{"breaking wins", []string{"fix: a", "feat: b", "feat!: c"}, "major", nil, false},
		{"breaking first still reports later skips", []string{"feat!: c", "WIP", "fix: a"}, "major", []string{"WIP"}, false},
		{"merge commit skipped", []string{"Merge branch 'x'", "feat: b"}, "minor", []string{"Merge branch 'x'"}, false},
		{"all non-conventional", []string{"WIP", "Merge branch 'x'"}, "", []string{"WIP", "Merge branch 'x'"}, true},
		{"empty range", nil, "", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, skipped, err := MaxBumpType(tt.titles)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("bump = %q, want %q", got, tt.want)
			}
			if strings.Join(skipped, "|") != strings.Join(tt.wantSkipped, "|") {
				t.Errorf("skipped = %q, want %q", skipped, tt.wantSkipped)
			}
		})
	}
}

// TestMaxBumpType_EmptyIsErrNoCommits verifies callers can distinguish an
// empty range from a range with no conventional commits.
func TestMaxBumpType_EmptyIsErrNoCommits(t *testing.T) {
	if _, _, err := MaxBumpType(nil); !errors.Is(err, ErrNoCommits) {
		t.Errorf("err = %v, want ErrNoCommits", err)
	}
}
