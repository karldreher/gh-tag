package lib

import (
	"os/exec"
	"strings"
	"testing"
)

// gitIn runs a git command in dir and returns trimmed stdout, failing the test on error.
func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// commitIn creates an empty commit with the given subject in dir.
func commitIn(t *testing.T, dir, subject string) {
	t.Helper()
	gitIn(t, dir, "commit", "--allow-empty", "-m", subject)
}

// runAutoCmdsIn points every git factory used by the --auto path at dir and
// restores the originals when the test ends.
func runAutoCmdsIn(t *testing.T, dir string) {
	t.Helper()
	inDir := func(c *exec.Cmd) *exec.Cmd { c.Dir = dir; return c }

	oldTitles, oldRemote, oldHas, oldFetch, oldLocal, oldAnc := commitTitlesCmd, remoteTagCommitCmd, hasCommitCmd, fetchTagCmd, localTagCommitCmd, isAncestorCmd
	t.Cleanup(func() {
		commitTitlesCmd, remoteTagCommitCmd, hasCommitCmd, fetchTagCmd, localTagCommitCmd, isAncestorCmd = oldTitles, oldRemote, oldHas, oldFetch, oldLocal, oldAnc
	})
	commitTitlesCmd = func(r string) *exec.Cmd { return inDir(oldTitles(r)) }
	remoteTagCommitCmd = func(tag string) *exec.Cmd { return inDir(oldRemote(tag)) }
	hasCommitCmd = func(sha string) *exec.Cmd { return inDir(oldHas(sha)) }
	fetchTagCmd = func(tag string) *exec.Cmd { return inDir(oldFetch(tag)) }
	localTagCommitCmd = func(tag string) *exec.Cmd { return inDir(oldLocal(tag)) }
	isAncestorCmd = func(sha string) *exec.Cmd { return inDir(oldAnc(sha)) }
}

// initRepoWithOrigin creates a bare "origin" and a working clone with user
// identity configured. Returns (origin, clone).
func initRepoWithOrigin(t *testing.T) (string, string) {
	t.Helper()
	origin := t.TempDir()
	gitIn(t, origin, "init", "--bare")
	clone := t.TempDir()
	gitIn(t, clone, "clone", origin, ".")
	gitIn(t, clone, "config", "user.email", "test@example.com")
	gitIn(t, clone, "config", "user.name", "Test")
	return origin, clone
}

// TestAuto_Issue14_FeatBeforeChore reproduces https://github.com/karldreher/gh-tag/issues/14:
// a chore at HEAD after a feat since v0.1.0 must yield a minor bump.
func TestAuto_Issue14_FeatBeforeChore(t *testing.T) {
	_, dir := initRepoWithOrigin(t)
	commitIn(t, dir, "feat: new engine")
	gitIn(t, dir, "tag", "v0.1.0")
	commitIn(t, dir, "feat: example")
	commitIn(t, dir, "chore: readme update")
	gitIn(t, dir, "push", "origin", "HEAD", "v0.1.0")
	runAutoCmdsIn(t, dir)

	sha, err := RemoteTagCommit("v0.1.0")
	if err != nil {
		t.Fatalf("RemoteTagCommit: %v", err)
	}
	titles, err := CommitTitlesSince(sha)
	if err != nil {
		t.Fatalf("CommitTitlesSince: %v", err)
	}
	if len(titles) != 2 {
		t.Fatalf("titles = %q, want 2 commits since tag", titles)
	}
	bump, _, err := MaxBumpType(titles)
	if err != nil || bump != "minor" {
		t.Errorf("bump = %q, err = %v; want minor", bump, err)
	}
}

// TestCommitTitlesSince_NoBase returns full history when no tag exists.
func TestCommitTitlesSince_NoBase(t *testing.T) {
	dir := initTempRepo(t)
	commitIn(t, dir, "feat: one")
	runAutoCmdsIn(t, dir)

	titles, err := CommitTitlesSince("")
	if err != nil {
		t.Fatalf("CommitTitlesSince: %v", err)
	}
	if strings.Join(titles, "|") != "feat: one|initial" {
		t.Errorf("titles = %q", titles)
	}
}

// TestCommitTitlesSince_EmptyRange returns no titles when HEAD is the tag.
func TestCommitTitlesSince_EmptyRange(t *testing.T) {
	dir := initTempRepo(t)
	head := gitIn(t, dir, "rev-parse", "HEAD")
	runAutoCmdsIn(t, dir)

	titles, err := CommitTitlesSince(head)
	if err != nil {
		t.Fatalf("CommitTitlesSince: %v", err)
	}
	if len(titles) != 0 {
		t.Errorf("titles = %q, want none", titles)
	}
}

