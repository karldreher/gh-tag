package lib

import (
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

// headCommitTitleCmd is the factory for git log to get HEAD subject. Replaceable in tests.
var headCommitTitleCmd = func() *exec.Cmd {
	return exec.Command("git", "log", "-1", "--format=%s")
}

// HeadCommitTitle returns the subject line of the HEAD commit.
func HeadCommitTitle() (string, error) {
	out, err := headCommitTitleCmd().Output()
	if err != nil {
		return "", fmt.Errorf("reading HEAD commit title: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
