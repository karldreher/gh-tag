package lib

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// TestParseVersion exercises ParseVersion across the full range of valid,
// malformed, and edge-case tag strings.
func TestParseVersion(t *testing.T) {
	tests := []struct {
		name          string
		tag           string
		prefix        string
		wantMaj       int
		wantMin       int
		wantPat       int
		wantOK        bool
	}{
		// Happy path
		{"standard", "v1.2.3", "v", 1, 2, 3, true},
		{"zeros", "v0.0.0", "v", 0, 0, 0, true},
		{"large numbers", "v10.20.30", "v", 10, 20, 30, true},
		{"very large", "v100.200.300", "v", 100, 200, 300, true},
		{"custom prefix", "release-1.2.3", "release-", 1, 2, 3, true},
		{"empty prefix", "1.2.3", "", 1, 2, 3, true},
		{"single digit each", "v1.0.0", "v", 1, 0, 0, true},
		{"patch only increment", "v0.0.1", "v", 0, 0, 1, true},
		{"minor only", "v0.1.0", "v", 0, 1, 0, true},
		{"two-digit minor", "v1.10.0", "v", 1, 10, 0, true},
		{"two-digit patch", "v1.0.10", "v", 1, 0, 10, true},
		{"numeric ordering edge", "v1.0.9", "v", 1, 0, 9, true},
		{"numeric ordering edge 10", "v1.0.10", "v", 1, 0, 10, true},

		// Wrong or missing prefix
		{"wrong prefix", "v1.2.3", "release-", 0, 0, 0, false},
		{"no prefix when expected", "1.2.3", "v", 0, 0, 0, false},
		{"partial prefix match", "va1.2.3", "v", 0, 0, 0, false}, // rest="a1.2.3"; "a1" fails digit check
		{"prefix only no version", "v", "v", 0, 0, 0, false},
		{"prefix longer than tag", "release-", "release-1.0.0", 0, 0, 0, false},

		// Component count
		{"too few parts", "v1.2", "v", 0, 0, 0, false},
		{"too many parts", "v1.2.3.4", "v", 0, 0, 0, false},
		{"single part", "v1", "v", 0, 0, 0, false},
		{"empty after prefix", "v", "v", 0, 0, 0, false},

		// Non-numeric components
		{"alpha in patch", "v1.2.abc", "v", 0, 0, 0, false},
		{"alpha in minor", "v1.abc.3", "v", 0, 0, 0, false},
		{"alpha in major", "vabc.2.3", "v", 0, 0, 0, false},
		{"mixed alphanum patch", "v1.2.3a", "v", 0, 0, 0, false},
		{"mixed alphanum major", "v1a.2.3", "v", 0, 0, 0, false},

		// Pre-release suffixes — must be rejected
		{"pre-release dash", "v1.0.0-beta", "v", 0, 0, 0, false},
		{"pre-release alpha", "v1.0.0-alpha.1", "v", 0, 0, 0, false},
		{"pre-release rc", "v1.0.0-rc1", "v", 0, 0, 0, false},
		{"pre-release dot", "v1.0.0.beta", "v", 0, 0, 0, false},

		// Annotated tag dereferences — must be rejected
		{"annotated deref", "v1.0.0^{}", "v", 0, 0, 0, false},
		{"annotated deref caret", "v1.2.3^", "v", 0, 0, 0, false},

		// Sign characters
		{"plus sign in major", "v+1.2.3", "v", 0, 0, 0, false},
		{"minus sign in major", "v-1.2.3", "v", 0, 0, 0, false},
		{"plus sign in patch", "v1.2.+3", "v", 0, 0, 0, false},

		// Empty components
		{"empty major", "v.2.3", "v", 0, 0, 0, false},
		{"empty minor", "v1..3", "v", 0, 0, 0, false},
		{"empty patch", "v1.2.", "v", 0, 0, 0, false},
		{"all empty", "v..", "v", 0, 0, 0, false},

		// Edge: empty string
		{"empty tag", "", "v", 0, 0, 0, false},
		{"empty tag empty prefix", "", "", 0, 0, 0, false},

		// Whitespace — must be rejected
		{"whitespace in patch", "v1.2. 3", "v", 0, 0, 0, false},
		{"trailing newline", "v1.2.3\n", "v", 0, 0, 0, false},
		{"leading space", " v1.2.3", "v", 0, 0, 0, false},

		// Dot-only separators in weird positions
		{"dots only", "v...", "v", 0, 0, 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			maj, min, pat, ok := ParseVersion(tc.tag, tc.prefix)
			if ok != tc.wantOK {
				t.Errorf("ParseVersion(%q, %q) ok=%v, want %v", tc.tag, tc.prefix, ok, tc.wantOK)
				return
			}
			if ok && (maj != tc.wantMaj || min != tc.wantMin || pat != tc.wantPat) {
				t.Errorf("ParseVersion(%q, %q) = %d.%d.%d, want %d.%d.%d",
					tc.tag, tc.prefix, maj, min, pat, tc.wantMaj, tc.wantMin, tc.wantPat)
			}
		})
	}
}

