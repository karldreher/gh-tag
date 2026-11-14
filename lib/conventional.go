package lib

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var conventionalCommitRe = regexp.MustCompile(`^([a-zA-Z]+)(\([^)]*\))?(!)?\s*:`)

// ParseConventionalCommitBumpType maps a conventional commit subject line to a
// semver bump type. Breaking changes (!) → major; feat → minor; all others → patch.
// Returns an error if the title does not follow the conventional commit format.
func ParseConventionalCommitBumpType(title string) (string, error) {
	m := conventionalCommitRe.FindStringSubmatch(title)
	if m == nil {
		return "", fmt.Errorf("commit %q does not follow conventional commit format", title)
	}
	if m[3] == "!" {
		return "major", nil
	}
	if strings.ToLower(m[1]) == "feat" {
		return "minor", nil
	}
	return "patch", nil
}

// bumpRank orders bump types so the highest-order change can be selected.
var bumpRank = map[string]int{"patch": 1, "minor": 2, "major": 3}

// ErrNoCommits is returned by MaxBumpType when the commit range is empty.
var ErrNoCommits = errors.New("no commits since last tag")

// MaxBumpType returns the highest-order bump type across all commit titles
// (major > minor > patch), per semver. Titles that do not follow the
// conventional commit format are returned in skipped rather than failing.
// Returns ErrNoCommits for an empty slice, and an error when no title is
// conventional. Runs in O(n) and stops early on the first major.
func MaxBumpType(titles []string) (bump string, skipped []string, err error) {
	if len(titles) == 0 {
		return "", nil, ErrNoCommits
	}
	for i, title := range titles {
		b, perr := ParseConventionalCommitBumpType(title)
		if perr != nil {
			skipped = append(skipped, title)
			continue
		}
		if bumpRank[b] > bumpRank[bump] {
			bump = b
		}
		if bump == "major" {
			// Remaining titles cannot raise the bump; still report skipped ones.
			for _, rest := range titles[i+1:] {
				if !conventionalCommitRe.MatchString(rest) {
					skipped = append(skipped, rest)
				}
			}
			break
		}
	}
	if bump == "" {
		return "", skipped, fmt.Errorf("none of the %d commits since last tag follow conventional commit format", len(titles))
	}
	return bump, skipped, nil
}

// commitTitlesCmd is the factory for git log over a revision range.
// Replaceable in tests.
var commitTitlesCmd = func(rangeSpec string) *exec.Cmd {
	return exec.Command("git", "log", "--format=%s", rangeSpec)
}

// CommitTitlesSince returns the subject lines of commits reachable from HEAD
// but not from base (base..HEAD), newest first. An empty base returns every
// commit reachable from HEAD, which covers repositories with no tags yet.
func CommitTitlesSince(base string) ([]string, error) {
	rangeSpec := "HEAD"
	if base != "" {
		rangeSpec = base + "..HEAD"
	}
	out, err := commitTitlesCmd(rangeSpec).Output()
	if err != nil {
		return nil, fmt.Errorf("reading commits in %s: %w", rangeSpec, err)
	}
	var titles []string
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			titles = append(titles, line)
		}
	}
	return titles, nil
}
