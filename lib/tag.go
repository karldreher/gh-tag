package lib

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"sort"
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
	var nums [3]int
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
	return major, minor, patch, found
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

// ParseBumpType maps a user input string to a canonical bump type.
// Single-char keys are case-sensitive (M=major, m=minor, p/P=patch).
// Full words are accepted with initial-cap variants (e.g. "Major", "Patch").
func ParseBumpType(input string) (string, error) {
	switch input {
	case "M", "major", "Major":
		return "major", nil
	case "m", "minor", "Minor":
		return "minor", nil
	case "p", "P", "patch", "Patch":
		return "patch", nil
	default:
		return "", fmt.Errorf("invalid bump type: %q (use M=major, m=minor, p=patch)", input)
	}
}

// SortTags returns tag names filtered to valid semver entries matching prefix,
// sorted by semantic version. Descending (newest first) by default;
// ascending=true reverses the order. Invalid tags are silently dropped,
// consistent with FindLatestTag.
func SortTags(tags []string, prefix string, ascending bool) []string {
	type entry struct {
		name                string
		major, minor, patch int
	}
	var entries []entry
	for _, t := range tags {
		major, minor, patch, ok := ParseVersion(t, prefix)
		if !ok {
			continue
		}
		entries = append(entries, entry{t, major, minor, patch})
	}
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.major != b.major {
			if ascending {
				return a.major < b.major
			}
			return a.major > b.major
		}
		if a.minor != b.minor {
			if ascending {
				return a.minor < b.minor
			}
			return a.minor > b.minor
		}
		if ascending {
			return a.patch < b.patch
		}
		return a.patch > b.patch
	})
	result := make([]string, 0, len(entries))
	for _, e := range entries {
		result = append(result, e.name)
	}
	return result
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

// ErrPushImmutable is returned by ForcePushTag when the remote rejects the
// force-push due to release immutability being enabled on the repository.
var ErrPushImmutable = errors.New("remote rejected force-push: release immutability enabled")

// shortSHA runs cmd, expects a git SHA on stdout, and returns the first 7 chars.
func shortSHA(cmd *exec.Cmd) (string, error) {
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	sha := strings.TrimSpace(string(out))
	if len(sha) >= 7 {
		sha = sha[:7]
	}
	return sha, nil
}

// resolveTagRefCmd resolves a tag to its commit SHA. Replaceable in tests.
var resolveTagRefCmd = func(tag string) *exec.Cmd {
	return exec.Command("git", "rev-list", "-n", "1", tag)
}

// ResolveTagRef returns the short (7-char) commit SHA that tag points to.
func ResolveTagRef(tag string) (string, error) {
	sha, err := shortSHA(resolveTagRefCmd(tag))
	if err != nil {
		return "", fmt.Errorf("resolving ref for tag %s: %w", tag, err)
	}
	return sha, nil
}

// resolveHeadCmd resolves HEAD to its commit SHA. Replaceable in tests.
var resolveHeadCmd = func() *exec.Cmd {
	return exec.Command("git", "rev-parse", "HEAD")
}

// ResolveHead returns the short (7-char) commit SHA of HEAD.
func ResolveHead() (string, error) {
	sha, err := shortSHA(resolveHeadCmd())
	if err != nil {
		return "", fmt.Errorf("resolving HEAD: %w", err)
	}
	return sha, nil
}

// overwriteTagCmd is the factory for git tag -f. Replaceable in tests.
var overwriteTagCmd = func(tag string) *exec.Cmd {
	return exec.Command("git", "tag", "-f", tag)
}

// OverwriteTag force-updates a local tag to point at the current HEAD.
func OverwriteTag(tag string) error {
	cmd := overwriteTagCmd(tag)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("overwriting tag %s: %w (output: %s)", tag, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// forcePushTagCmd is the factory for git push --force of a tag. Replaceable in tests.
var forcePushTagCmd = func(tag string) *exec.Cmd {
	return exec.Command("git", "push", "--force", "origin", tag)
}

// ForcePushTag force-pushes a local tag to origin. If the push is rejected due
// to release immutability, it returns ErrPushImmutable (use errors.Is to check).
func ForcePushTag(tag string) error {
	cmd := forcePushTagCmd(tag)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		errText := stderr.String()
		if strings.Contains(errText, "GH013") || strings.Contains(errText, "Repository rule violations") {
			return ErrPushImmutable
		}
		return fmt.Errorf("force-pushing tag %s: %w (output: %s)", tag, err, strings.TrimSpace(errText))
	}
	return nil
}