// TestParseVersion_NonIntegerMatrix injects non-integer values into each
// component position (major, minor, patch) while the other two hold valid
// integers, and asserts ok=false for every combination.
func TestParseVersion_NonIntegerMatrix(t *testing.T) {
	nonIntValues := []string{
		"abc",  // pure alpha
		"1a",   // trailing alpha
		"a1",   // leading alpha
		"1-0",  // hyphen (pre-release style within a component)
		"!",    // special character
		" ",    // whitespace
		"",     // empty component
	}

	for _, bad := range nonIntValues {
		tests := []struct {
			name string
			tag  string
		}{
			{
				name: fmt.Sprintf("major=%q/minor=int/patch=int", bad),
				tag:  fmt.Sprintf("v%s.1.2", bad),
			},
			{
				name: fmt.Sprintf("major=int/minor=%q/patch=int", bad),
				tag:  fmt.Sprintf("v0.%s.2", bad),
			},
			{
				name: fmt.Sprintf("major=int/minor=int/patch=%q", bad),
				tag:  fmt.Sprintf("v0.1.%s", bad),
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				_, _, _, ok := ParseVersion(tc.tag, "v")
				if ok {
					t.Errorf("ParseVersion(%q) returned ok=true, want false", tc.tag)
				}
			})
		}
	}
}

// TestFindLatestTag verifies that the highest semantic version is selected
// correctly, including numeric (not lexicographic) ordering and mixed-validity
// inputs where invalid tags are silently skipped.
func TestFindLatestTag(t *testing.T) {
	tests := []struct {
		name      string
		tags      []string
		prefix    string
		wantMaj   int
		wantMin   int
		wantPat   int
		wantFound bool
	}{
		// Empty / no valid tags
		{"empty slice", []string{}, "v", 0, 0, 0, false},
		{"nil slice", nil, "v", 0, 0, 0, false},
		{"all junk", []string{"junk", "nonsense", "latest"}, "v", 0, 0, 0, false},
		{"wrong prefix only", []string{"release-1.0.0", "release-2.0.0"}, "v", 0, 0, 0, false},

		// Single valid tag
		{"single tag", []string{"v1.0.0"}, "v", 1, 0, 0, true},
		{"single zero tag", []string{"v0.0.0"}, "v", 0, 0, 0, true},

		// Multiple tags — pick highest
		{"pick highest minor", []string{"v1.0.0", "v1.2.0", "v1.1.0"}, "v", 1, 2, 0, true},
		{"pick highest major", []string{"v1.0.0", "v2.0.0", "v1.9.9"}, "v", 2, 0, 0, true},
		{"pick highest patch", []string{"v1.0.0", "v1.0.2", "v1.0.1"}, "v", 1, 0, 2, true},

		// Numeric ordering — v1.0.10 must beat v1.0.9 (would fail lexicographic sort)
		{"numeric patch order", []string{"v1.0.9", "v1.0.10"}, "v", 1, 0, 10, true},
		{"numeric minor order", []string{"v1.9.0", "v1.10.0"}, "v", 1, 10, 0, true},
		{"numeric major order", []string{"v9.0.0", "v10.0.0"}, "v", 10, 0, 0, true},

		// Mix of valid and invalid — invalid silently skipped
		{"mixed with junk", []string{"v1.0.0", "junk", "v0.9.0"}, "v", 1, 0, 0, true},
		{"mixed with pre-release", []string{"v1.0.0", "v2.0.0-beta", "v1.9.9"}, "v", 1, 9, 9, true},
		{"mixed with annotated deref", []string{"v1.0.0", "v1.0.0^{}", "v0.9.0"}, "v", 1, 0, 0, true},
		{"annotated pairs only", []string{"v1.0.0", "v1.0.0^{}"}, "v", 1, 0, 0, true},

		// Prefix variations
		{"custom prefix found", []string{"release-1.2.3", "release-0.9.0"}, "release-", 1, 2, 3, true},
		{"custom prefix not found", []string{"v1.2.3"}, "release-", 0, 0, 0, false},
		{"empty prefix", []string{"1.2.3", "0.9.0"}, "", 1, 2, 3, true},

		// Tags given in reverse order — result must still be highest
		{"reverse order", []string{"v3.0.0", "v2.0.0", "v1.0.0"}, "v", 3, 0, 0, true},
		{"single descending", []string{"v0.0.1"}, "v", 0, 0, 1, true},

		// All same version
		{"duplicates", []string{"v1.0.0", "v1.0.0", "v1.0.0"}, "v", 1, 0, 0, true},

		// Long version history stress
		{"many versions", []string{
			"v0.1.0", "v0.2.0", "v0.9.0", "v1.0.0", "v1.0.1",
			"v1.1.0", "v1.9.0", "v1.10.0", "v2.0.0", "v2.0.1",
		}, "v", 2, 0, 1, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			maj, min, pat, found := FindLatestTag(tc.tags, tc.prefix)
			if found != tc.wantFound {
				t.Errorf("FindLatestTag found=%v, want %v (tags=%v)", found, tc.wantFound, tc.tags)
				return
			}
			if found && (maj != tc.wantMaj || min != tc.wantMin || pat != tc.wantPat) {
				t.Errorf("FindLatestTag = %d.%d.%d, want %d.%d.%d (tags=%v)",
					maj, min, pat, tc.wantMaj, tc.wantMin, tc.wantPat, tc.tags)
			}
		})
	}
}

