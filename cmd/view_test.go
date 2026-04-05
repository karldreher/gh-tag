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
