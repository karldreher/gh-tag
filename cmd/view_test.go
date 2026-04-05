package cmd

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestViewCmd_TooManyArgs(t *testing.T) {
	cmd := newRootCmd()
	cmd.AddCommand(newViewCmd())
	cmd.SetArgs([]string{"view", "v1.0.0", "v2.0.0"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for two args, got nil")
	}
}

func TestViewCmd_WebFlagTooManyArgs(t *testing.T) {
	// --web with two positional args is still rejected by cobra before any
	// browser call is made, so this is safe to run in CI.
	cmd := newRootCmd()
	cmd.AddCommand(newViewCmd())
	cmd.SetArgs([]string{"view", "v0.1.1", "v0.1.0", "--web"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for two args with --web, got nil")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// requireSingleTag
// ──────────────────────────────────────────────────────────────────────────────

func TestRequireSingleTag(t *testing.T) {
	tests := []struct {
		name        string
		tags        []string
		prefix      string
		wantTag     singleTag
		wantErr     bool
		wantNoTags  bool // true when error should be errNoSemverTags specifically
	}{
		{
			name:       "nil tags",
			tags:       nil,
			prefix:     "v",
			wantNoTags: true,
		},
		{
			name:       "empty tags",
			tags:       []string{},
			prefix:     "v",
			wantNoTags: true,
		},
		{
			name:       "no matching prefix",
			tags:       []string{"release-1.0.0", "release-2.0.0"},
			prefix:     "v",
			wantNoTags: true,
		},
		{
			name:       "invalid tags only",
			tags:       []string{"not-semver", "v1.0.0-beta", "v1.0"},
			prefix:     "v",
			wantNoTags: true,
		},
		{
			name:    "exactly one matching tag",
			tags:    []string{"v1.0.0"},
			prefix:  "v",
			wantTag: singleTag("v1.0.0"),
		},
		{
			name:    "one matching, others invalid",
			tags:    []string{"v1.0.0", "not-semver", "v1.0.0-beta"},
			prefix:  "v",
			wantTag: singleTag("v1.0.0"),
		},
		{
			name:    "two matching tags",
			tags:    []string{"v1.0.0", "v2.0.0"},
			prefix:  "v",
			wantErr: true,
		},
		{
			name:    "many matching tags",
			tags:    []string{"v1.0.0", "v1.1.0", "v2.0.0"},
			prefix:  "v",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := requireSingleTag(tc.tags, tc.prefix)

			if tc.wantNoTags {
				if !errors.Is(err, errNoSemverTags) {
					t.Errorf("expected errNoSemverTags, got %v", err)
				}
				return
			}

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if errors.Is(err, errNoSemverTags) {
					t.Errorf("expected non-sentinel error for >1 tags, got errNoSemverTags")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantTag {
				t.Errorf("got %q, want %q", got, tc.wantTag)
			}
		})
	}
}

func TestRequireSingleTag_ErrorMessageContent(t *testing.T) {
	// The error for >1 tags must include the count and a usage hint so the
	// user knows how to resolve it without consulting documentation.
	_, err := requireSingleTag([]string{"v1.0.0", "v2.0.0", "v3.0.0"}, "v")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	for _, want := range []string{"3", "gh tag view", "--web"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message %q missing %q", msg, want)
		}
	}
}