// TestBumpVersion verifies that each bump type increments the correct component
// and resets lower components to zero.
func TestBumpVersion(t *testing.T) {
	tests := []struct {
		name     string
		maj, min, pat int
		bumpType string
		wantMaj, wantMin, wantPat int
	}{
		{"patch", 1, 2, 3, "patch", 1, 2, 4},
		{"minor", 1, 2, 3, "minor", 1, 3, 0},
		{"major", 1, 2, 3, "major", 2, 0, 0},
		{"patch from zero", 0, 0, 0, "patch", 0, 0, 1},
		{"minor from zero", 0, 0, 0, "minor", 0, 1, 0},
		{"major from zero", 0, 0, 0, "major", 1, 0, 0},
		{"minor resets patch", 1, 9, 9, "minor", 1, 10, 0},
		{"major resets minor+patch", 1, 9, 9, "major", 2, 0, 0},
		{"patch carry edge", 1, 0, 9, "patch", 1, 0, 10},
		{"minor carry edge", 1, 9, 0, "minor", 1, 10, 0},
		{"unknown bumpType unchanged", 1, 2, 3, "unknown", 1, 2, 3},
		{"empty bumpType unchanged", 1, 2, 3, "", 1, 2, 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			maj, min, pat := BumpVersion(tc.maj, tc.min, tc.pat, tc.bumpType)
			if maj != tc.wantMaj || min != tc.wantMin || pat != tc.wantPat {
				t.Errorf("BumpVersion(%d,%d,%d,%q) = %d.%d.%d, want %d.%d.%d",
					tc.maj, tc.min, tc.pat, tc.bumpType,
					maj, min, pat,
					tc.wantMaj, tc.wantMin, tc.wantPat)
			}
		})
	}
}

// TestFormatTag verifies that version components are serialised with the correct
// prefix and dot-separated format.
func TestFormatTag(t *testing.T) {
	tests := []struct {
		prefix         string
		maj, min, pat  int
		want           string
	}{
		{"v", 1, 2, 3, "v1.2.3"},
		{"v", 0, 0, 0, "v0.0.0"},
		{"", 1, 2, 3, "1.2.3"},
		{"release-", 0, 1, 0, "release-0.1.0"},
		{"v", 10, 20, 30, "v10.20.30"},
	}
	for _, tc := range tests {
		got := FormatTag(tc.prefix, tc.maj, tc.min, tc.pat)
		if got != tc.want {
			t.Errorf("FormatTag(%q,%d,%d,%d) = %q, want %q",
				tc.prefix, tc.maj, tc.min, tc.pat, got, tc.want)
		}
	}
}

// TestListRemoteTags_Empty verifies that an empty git ls-remote output returns
// a nil slice without error.
func TestListRemoteTags_Empty(t *testing.T) {
	old := listRemoteTagsCmd
	defer func() { listRemoteTagsCmd = old }()
	listRemoteTagsCmd = func() *exec.Cmd { return exec.Command("true") }

	tags, err := ListRemoteTags()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("expected empty slice, got %v", tags)
	}
}

// TestListRemoteTags_LightweightTags verifies that standard lightweight tag refs
// are parsed into bare tag names.
func TestListRemoteTags_LightweightTags(t *testing.T) {
	old := listRemoteTagsCmd
	defer func() { listRemoteTagsCmd = old }()

	output := "abc123\trefs/tags/v1.0.0\ndef456\trefs/tags/v2.0.0\n"
	listRemoteTagsCmd = func() *exec.Cmd {
		return exec.Command("printf", output)
	}

	tags, err := ListRemoteTags()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d: %v", len(tags), tags)
	}
	if tags[0] != "v1.0.0" || tags[1] != "v2.0.0" {
		t.Errorf("unexpected tags: %v", tags)
	}
}

// TestListRemoteTags_AnnotatedTagDedup verifies that the peeled '^{}' ref for
// an annotated tag is stripped, yielding only the plain name without duplication.
func TestListRemoteTags_AnnotatedTagDedup(t *testing.T) {
	old := listRemoteTagsCmd
	defer func() { listRemoteTagsCmd = old }()

	// Annotated tags produce both the plain ref and a "^{}" peeled ref.
	// We must return only the plain name, not a duplicate.
	output := "abc123\trefs/tags/v1.0.0\nxyz789\trefs/tags/v1.0.0^{}\n"
	listRemoteTagsCmd = func() *exec.Cmd {
		return exec.Command("printf", output)
	}

	tags, err := ListRemoteTags()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("expected exactly 1 tag (deduped), got %d: %v", len(tags), tags)
	}
	if tags[0] != "v1.0.0" {
		t.Errorf("expected v1.0.0, got %q", tags[0])
	}
}

