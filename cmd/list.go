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
	var ascending, descending, all bool
	var limit int
	var web bool
	var tagPrefix string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List semver tags sorted by version (newest first)",
		Long: `List semver tags sorted by version (newest first).

By default, all semver tags are listed regardless of prefix (e.g. v1.2.3,
release-2.0.0, and 1.0.0 all appear together). This differs from other gh tag
commands (tag, view) which filter by the configured prefix so they can operate
on a specific version series.

Use --tag-prefix to restrict output to a single prefix.
Use --all to list every tag on the repo, including non-semver tags.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runListCmd(ascending, limit, web, all, tagPrefix)
		},
	}
	cmd.Flags().BoolVar(&ascending, "ascending", false, "sort oldest first")
	// --descending is the default behavior. The flag exists so users can be
	// explicit in scripts and so Cobra can enforce mutual exclusion with
	// --ascending. It is intentionally not passed to runListCmd.
	cmd.Flags().BoolVar(&descending, "descending", false, "sort newest first (default)")
	cmd.Flags().IntVar(&limit, "limit", 0, "maximum number of tags to show (0 = unlimited)")
	cmd.Flags().BoolVar(&web, "web", false, "open releases page in browser")
	cmd.Flags().BoolVar(&all, "all", false, "list every tag on the repo, including non-semver tags (unsorted)")
	cmd.Flags().StringVar(&tagPrefix, "tag-prefix", "", "filter by tag prefix (default: show all semver tags)")
	cmd.MarkFlagsMutuallyExclusive("ascending", "descending")
	cmd.MarkFlagsMutuallyExclusive("all", "tag-prefix")
	cmd.MarkFlagsMutuallyExclusive("all", "ascending")
	cmd.MarkFlagsMutuallyExclusive("all", "descending")
	return cmd
}

// runListCmd implements the `gh tag list` subcommand. It fetches remote tags
// and prints them one per line, sorted by semantic version (descending by default).
// Without --tag-prefix all semver tags are shown regardless of prefix.
// --all bypasses semver filtering and prints every tag name as-is.
// --limit caps the number of results; --web opens the releases page in the browser.
func runListCmd(ascending bool, limit int, web, all bool, tagPrefix string) error {
	// --web skips tag fetching entirely — no network round trip needed when
	// the goal is just to open the browser.
	if web {
		return openInBrowser("tags")
	}

	if limit < 0 {
		return fmt.Errorf("--limit must be a non-negative integer")
	}

	tags, err := lib.ListRemoteTags()
	if err != nil {
		return err
	}

	if all {
		if len(tags) == 0 {
			fmt.Println("No tags found.")
			return nil
		}
		out := tags
		if limit > 0 && len(out) > limit {
			out = out[:limit]
		}
		for _, t := range out {
			fmt.Println(t)
		}
		return nil
	}

	var sorted []string
	if tagPrefix != "" {
		sorted = lib.SortTags(tags, tagPrefix, ascending)
	} else {
		sorted = lib.SortAllTags(tags, ascending)
	}

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
