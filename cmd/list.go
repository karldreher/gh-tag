package cmd

import (
	"fmt"

	"github.com/karldreher/gh-tag/lib"
	"github.com/spf13/cobra"
)

// listCmd is the package-level list subcommand, registered with rootCmd via init.
var listCmd = newListCmd()

// init registers listCmd with the root command.
func init() {
	rootCmd.AddCommand(listCmd)
}

// newListCmd constructs a fresh list cobra.Command with its --ascending flag.
func newListCmd() *cobra.Command {
	var ascending bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List semver tags sorted by version (newest first)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runListCmd(ascending)
		},
	}
	cmd.Flags().BoolVar(&ascending, "ascending", false, "sort oldest first")
	return cmd
}

// runListCmd implements the `gh tag list` subcommand. It fetches remote tags,
// filters to valid semver entries matching the configured prefix, and prints
// them one per line, sorted by semantic version (descending by default).
func runListCmd(ascending bool) error {
	prefix, err := lib.EffectivePrefix()
	if err != nil {
		return err
	}

	tags, err := lib.ListRemoteTags()
	if err != nil {
		return err
	}

	sorted := lib.SortTags(tags, prefix, ascending)
	if len(sorted) == 0 {
		fmt.Println("No tags found.")
		return nil
	}

	for _, t := range sorted {
		fmt.Println(t)
	}
	return nil
}