// TestListRemoteTags_MixedAnnotatedAndLightweight verifies that annotated and
// lightweight tags can coexist in ls-remote output without duplication.
func TestListRemoteTags_MixedAnnotatedAndLightweight(t *testing.T) {
	old := listRemoteTagsCmd
	defer func() { listRemoteTagsCmd = old }()

	// v1.0.0 is annotated (plain + peeled); v2.0.0 is lightweight (plain only).
	output := strings.Join([]string{
		"abc123\trefs/tags/v1.0.0",
		"def456\trefs/tags/v1.0.0^{}",
		"ghi789\trefs/tags/v2.0.0",
	}, "\n") + "\n"
	listRemoteTagsCmd = func() *exec.Cmd {
		return exec.Command("printf", output)
	}

	tags, err := ListRemoteTags()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d: %v", len(tags), tags)
	}
	if tags[0] != "v1.0.0" || tags[1] != "v2.0.0" {
		t.Errorf("unexpected tags: %v", tags)
	}
}

// TestListRemoteTags_CommandFailure verifies that a non-zero git exit code
// surfaces an error containing the expected message prefix.
func TestListRemoteTags_CommandFailure(t *testing.T) {
	old := listRemoteTagsCmd
	defer func() { listRemoteTagsCmd = old }()
	listRemoteTagsCmd = func() *exec.Cmd { return exec.Command("false") }

	_, err := ListRemoteTags()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "running git ls-remote") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestListRemoteTags_MalformedLines verifies that lines without a tab separator
// or with non-tag refs (e.g. refs/heads/) are silently skipped.
func TestListRemoteTags_MalformedLines(t *testing.T) {
	old := listRemoteTagsCmd
	defer func() { listRemoteTagsCmd = old }()

	// Lines without a tab, or with a non-tags ref, should be skipped gracefully.
	output := strings.Join([]string{
		"notatabseparatedline",
		"abc123\trefs/heads/main",
		"def456\trefs/tags/v1.0.0",
	}, "\n") + "\n"
	listRemoteTagsCmd = func() *exec.Cmd {
		return exec.Command("printf", output)
	}

	tags, err := ListRemoteTags()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 1 || tags[0] != "v1.0.0" {
		t.Errorf("expected [v1.0.0], got %v", tags)
	}
}

// TestCreateLocalTag_Mock_Success verifies that a successful git command
// produces no error.
func TestCreateLocalTag_Mock_Success(t *testing.T) {
	old := createTagCmd
	defer func() { createTagCmd = old }()
	createTagCmd = func(tag string) *exec.Cmd { return exec.Command("true") }

	if err := CreateLocalTag("v1.0.0"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestCreateLocalTag_Mock_Failure verifies that a failing git command surfaces
// an error containing the expected message prefix.
func TestCreateLocalTag_Mock_Failure(t *testing.T) {
	old := createTagCmd
	defer func() { createTagCmd = old }()
	createTagCmd = func(tag string) *exec.Cmd { return exec.Command("false") }

	err := CreateLocalTag("v1.0.0")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "creating tag") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestPushTag_Mock_Success verifies that a successful push produces no error.
func TestPushTag_Mock_Success(t *testing.T) {
	old := pushTagCmd
	defer func() { pushTagCmd = old }()
	pushTagCmd = func(tag string) *exec.Cmd { return exec.Command("true") }

	if err := PushTag("v1.0.0"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestPushTag_Mock_Failure verifies that a failing push surfaces an error
// containing the expected message prefix.
func TestPushTag_Mock_Failure(t *testing.T) {
	old := pushTagCmd
	defer func() { pushTagCmd = old }()
	pushTagCmd = func(tag string) *exec.Cmd { return exec.Command("false") }

	err := PushTag("v1.0.0")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "pushing tag") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// initTempRepo creates a temporary git repository with a single empty commit.
// The directory is automatically removed when the test ends.
func initTempRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	commands := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "initial"},
	}
	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup %v failed: %v\n%s", args, err, out)
		}
	}
	return dir
}

// TestCreateLocalTag_Integration creates a tag in a real temporary repository
// and confirms it appears in git tag output.
func TestCreateLocalTag_Integration(t *testing.T) {
	dir := initTempRepo(t)

	old := createTagCmd
	defer func() { createTagCmd = old }()
	createTagCmd = func(tag string) *exec.Cmd {
		cmd := exec.Command("git", "tag", tag)
		cmd.Dir = dir
		return cmd
	}

	if err := CreateLocalTag("v1.0.0"); err != nil {
		t.Fatalf("CreateLocalTag failed: %v", err)
	}

	// Verify the tag exists in the repo.
	out, err := exec.Command("git", "-C", dir, "tag", "--list", "v1.0.0").Output()
	if err != nil {
		t.Fatalf("git tag --list failed: %v", err)
	}
	if strings.TrimSpace(string(out)) != "v1.0.0" {
		t.Errorf("expected tag v1.0.0 to exist, got: %q", string(out))
	}
}

