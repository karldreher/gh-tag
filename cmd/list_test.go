package cmd

import (
	"io"
	"testing"
)

func TestListCmd_Flags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"no flags", []string{"list"}, false},
		{"ascending flag", []string{"list", "--ascending"}, false},
		{"descending flag", []string{"list", "--descending"}, false},
		{"ascending and descending", []string{"list", "--ascending", "--descending"}, true},
		{"limit flag", []string{"list", "--limit", "5"}, false},
		// --web is intentionally not tested for the success case: it calls
		// gh browse which opens a browser, making it unsafe to run in CI.
		{"unknown flag", []string{"list", "--unknown"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRootCmd()
			cmd.AddCommand(newListCmd())
			cmd.SetArgs(tc.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			err := cmd.Execute()
			if (err != nil) != tc.wantErr {
				t.Errorf("Execute() err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}
