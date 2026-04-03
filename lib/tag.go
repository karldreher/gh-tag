package lib

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// ParseVersion parses a semver tag string with the given prefix into its
// numeric components. It returns ok=false (and zero values) for any input
// that is not a well-formed "prefix + major.minor.patch" string, including:
//   - wrong or missing prefix
//   - fewer or more than three dot-separated components
//   - non-numeric components
//   - negative numbers
//   - pre-release suffixes (e.g. "v1.0.0-beta")
//   - annotated tag dereferences (e.g. "v1.0.0^{}")
//   - empty string
func ParseVersion(tag, prefix string) (major, minor, patch int, ok bool) {
	if tag == "" {
		return 0, 0, 0, false
	}
	if !strings.HasPrefix(tag, prefix) {
		return 0, 0, 0, false
	}
	rest := tag[len(prefix):]
	if rest == "" {
		return 0, 0, 0, false
	}

	// Reject anything with a "^{}" annotated-tag dereference suffix.
	if strings.Contains(rest, "^") {
		return 0, 0, 0, false
	}

	parts := strings.Split(rest, ".")
	if len(parts) != 3 {
		return 0, 0, 0, false
	}

	// Each part must be a non-negative integer with no extra characters
	// (e.g. "0-beta" must fail, not silently succeed as 0).
	nums := make([]int, 3)
	for i, p := range parts {
		// Reject empty parts, leading signs, or any non-digit characters
		// that Atoi would silently accept (it does not accept them, but be explicit).
		if p == "" {
			return 0, 0, 0, false
		}
		// Reject leading plus/minus signs — Atoi accepts them, we don't.
		if p[0] == '+' || p[0] == '-' {
			return 0, 0, 0, false
		}
		// Reject any non-digit character in the component (catches "0-beta", "1rc2", etc.)
		for _, c := range p {
			if c < '0' || c > '9' {
				return 0, 0, 0, false
			}
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return 0, 0, 0, false
		}
		nums[i] = n
	}

	return nums[0], nums[1], nums[2], true
}

// FindLatestTag scans tags for the highest semver tag matching prefix.
// Comparison is numeric (not lexicographic) so v1.0.10 > v1.0.9.
// Tags that do not parse as valid semver with prefix are silently skipped.
// Returns found=false when no valid semver tags exist in the slice.
func FindLatestTag(tags []string, prefix string) (major, minor, patch int, found bool) {
	for _, t := range tags {
		ma, mi, pa, ok := ParseVersion(t, prefix)
		if !ok {
			continue
		}
		if !found {
			major, minor, patch, found = ma, mi, pa, true
			continue
		}
		if ma > major ||
			(ma == major && mi > minor) ||
			(ma == major && mi == minor && pa > patch) {
			major, minor, patch = ma, mi, pa
		}
	}
	return
}

// BumpVersion returns the next version for the given bump type.
// "major" increments major and resets minor and patch to zero.
// "minor" increments minor and resets patch to zero.
// "patch" increments patch only.
// Any other bumpType returns the inputs unchanged.
func BumpVersion(major, minor, patch int, bumpType string) (int, int, int) {
	switch bumpType {
	case "major":
		return major + 1, 0, 0
	case "minor":
		return major, minor + 1, 0
	case "patch":
		return major, minor, patch + 1
	default:
		return major, minor, patch
	}
}

// FormatTag formats a semver version as a tag string with the given prefix.
func FormatTag(prefix string, major, minor, patch int) string {
	return fmt.Sprintf("%s%d.%d.%d", prefix, major, minor, patch)
}

// HasTagsWithDifferentPrefix reports whether the remote has tags but none of
// them match prefix. This is the onboarding mismatch scenario: the repo uses a
// tagging convention (e.g. "release-") that differs from the configured prefix
// (e.g. "v"). Callers should warn the user rather than silently treating the
// repo as untagged.
func HasTagsWithDifferentPrefix(tags []string, prefix string) bool {
	if len(tags) == 0 {
		return false
	}
	_, _, _, found := FindLatestTag(tags, prefix)
	return !found
}

// listRemoteTagsCmd is the factory for the git ls-remote command.
// It is a package-level variable so tests can replace it.
var listRemoteTagsCmd = func() *exec.Cmd {
	return exec.Command("git", "ls-remote", "--tags", "origin")
}

// ListRemoteTags fetches tag names from the remote via git ls-remote.
// It strips the "refs/tags/" prefix and skips annotated tag dereferences
// (lines ending in "^{}"). Returns nil, nil when the remote has no tags.
func ListRemoteTags() ([]string, error) {
	cmd := listRemoteTagsCmd()
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running git ls-remote: %w", err)
	}
	if len(out) == 0 {
		return nil, nil
	}

	var tags []string
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		ref := parts[1]
		const refPrefix = "refs/tags/"
		if !strings.HasPrefix(ref, refPrefix) {
			continue
		}
		name := ref[len(refPrefix):]
		// Skip annotated tag dereferences — the plain name also appears.
		if strings.HasSuffix(name, "^{}") {
			continue
		}
		tags = append(tags, name)
	}
	return tags, nil
}

// createTagCmd is the factory for git tag creation. Replaceable in tests.
var createTagCmd = func(tag string) *exec.Cmd {
	return exec.Command("git", "tag", tag)
}

// CreateLocalTag creates a local git tag with the given name.
func CreateLocalTag(tag string) error {
	cmd := createTagCmd(tag)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("creating tag %s: %w (output: %s)", tag, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// pushTagCmd is the factory for git push of a tag. Replaceable in tests.
var pushTagCmd = func(tag string) *exec.Cmd {
	return exec.Command("git", "push", "origin", tag)
}

// PushTag pushes a local tag to origin.
func PushTag(tag string) error {
	cmd := pushTagCmd(tag)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pushing tag %s: %w (output: %s)", tag, err, strings.TrimSpace(string(out)))
	}
	return nil
}