// TestCreateLocalTag_Integration_Duplicate verifies that creating a tag that
// already exists returns an error.
func TestCreateLocalTag_Integration_Duplicate(t *testing.T) {
	dir := initTempRepo(t)

	old := createTagCmd
	defer func() { createTagCmd = old }()
	createTagCmd = func(tag string) *exec.Cmd {
		cmd := exec.Command("git", "tag", tag)
		cmd.Dir = dir
		return cmd
	}

	if err := CreateLocalTag("v1.0.0"); err != nil {
		t.Fatalf("first CreateLocalTag failed: %v", err)
	}
	if err := CreateLocalTag("v1.0.0"); err == nil {
		t.Fatal("expected error for duplicate tag, got nil")
	}
}

// TestCreateLocalTag_Integration_MultipleVersions verifies that multiple
// distinct tags can be created in a single repository.
func TestCreateLocalTag_Integration_MultipleVersions(t *testing.T) {
	dir := initTempRepo(t)

	old := createTagCmd
	defer func() { createTagCmd = old }()
	createTagCmd = func(tag string) *exec.Cmd {
		cmd := exec.Command("git", "tag", tag)
		cmd.Dir = dir
		return cmd
	}

	for _, tag := range []string{"v1.0.0", "v1.1.0", "v2.0.0"} {
		if err := CreateLocalTag(tag); err != nil {
			t.Fatalf("CreateLocalTag(%q) failed: %v", tag, err)
		}
	}

	out, err := exec.Command("git", "-C", dir, "tag", "--list").Output()
	if err != nil {
		t.Fatalf("git tag --list failed: %v", err)
	}
	listed := strings.Fields(string(out))
	if len(listed) != 3 {
		t.Errorf("expected 3 tags, got: %v", listed)
	}
}

// TestCreateLocalTag_Integration_CustomPrefix verifies that tags with a non-v
// prefix are created and stored correctly in the repository.
func TestCreateLocalTag_Integration_CustomPrefix(t *testing.T) {
	dir := initTempRepo(t)

	old := createTagCmd
	defer func() { createTagCmd = old }()
	createTagCmd = func(tag string) *exec.Cmd {
		cmd := exec.Command("git", "tag", tag)
		cmd.Dir = dir
		return cmd
	}

	if err := CreateLocalTag("release-1.0.0"); err != nil {
		t.Fatalf("CreateLocalTag failed: %v", err)
	}

	out, err := exec.Command("git", "-C", dir, "tag", "--list", "release-1.0.0").Output()
	if err != nil {
		t.Fatalf("git tag --list failed: %v", err)
	}
	if strings.TrimSpace(string(out)) != "release-1.0.0" {
		t.Errorf("expected tag release-1.0.0 to exist, got: %q", string(out))
	}
}

// TestParseVersion_PartialPrefixMatch verifies that a tag beginning with the
// prefix characters but followed by non-digit content is correctly rejected.
func TestParseVersion_PartialPrefixMatch(t *testing.T) {
	// "va1.2.3" has prefix "v", rest="a1.2.3"; "a1" contains non-digit 'a'.
	_, _, _, ok := ParseVersion("va1.2.3", "v")
	if ok {
		t.Error("expected ParseVersion(va1.2.3, v) to return ok=false")
	}
}

// TestFindLatestTag_OnlyAnnotatedDerefs verifies that a slice containing only
// "^{}" dereference entries (no plain tag names) yields found=false.
func TestFindLatestTag_OnlyAnnotatedDerefs(t *testing.T) {
	tags := []string{"v1.0.0^{}", "v2.0.0^{}"}
	_, _, _, found := FindLatestTag(tags, "v")
	if found {
		t.Error("expected found=false for slice of only annotated derefs")
	}
}

// TestFindLatestTag_ZeroValuesAreValid verifies that v0.0.0 is a legitimate
// result, not confused with the zero-value of a not-found result.
func TestFindLatestTag_ZeroValuesAreValid(t *testing.T) {
	tags := []string{"v0.0.0"}
	maj, min, pat, found := FindLatestTag(tags, "v")
	if !found {
		t.Fatal("expected found=true for v0.0.0")
	}
	if maj != 0 || min != 0 || pat != 0 {
		t.Errorf("expected 0.0.0, got %d.%d.%d", maj, min, pat)
	}
}

