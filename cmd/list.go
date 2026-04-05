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

// newListCmd constructs a fresh list cobra.Command with its flags.
func newListCmd() *cobra.Command {
	var ascending, descending bool
	var limit int
	var web bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List semver tags sorted by version (newest first)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runListCmd(ascending, limit, web)
		},
	}
	cmd.Flags().BoolVar(&ascending, "ascending", false, "sort oldest first")
	// --descending is the default behavior. The flag exists so users can be
	// explicit in scripts and so Cobra can enforce mutual exclusion with
	// --ascending. It is intentionally not passed to runListCmd.
	cmd.Flags().BoolVar(&descending, "descending", false, "sort newest first (default)")
	cmd.Flags().IntVar(&limit, "limit", 0, "maximum number of tags to show (0 = unlimited)")
	cmd.Flags().BoolVar(&web, "web", false, "open releases page in browser")
	cmd.MarkFlagsMutuallyExclusive("ascending", "descending")
	return cmd
}

// runListCmd implements the `gh tag list` subcommand. It fetches remote tags,
// filters to valid semver entries matching the configured prefix, and prints
// them one per line, sorted by semantic version (descending by default).
// --limit caps the number of results; --web opens the releases page in the browser.
func runListCmd(ascending bool, limit int, web bool) error {
	// --web skips tag fetching entirely — no network round trip needed when
	// the goal is just to open the browser.
	if web {
		return openInBrowser("tags")
	}

	if limit < 0 {
		return fmt.Errorf("--limit must be a non-negative integer")
	}

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

	if limit > 0 && len(sorted) > limit {
		sorted = sorted[:limit]
	}

	for _, t := range sorted {
		fmt.Println(t)
	}
	return nil
}