// TestRemoteTagCommit_AnnotatedResolvesToCommit verifies the ^{} line is used
// so annotated tags resolve to the commit, not the tag object.
func TestRemoteTagCommit_AnnotatedResolvesToCommit(t *testing.T) {
	_, dir := initRepoWithOrigin(t)
	commitIn(t, dir, "feat: a")
	gitIn(t, dir, "tag", "-a", "v1.0.0", "-m", "release")
	gitIn(t, dir, "push", "origin", "HEAD", "v1.0.0")
	runAutoCmdsIn(t, dir)

	got, err := RemoteTagCommit("v1.0.0")
	if err != nil {
		t.Fatalf("RemoteTagCommit: %v", err)
	}
	if want := gitIn(t, dir, "rev-parse", "HEAD"); got != want {
		t.Errorf("got %s, want commit %s", got, want)
	}
}

// TestRemoteTagCommit_Mock covers lightweight, annotated, and missing output.
func TestRemoteTagCommit_Mock(t *testing.T) {
	old := remoteTagCommitCmd
	t.Cleanup(func() { remoteTagCommitCmd = old })

	tests := []struct {
		name, output, want string
		wantErr            bool
	}{
		{"lightweight", "aaa\trefs/tags/v1.0.0\n", "aaa", false},
		{"annotated", "tagobj\trefs/tags/v1.0.0\nccc\trefs/tags/v1.0.0^{}\n", "ccc", false},
		{"missing", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remoteTagCommitCmd = func(string) *exec.Cmd { return exec.Command("printf", tt.output) }
			got, err := RemoteTagCommit("v1.0.0")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestEnsureCommit_FetchesWithoutCreatingLocalTag verifies that a commit only
// on origin is fetched and that no local tag ref is created or changed.
func TestEnsureCommit_FetchesWithoutCreatingLocalTag(t *testing.T) {
	origin, dir := initRepoWithOrigin(t)
	commitIn(t, dir, "feat: base")
	gitIn(t, dir, "push", "origin", "HEAD")

	// A second clone pushes a new commit and tag the first clone has never seen.
	other := t.TempDir()
	gitIn(t, other, "clone", origin, ".")
	gitIn(t, other, "config", "user.email", "o@example.com")
	gitIn(t, other, "config", "user.name", "Other")
	commitIn(t, other, "feat: remote only")
	gitIn(t, other, "tag", "v2.0.0")
	gitIn(t, other, "push", "origin", "HEAD", "v2.0.0")
	remoteSHA := gitIn(t, other, "rev-parse", "HEAD")

	runAutoCmdsIn(t, dir)
	if err := EnsureCommit(remoteSHA, "v2.0.0"); err != nil {
		t.Fatalf("EnsureCommit: %v", err)
	}
	gitIn(t, dir, "cat-file", "-e", remoteSHA+"^{commit}")
	if _, exists, _ := LocalTagCommit("v2.0.0"); exists {
		t.Error("EnsureCommit must not create a local tag ref")
	}
	// HEAD is behind the tag, so the tag is not its ancestor.
	if anc, err := IsAncestorOfHead(remoteSHA); err != nil || anc {
		t.Errorf("IsAncestorOfHead = %v, %v; want false, nil", anc, err)
	}
}

// TestLocalTagCommit_DiffersFromRemote detects a stale local tag.
func TestLocalTagCommit_DiffersFromRemote(t *testing.T) {
	_, dir := initRepoWithOrigin(t)
	commitIn(t, dir, "feat: a")
	gitIn(t, dir, "tag", "v1.0.0")
	gitIn(t, dir, "push", "origin", "HEAD", "v1.0.0")
	commitIn(t, dir, "fix: b")
	gitIn(t, dir, "tag", "-f", "v1.0.0") // local moved, remote unchanged
	runAutoCmdsIn(t, dir)

	remote, err := RemoteTagCommit("v1.0.0")
	if err != nil {
		t.Fatalf("RemoteTagCommit: %v", err)
	}
	local, exists, err := LocalTagCommit("v1.0.0")
	if err != nil || !exists {
		t.Fatalf("LocalTagCommit = %v, %v", exists, err)
	}
	if local == remote {
		t.Error("expected local and remote tag commits to differ")
	}
	if anc, err := IsAncestorOfHead(remote); err != nil || !anc {
		t.Errorf("IsAncestorOfHead = %v, %v; want true, nil", anc, err)
	}
}
