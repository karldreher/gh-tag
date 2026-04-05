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
