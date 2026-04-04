package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeReader wraps a string as a bufio.Reader for use as fake stdin.
func makeReader(input string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(input))
}

// ──────────────────────────────────────────────────────────────────────────────
// readBumpType
// ──────────────────────────────────────────────────────────────────────────────

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

func TestReadBumpType_Interactive(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		// Single-char keys (case-sensitive)
		{"M → major", "M\n", "major", false},
		{"m → minor", "m\n", "minor", false},
		{"p → patch", "p\n", "patch", false},
		{"P → patch", "P\n", "patch", false},

		// Full word inputs
		{"major word", "major\n", "major", false},
		{"minor word", "minor\n", "minor", false},
		{"patch word", "patch\n", "patch", false},
		{"Major capitalized", "Major\n", "major", false},
		{"Minor capitalized", "Minor\n", "minor", false},
		{"Patch capitalized", "Patch\n", "patch", false},

		// Invalid inputs
		{"empty", "\n", "", true},
		{"x invalid", "x\n", "", true},
		{"lowercase major misspelled", "MAJOR\n", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := makeReader(tc.input)
			got, err := readBumpType(r, false, false, false)
			if (err != nil) != tc.wantErr {
				t.Fatalf("readBumpType(%q) err=%v, wantErr=%v", tc.input, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReadBumpType_ReaderError(t *testing.T) {
	// Empty reader produces EOF before the newline delimiter, triggering the
	// ReadString error branch.
	r := makeReader("")
	_, err := readBumpType(r, false, false, false)
	if err == nil {
		t.Fatal("expected error from empty reader")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// confirmAction
// ──────────────────────────────────────────────────────────────────────────────

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

func TestConfirmAction_ReaderError(t *testing.T) {
	r := makeReader("")
	_, err := confirmAction(r, false, "prompt: ")
	if err == nil {
		t.Fatal("expected error from empty reader")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// effectivePrefix
// ──────────────────────────────────────────────────────────────────────────────

func TestEffectivePrefix(t *testing.T) {
	t.Run("no config returns v", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		got, err := effectivePrefix()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "v" {
			t.Errorf("got %q, want %q", got, "v")
		}
	})

	t.Run("config with prefix returns it", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)
		if err := os.MkdirAll(filepath.Join(tmpDir, ".gh-tag"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(
			filepath.Join(tmpDir, ".gh-tag", "config.json"),
			[]byte(`{"prefix":"release-"}`), 0644,
		); err != nil {
			t.Fatal(err)
		}
		got, err := effectivePrefix()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "release-" {
			t.Errorf("got %q, want %q", got, "release-")
		}
	})

	t.Run("load error returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)
		// config.json as a directory forces LoadConfig to fail
		if err := os.MkdirAll(filepath.Join(tmpDir, ".gh-tag", "config.json"), 0755); err != nil {
			t.Fatal(err)
		}
		_, err := effectivePrefix()
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("config with empty prefix returns v", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)
		if err := os.MkdirAll(filepath.Join(tmpDir, ".gh-tag"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(
			filepath.Join(tmpDir, ".gh-tag", "config.json"),
			[]byte(`{"prefix":""}`), 0644,
		); err != nil {
			t.Fatal(err)
		}
		got, err := effectivePrefix()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "v" {
			t.Errorf("got %q, want %q", got, "v")
		}
	})
}


