package cmd

import (
	"errors"
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

// singleTag is a semver tag name that has been verified to be the only
// matching tag in the repository. --web with no explicit argument requires
// this guarantee: opening a releases page should be an unambiguous act —
// the user must not be silently taken to a tag they did not intend.
// Named arguments bypass this type entirely because the user already chose.
type singleTag string

// errNoSemverTags is returned by requireSingleTag when no semver tags exist.
// It is a sentinel distinct from the >1-tag error so callers can exit 0
// with an informational message rather than treating it as a failure.
var errNoSemverTags = errors.New("no semver tags found")

// requireSingleTag returns the one semver tag matching prefix in tags.
// It errors if zero tags exist (errNoSemverTags) or if more than one exists,
// enforcing that --web with no argument is always unambiguous.
func requireSingleTag(tags []string, prefix string) (singleTag, error) {
	matching := lib.SortTags(tags, prefix, false)
	switch len(matching) {
	case 0:
		return "", errNoSemverTags
	case 1:
		return singleTag(matching[0]), nil
	default:
		return "", fmt.Errorf(
			"%d semver tags exist; specify a tag name to avoid ambiguity (e.g. gh tag view %s --web)",
			len(matching), matching[0])
	}
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

	if web && len(args) == 0 {
		// --web with no argument must be unambiguous: if multiple tags exist the
		// user must name one. requireSingleTag enforces this via the singleTag
		// type, which is only constructible when exactly one tag is present.
		tag, err := requireSingleTag(tags, prefix)
		if errors.Is(err, errNoSemverTags) {
			fmt.Println("No semver tags found.")
			return nil
		}
		if err != nil {
			return err
		}
		// The /releases/tag/<name> URL is used rather than /tree/<name> because it
		// surfaces release notes and assets when a GitHub release exists for the tag.
		// When no release exists, GitHub still serves the page and offers a prompt to
		// create one — the URL is valid for any pushed tag regardless of release
		// status. See: https://docs.github.com/en/repositories/releasing-projects-on-github/linking-to-releases
		return openInBrowser("releases/tag/" + string(tag))
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

	if web {
		// Named argument: the user was explicit, no ambiguity constraint applies.
		// ResolveTagRef is skipped — the SHA is not needed when opening the browser.
		return openInBrowser("releases/tag/" + tag)
	}

	sha, err := lib.ResolveTagRef(tag)
	if err != nil {
		return err
	}

	fmt.Printf("%s  %s\n", tag, sha)
	return nil
}
