package cmd

import (
	"fmt"

	"github.com/karldreher/gh-tag/lib"
	"github.com/spf13/cobra"
)

// viewCmd is the package-level view subcommand, registered with rootCmd via init.
var viewCmd = newViewCmd()

// init registers viewCmd with the root command.
func init() {
	rootCmd.AddCommand(viewCmd)
}

// newViewCmd constructs a fresh view cobra.Command.
func newViewCmd() *cobra.Command {
	var web bool
	cmd := &cobra.Command{
		Use:   "view [tag]",
		Short: "Show a tag and its commit SHA",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runViewCmd(args, web)
		},
	}
	cmd.Flags().BoolVar(&web, "web", false, "open tag in browser")
	return cmd
}

// runViewCmd implements the `gh tag view` subcommand. With no arguments it
// prints the latest semver tag and its commit SHA. With a tag name argument it
// prints that specific tag and its commit SHA. With --web, opens the tag's
// GitHub releases page in the browser instead.
func runViewCmd(args []string, web bool) error {
	prefix, err := lib.EffectivePrefix()
	if err != nil {
		return err
	}

	tags, err := lib.ListRemoteTags()
	if err != nil {
		return err
	}

	var tag string
	if len(args) == 0 {
		major, minor, patch, found := lib.FindLatestTag(tags, prefix)
		if !found {
			fmt.Println("No semver tags found.")
			return nil
		}
		tag = lib.FormatTag(prefix, major, minor, patch)
	} else {
		target := args[0]
		found := false
		for _, t := range tags {
			if t == target {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("tag %q not found on remote", target)
		}
		tag = target
	}

	// --web is checked after tag resolution because we need the tag name to
	// build the URL. ResolveTagRef (a local git call) is intentionally skipped
	// here — the SHA is not needed when opening the browser.
	//
	// The /releases/tag/<name> URL is used rather than /tree/<name> because it
	// surfaces release notes and assets when a GitHub release exists for the tag.
	// When no release exists, GitHub still serves the page and offers a prompt to
	// create one — the URL is valid for any pushed tag regardless of release
	// status. See: https://docs.github.com/en/repositories/releasing-projects-on-github/linking-to-releases
	if web {
		return openInBrowser("releases/tag/" + tag)
	}

	sha, err := lib.ResolveTagRef(tag)
	if err != nil {
		return err
	}

	fmt.Printf("%s  %s\n", tag, sha)
	return nil
}
