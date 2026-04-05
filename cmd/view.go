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
	return &cobra.Command{
		Use:   "view [tag]",
		Short: "Show a tag and its commit SHA",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runViewCmd(args)
		},
	}
}

// runViewCmd implements the `gh tag view` subcommand. With no arguments it
// prints the latest semver tag and its commit SHA. With a tag name argument it
// prints that specific tag and its commit SHA.
func runViewCmd(args []string) error {
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

	sha, err := lib.ResolveTagRef(tag)
	if err != nil {
		return err
	}

	fmt.Printf("%s  %s\n", tag, sha)
	return nil
}
