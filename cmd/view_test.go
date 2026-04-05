package cmd

import (
	"io"
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