// TestListRemoteTags_TagsNeverReturnsTrailingNewlineInName verifies that
// tag names do not have trailing whitespace or newlines.
func TestListRemoteTags_TagsNeverReturnsTrailingNewlineInName(t *testing.T) {
	old := listRemoteTagsCmd
	defer func() { listRemoteTagsCmd = old }()

	output := "abc123\trefs/tags/v1.0.0\n"
	listRemoteTagsCmd = func() *exec.Cmd {
		return exec.Command("printf", output)
	}

	tags, err := ListRemoteTags()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(tags))
	}
	if tags[0] != "v1.0.0" {
		t.Errorf("tag name contains unexpected characters: %q", tags[0])
	}
}

// TestCreateLocalTag_Integration_AfterInitialCommit verifies that a tag can be
// created on a repository that has only a single initial commit.
func TestCreateLocalTag_Integration_AfterInitialCommit(t *testing.T) {
	dir := initTempRepo(t)

	// Verify the repo has exactly one commit.
	out, err := exec.Command("git", "-C", dir, "log", "--oneline").Output()
	if err != nil {
		t.Fatalf("git log failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(lines))
	}

	old := createTagCmd
	defer func() { createTagCmd = old }()
	createTagCmd = func(tag string) *exec.Cmd {
		cmd := exec.Command("git", "tag", tag)
		cmd.Dir = dir
		return cmd
	}

	if err := CreateLocalTag("v0.1.0"); err != nil {
		t.Fatalf("CreateLocalTag on fresh repo failed: %v", err)
	}
}

// TestListRemoteTags_LargeOutput verifies that many tags are all parsed
// correctly without truncation or off-by-one errors.
func TestListRemoteTags_LargeOutput(t *testing.T) {
	old := listRemoteTagsCmd
	defer func() { listRemoteTagsCmd = old }()

	var lines []string
	for i := range 50 {
		lines = append(lines, "abc123\trefs/tags/v1.0."+strconv.Itoa(i))
	}
	output := strings.Join(lines, "\n") + "\n"
	listRemoteTagsCmd = func() *exec.Cmd {
		return exec.Command("printf", output)
	}

	tags, err := ListRemoteTags()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 50 {
		t.Errorf("expected 50 tags, got %d", len(tags))
	}
}

// TestResolveTagRef_Success verifies that a full SHA returned by git is
// truncated to 7 characters.
func TestResolveTagRef_Success(t *testing.T) {
	old := resolveTagRefCmd
	defer func() { resolveTagRefCmd = old }()
	resolveTagRefCmd = func(tag string) *exec.Cmd {
		return exec.Command("printf", "abc1234def5678901234567890123456789012345\n")
	}

	sha, err := ResolveTagRef("v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sha != "abc1234" {
		t.Errorf("got %q, want %q", sha, "abc1234")
	}
}

// TestResolveTagRef_Failure verifies that a failing command surfaces an error
// with the expected prefix.
func TestResolveTagRef_Failure(t *testing.T) {
	old := resolveTagRefCmd
	defer func() { resolveTagRefCmd = old }()
	resolveTagRefCmd = func(tag string) *exec.Cmd { return exec.Command("false") }

	_, err := ResolveTagRef("v1.0.0")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "resolving ref for tag") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestResolveHead_Success verifies that HEAD's full SHA is truncated to 7
// characters.
func TestResolveHead_Success(t *testing.T) {
	old := resolveHeadCmd
	defer func() { resolveHeadCmd = old }()
	resolveHeadCmd = func() *exec.Cmd {
		return exec.Command("printf", "def5678901234567890123456789012345678901\n")
	}

	sha, err := ResolveHead()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sha != "def5678" {
		t.Errorf("got %q, want %q", sha, "def5678")
	}
}

// TestResolveHead_Failure verifies that a failing command surfaces an error
// with the expected message.
func TestResolveHead_Failure(t *testing.T) {
	old := resolveHeadCmd
	defer func() { resolveHeadCmd = old }()
	resolveHeadCmd = func() *exec.Cmd { return exec.Command("false") }

	_, err := ResolveHead()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "resolving HEAD") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestOverwriteTag_Success verifies that a successful git tag -f produces no
// error.
func TestOverwriteTag_Success(t *testing.T) {
	old := overwriteTagCmd
	defer func() { overwriteTagCmd = old }()
	overwriteTagCmd = func(tag string) *exec.Cmd { return exec.Command("true") }

	if err := OverwriteTag("v1.0.0"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestOverwriteTag_Failure verifies that a failing git tag -f surfaces an error
// containing the expected message prefix.
func TestOverwriteTag_Failure(t *testing.T) {
	old := overwriteTagCmd
	defer func() { overwriteTagCmd = old }()
	overwriteTagCmd = func(tag string) *exec.Cmd { return exec.Command("false") }

	err := OverwriteTag("v1.0.0")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "overwriting tag") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestForcePushTag_Success verifies that a successful force-push produces no
// error.
func TestForcePushTag_Success(t *testing.T) {
	old := forcePushTagCmd
	defer func() { forcePushTagCmd = old }()
	forcePushTagCmd = func(tag string) *exec.Cmd { return exec.Command("true") }

	if err := ForcePushTag("v1.0.0"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestForcePushTag_Immutability verifies that a GH013 rejection from the remote
// is surfaced as ErrPushImmutable rather than a generic error.
func TestForcePushTag_Immutability(t *testing.T) {
	old := forcePushTagCmd
	defer func() { forcePushTagCmd = old }()
	forcePushTagCmd = func(tag string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo 'remote: error: GH013: Repository rule violations found' >&2; exit 1")
	}

	err := ForcePushTag("v1.0.0")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrPushImmutable) {
		t.Errorf("expected ErrPushImmutable, got: %v", err)
	}
}

// TestForcePushTag_OtherFailure verifies that a generic push failure returns an
// error that is not ErrPushImmutable.
func TestForcePushTag_OtherFailure(t *testing.T) {
	old := forcePushTagCmd
	defer func() { forcePushTagCmd = old }()
	forcePushTagCmd = func(tag string) *exec.Cmd { return exec.Command("false") }

	err := ForcePushTag("v1.0.0")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, ErrPushImmutable) {
		t.Error("generic failure should not return ErrPushImmutable")
	}
	if !strings.Contains(err.Error(), "force-pushing tag") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestHasTagsWithDifferentPrefix covers the onboarding mismatch scenario: the
// remote has tags but none match the configured prefix. Without this detection
// the tool would silently treat the repo as untagged and create an unrelated
// tag series.
func TestHasTagsWithDifferentPrefix(t *testing.T) {
	tests := []struct {
		name   string
		tags   []string
		prefix string
		want   bool
	}{
		// No tags at all — not a mismatch, just a new repo.
		{"no tags", []string{}, "v", false},
		{"nil tags", nil, "v", false},

		// Tags match the prefix — no mismatch.
		{"matching prefix v", []string{"v1.0.0", "v1.1.0"}, "v", false},
		{"matching prefix release-", []string{"release-1.0.0", "release-2.0.0"}, "release-", false},
		{"matching empty prefix", []string{"1.0.0", "2.0.0"}, "", false},
		{"single matching tag", []string{"v1.0.0"}, "v", false},

		// Tags exist but NONE match the configured prefix — mismatch.
		{"release tags, v prefix", []string{"release-1.0.0", "release-2.0.0"}, "v", true},
		{"v tags, release- prefix", []string{"v1.0.0", "v2.0.0"}, "release-", true},
		{"v tags, empty prefix configured",
			// If user blanked their prefix but remote uses "v", non-semver bare
			// strings like "v1.0.0" won't parse with prefix="" so it's a mismatch.
			[]string{"v1.0.0"}, "", true},
		{"single mismatched tag", []string{"release-1.0.0"}, "v", true},

		// Mix: some matching tags among non-matching — NOT a mismatch (found=true).
		{"mixed: some match", []string{"release-1.0.0", "v2.0.0"}, "v", false},
		{"mixed: annotated deref of matching tag",
			[]string{"v1.0.0", "v1.0.0^{}"}, "v", false},

		// Only non-semver garbage on remote — repo has tags, none parse.
		{"junk tags only", []string{"latest", "stable", "main"}, "v", true},
		{"pre-release tags only", []string{"v1.0.0-beta", "v2.0.0-rc1"}, "v", true},

		// Annotated-deref only (would happen if ListRemoteTags had a bug;
		// but HasTagsWithDifferentPrefix should handle it gracefully).
		{"only annotated derefs", []string{"v1.0.0^{}", "v2.0.0^{}"}, "v", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := HasTagsWithDifferentPrefix(tc.tags, tc.prefix)
			if got != tc.want {
				t.Errorf("HasTagsWithDifferentPrefix(%v, %q) = %v, want %v",
					tc.tags, tc.prefix, got, tc.want)
			}
		})
	}
}

// TestParseBumpType verifies that all documented input aliases map to the
// correct canonical bump type, and that invalid inputs return an error.
func TestParseBumpType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		// Single-char keys (case-sensitive)
		{"M → major", "M", "major", false},
		{"m → minor", "m", "minor", false},
		{"p → patch", "p", "patch", false},
		{"P → patch", "P", "patch", false},

		// Full word inputs
		{"major word", "major", "major", false},
		{"minor word", "minor", "minor", false},
		{"patch word", "patch", "patch", false},
		{"Major capitalized", "Major", "major", false},
		{"Minor capitalized", "Minor", "minor", false},
		{"Patch capitalized", "Patch", "patch", false},

		// Invalid inputs
		{"empty", "", "", true},
		{"x invalid", "x", "", true},
		{"MAJOR uppercase", "MAJOR", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseBumpType(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseBumpType(%q) err=%v, wantErr=%v", tc.input, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestSortTags verifies that tags are filtered to valid semver entries and
// ordered numerically (not lexicographically) in both ascending and descending
// modes.
func TestSortTags(t *testing.T) {
	tests := []struct {
		name      string
		tags      []string
		prefix    string
		ascending bool
		want      []string
	}{
		{
			name:   "nil input returns empty slice",
			tags:   nil,
			prefix: "v",
			want:   []string{},
		},
		{
			name:   "empty input returns empty slice",
			tags:   []string{},
			prefix: "v",
			want:   []string{},
		},
		{
			name:   "invalid-only tags returns empty slice",
			tags:   []string{"not-a-tag", "v1.0", "v1.0.0-beta"},
			prefix: "v",
			want:   []string{},
		},
		{
			name:   "single valid tag",
			tags:   []string{"v1.2.3"},
			prefix: "v",
			want:   []string{"v1.2.3"},
		},
		{
			name:   "multiple tags descending",
			tags:   []string{"v1.0.0", "v2.0.0", "v1.2.0"},
			prefix: "v",
			want:   []string{"v2.0.0", "v1.2.0", "v1.0.0"},
		},
		{
			name:      "multiple tags ascending",
			tags:      []string{"v2.0.0", "v1.0.0", "v1.2.0"},
			prefix:    "v",
			ascending: true,
			want:      []string{"v1.0.0", "v1.2.0", "v2.0.0"},
		},
		{
			name:   "numeric ordering not lexicographic descending",
			tags:   []string{"v1.0.9", "v1.0.10", "v1.0.2"},
			prefix: "v",
			want:   []string{"v1.0.10", "v1.0.9", "v1.0.2"},
		},
		{
			name:   "mixed valid and invalid drops invalid",
			tags:   []string{"v1.0.0", "not-valid", "v2.0.0", "v1.0.0-beta"},
			prefix: "v",
			want:   []string{"v2.0.0", "v1.0.0"},
		},
		{
			name:   "custom prefix only matches prefix",
			tags:   []string{"release-1.0.0", "v1.0.0", "release-2.0.0"},
			prefix: "release-",
			want:   []string{"release-2.0.0", "release-1.0.0"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SortTags(tc.tags, tc.prefix, tc.ascending)
			if len(got) != len(tc.want) {
				t.Fatalf("SortTags() len=%d, want %d; got=%v", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("SortTags()[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestExtractTagPrefix verifies that ExtractTagPrefix correctly identifies the
// non-numeric prefix from tags ending in a semver suffix.
func TestExtractTagPrefix(t *testing.T) {
	tests := []struct {
		tag        string
		wantPrefix string
		wantOK     bool
	}{
		{"v1.2.3", "v", true},
		{"v0.0.0", "v", true},
		{"1.2.3", "", true},
		{"release-1.2.3", "release-", true},
		{"my-project-v1.2.3", "my-project-v", true},
		{"another-project-v2.0.0", "another-project-v", true},

		{"not-semver", "", false},
		{"", "", false},
		{"v1.2", "", false},
		{"v1", "", false},
		{"v1.2.3-beta", "", false},
		{"v1.0.0^{}", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.tag, func(t *testing.T) {
			prefix, ok := ExtractTagPrefix(tc.tag)
			if ok != tc.wantOK {
				t.Errorf("ExtractTagPrefix(%q) ok=%v, want %v", tc.tag, ok, tc.wantOK)
				return
			}
			if ok && prefix != tc.wantPrefix {
				t.Errorf("ExtractTagPrefix(%q) prefix=%q, want %q", tc.tag, prefix, tc.wantPrefix)
			}
		})
	}
}

// TestUniqueSemverPrefixes verifies that all unique semver prefixes are returned,
// deduplicated, and sorted.
func TestUniqueSemverPrefixes(t *testing.T) {
	tests := []struct {
		name string
		tags []string
		want []string
	}{
		{"empty input", []string{}, []string{}},
		{"nil input", nil, []string{}},
		{"single v prefix", []string{"v1.0.0", "v1.1.0", "v2.0.0"}, []string{"v"}},
		{
			"multiple distinct prefixes",
			[]string{"v1.0.0", "my-project-v1.0.0", "another-v2.0.0"},
			[]string{"another-v", "my-project-v", "v"},
		},
		{"deduplication", []string{"v1.0.0", "v1.1.0", "v2.0.0"}, []string{"v"}},
		{"non-semver tags skipped", []string{"latest", "stable", "v1.0.0"}, []string{"v"}},
		{"empty prefix bare semver", []string{"1.0.0", "2.0.0"}, []string{""}},
		{
			"mixed prefixes and non-semver",
			[]string{"v1.0.0", "release-1.0.0", "junk", "v2.0.0"},
			[]string{"release-", "v"},
		},
		{"pre-release tags skipped", []string{"v1.0.0-beta", "v1.0.0"}, []string{"v"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := UniqueSemverPrefixes(tc.tags)
			if len(got) != len(tc.want) {
				t.Fatalf("UniqueSemverPrefixes() = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("UniqueSemverPrefixes()[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
