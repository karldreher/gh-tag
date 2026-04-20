package cmd

import (
	"bufio"
	"io"
	"strings"
	"testing"
)

// makeReader returns a bufio.Reader backed by the given string, for injecting
// simulated stdin in tests.
func makeReader(input string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(input))
}

// TestMutualExclusionFlags verifies that flag pairs that must not coexist are
// rejected by cobra at the CLI level.
func TestMutualExclusionFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"overwrite+major", []string{"--overwrite", "--major"}},
		{"overwrite+minor", []string{"--overwrite", "--minor"}},
		{"overwrite+patch", []string{"--overwrite", "--patch"}},
		{"major+minor", []string{"--major", "--minor"}},
		{"major+patch", []string{"--major", "--patch"}},
		{"minor+patch", []string{"--minor", "--patch"}},
		{"auto+major", []string{"--auto", "--major"}},
		{"auto+minor", []string{"--auto", "--minor"}},
		{"auto+patch", []string{"--auto", "--patch"}},
		{"auto+overwrite", []string{"--auto", "--overwrite"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRootCmd()
			cmd.SetArgs(tc.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			if err := cmd.Execute(); err == nil {
				t.Errorf("expected error for args %v, got nil", tc.args)
			}
		})
	}
}

// TestReadBumpType_Flags verifies that each boolean flag short-circuits the
// reader and returns the correct bump type without consuming stdin.
func TestReadBumpType_Flags(t *testing.T) {
	tests := []struct {
		name  string
		major bool
		minor bool
		patch bool
		want  string
	}{
		{"major flag", true, false, false, "major"},
		{"minor flag", false, true, false, "minor"},
		{"patch flag", false, false, true, "patch"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := makeReader("") // stdin not read when flags are set
			got, err := readBumpType(r, tc.major, tc.minor, tc.patch)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestReadBumpType_ReaderError verifies that an empty reader triggers an error
// when no flag is set, because ReadString reaches EOF before the newline
// delimiter.
func TestReadBumpType_ReaderError(t *testing.T) {
	r := makeReader("")
	_, err := readBumpType(r, false, false, false)
	if err == nil {
		t.Fatal("expected error from empty reader")
	}
}

// TestConfirmAction verifies that skipConfirm bypasses the reader, that y/yes
// variants (case-insensitive) confirm, and that all other inputs deny.
func TestConfirmAction(t *testing.T) {
	tests := []struct {
		name        string
		skipConfirm bool
		input       string
		want        bool
	}{
		{"skip confirm bypasses", true, "", true},
		{"y confirms", false, "y\n", true},
		{"yes confirms", false, "yes\n", true},
		{"Y confirms", false, "Y\n", true},
		{"YES confirms", false, "YES\n", true},
		{"n denies", false, "n\n", false},
		{"N denies", false, "N\n", false},
		{"no denies", false, "no\n", false},
		{"empty denies", false, "\n", false},
		{"random denies", false, "sure\n", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := makeReader(tc.input)
			got, err := confirmAction(r, tc.skipConfirm, "Proceed? [y/N]: ")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("confirmAction(skip=%v, input=%q) = %v, want %v",
					tc.skipConfirm, tc.input, got, tc.want)
			}
		})
	}
}

// TestConfirmAction_ReaderError verifies that an empty reader triggers an error
// when skipConfirm is false.
func TestConfirmAction_ReaderError(t *testing.T) {
	r := makeReader("")
	_, err := confirmAction(r, false, "prompt: ")
	if err == nil {
		t.Fatal("expected error from empty reader")
	}
}
