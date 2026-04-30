package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/karldreher/gh-tag/lib"
	"github.com/spf13/cobra"
)

// rootCmd is the package-level root command, built once at package initialisation.
// Subcommands register themselves against it via init() in their own files.
var rootCmd = newRootCmd()

// Execute runs the root command and returns any error for the caller to handle.
func Execute() error {
	return rootCmd.Execute()
}

// newRootCmd constructs a fresh root cobra.Command with all flags bound to
// local variables. Returning a new instance each time makes the function safe
// to call from tests without shared flag state.
func newRootCmd() *cobra.Command {
	var major, minor, patch, confirm, overwrite, auto bool
	var tagPrefix string
	cmd := &cobra.Command{
		Use:           "gh tag",
		Short:         "🏷️  The missing tag command.",
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTagCmd(major, minor, patch, confirm, overwrite, auto, tagPrefix)
		},
	}
	cmd.Flags().BoolVar(&major, "major", false, "bump major version")
	cmd.Flags().BoolVar(&minor, "minor", false, "bump minor version")
	cmd.Flags().BoolVar(&patch, "patch", false, "bump patch version")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "skip confirmation prompt")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "overwrite the latest tag at HEAD")
	cmd.Flags().BoolVar(&auto, "auto", false, "infer bump type from HEAD commit (conventional commits)")
	cmd.Flags().StringVar(&tagPrefix, "tag-prefix", "", "override tag prefix for this operation (ephemeral, not saved to config)")
	cmd.MarkFlagsMutuallyExclusive("overwrite", "major", "minor", "patch", "auto")
	return cmd
}

// resolveBumpType returns "major", "minor", or "patch". When autoFlag is set it
// reads the HEAD commit title and infers the bump type from conventional commits.
// Otherwise it delegates to readBumpType (flags or interactive prompt).
func resolveBumpType(reader *bufio.Reader, majorFlag, minorFlag, patchFlag, autoFlag bool) (string, error) {
	if !autoFlag {
		return readBumpType(reader, majorFlag, minorFlag, patchFlag)
	}
	title, err := lib.HeadCommitTitle()
	if err != nil {
		return "", err
	}
	bumpType, err := lib.ParseConventionalCommitBumpType(title)
	if err != nil {
		return "", err
	}
	fmt.Printf("🤖 Auto: inferred %q from commit: %s\n", bumpType, title)
	return bumpType, nil
}

// readBumpType returns "major", "minor", or "patch" — either from flags or
// by prompting the user interactively.
func readBumpType(reader *bufio.Reader, majorFlag, minorFlag, patchFlag bool) (string, error) {
	if majorFlag {
		return "major", nil
	}
	if minorFlag {
		return "minor", nil
	}
	if patchFlag {
		return "patch", nil
	}

	fmt.Print("📦 Bump type? [M]ajor / [m]inor / [p]atch: ")
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}
	return lib.ParseBumpType(strings.TrimSpace(line))
}

// confirmAction prints prompt and reads a y/yes response unless skipConfirm is true.
func confirmAction(reader *bufio.Reader, skipConfirm bool, prompt string) (bool, error) {
	if skipConfirm {
		return true, nil
	}
	fmt.Print(prompt)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("reading input: %w", err)
	}
	input := strings.ToLower(strings.TrimSpace(line))
	return input == "y" || input == "yes", nil
}

// resolveOperatingPrefix determines which tag prefix to use for the current
// operation. Decision tree:
//
//  1. If tagPrefixFlag is non-empty, return it immediately (user override).
//  2. If configuredPrefix already has matching tags, return it (normal path).
//  3. Otherwise collect alternative semver prefixes from remote tags:
//     - 0 alternatives → return configuredPrefix (new series will be created).
//     - 1 alternative  → prompt user to use it or fall back to configuredPrefix.
//     - 2+ alternatives → present numbered list; user must select or type "n".
//
// The returned prefix is ephemeral and is never written to config.
func resolveOperatingPrefix(reader *bufio.Reader, tags []string, configuredPrefix, tagPrefixFlag string) (string, error) {
	if tagPrefixFlag != "" {
		return tagPrefixFlag, nil
	}

	_, _, _, found := lib.FindLatestTag(tags, configuredPrefix)
	if found {
		return configuredPrefix, nil
	}

	allPrefixes := lib.UniqueSemverPrefixes(tags)
	var alternatives []string
	for _, p := range allPrefixes {
		if p != configuredPrefix {
			alternatives = append(alternatives, p)
		}
	}

	switch len(alternatives) {
	case 0:
		return configuredPrefix, nil

	case 1:
		fmt.Printf("⚠️  Found tags with prefix %q. Use this prefix? [y/N]: ", alternatives[0])
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("reading input: %w", err)
		}
		input := strings.ToLower(strings.TrimSpace(line))
		if input == "y" || input == "yes" {
			return alternatives[0], nil
		}
		return configuredPrefix, nil

	default:
		fmt.Println("⚠️  Multiple tag prefix groups found on remote:")
		for i, p := range alternatives {
			fmt.Printf("   %d) %s\n", i+1, p)
		}
		fmt.Printf("   n) Start new %q series\n", configuredPrefix)
		fmt.Print("Select prefix: ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("reading input: %w", err)
		}
		input := strings.ToLower(strings.TrimSpace(line))
		if input == "n" {
			return configuredPrefix, nil
		}
		n, err := strconv.Atoi(input)
		if err != nil || n < 1 || n > len(alternatives) {
			return "", fmt.Errorf("invalid selection %q: enter a number between 1 and %d, or n", input, len(alternatives))
		}
		return alternatives[n-1], nil
	}
}

// runTagCmd implements the root `gh tag` command. It fetches remote tags,
// determines the next version (or re-points the latest tag when overwriteFlag
// is set), confirms with the user, then creates and pushes the tag.
func runTagCmd(majorFlag, minorFlag, patchFlag, skipConfirm, overwriteFlag, autoFlag bool, tagPrefixFlag string) error {
	prefix, err := lib.EffectivePrefix()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("🏷️  gh tag — The missing command.")
	fmt.Println()
	fmt.Print("📡 Fetching remote tags...")

	tags, err := lib.ListRemoteTags()
	if err != nil {
		fmt.Println()
		return fmt.Errorf("fetching remote tags: %w", err)
	}
	fmt.Println()

	prefix, err = resolveOperatingPrefix(reader, tags, prefix, tagPrefixFlag)
	if err != nil {
		return err
	}

	curMajor, curMinor, curPatch, found := lib.FindLatestTag(tags, prefix)

	if overwriteFlag {
		if !found {
			return fmt.Errorf("no existing tag found to overwrite")
		}
		currentTag := lib.FormatTag(prefix, curMajor, curMinor, curPatch)

		currentRef, err := lib.ResolveTagRef(currentTag)
		if err != nil {
			return fmt.Errorf("resolving tag ref: %w", err)
		}
		headRef, err := lib.ResolveHead()
		if err != nil {
			return fmt.Errorf("resolving HEAD: %w", err)
		}

		fmt.Printf("⚠️  Overwrite mode.\n")
		fmt.Printf("   Tag:      %s\n", currentTag)
		fmt.Printf("   Current:  %s (remote)\n", currentRef)
		fmt.Printf("   New ref:  %s (HEAD)\n\n", headRef)

		cfg, err := lib.LoadConfig()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		alreadyConfirmed := cfg.OverwriteConfirmed

		if !skipConfirm || !alreadyConfirmed {
			fmt.Println("This is a destructive operation. The remote tag will be force-pushed.")
			fmt.Println()
		}
		if !alreadyConfirmed {
			cfg.OverwriteConfirmed = true
			if err := lib.SaveConfig(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}
		}

		if !skipConfirm {
			fmt.Printf("Type the tag name to confirm: ")
			line, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading input: %w", err)
			}
			if strings.TrimSpace(line) != currentTag {
				fmt.Println("Aborted: tag name did not match.")
				return nil
			}
		}

		if err := lib.OverwriteTag(currentTag); err != nil {
			return err
		}
		if err := lib.ForcePushTag(currentTag); err != nil {
			if errors.Is(err, lib.ErrPushImmutable) {
				return fmt.Errorf("push rejected: the remote may have release immutability enabled\n  The local tag has been updated but the remote was not changed\n  To restore the original local tag: git tag -f %s %s", currentTag, currentRef)
			}
			return err
		}

		fmt.Printf("\n✅ Done! Tag %s overwritten and pushed.\n", currentTag)
		return nil
	}

	var newTag string

	if !found {
		if tagPrefixFlag != "" {
			fmt.Printf("🆕 No existing tags found for prefix %q. Starting a new series.\n\n", prefix)
		} else {
			fmt.Println("🆕 No existing tags found.")
			fmt.Println()
		}
		bumpType, err := resolveBumpType(reader, majorFlag, minorFlag, patchFlag, autoFlag)
		if err != nil {
			return err
		}
		switch bumpType {
		case "major":
			newTag = lib.FormatTag(prefix, 1, 0, 0)
		case "minor":
			newTag = lib.FormatTag(prefix, 0, 1, 0)
		default: // patch
			newTag = lib.FormatTag(prefix, 0, 0, 1)
		}
	} else {
		current := lib.FormatTag(prefix, curMajor, curMinor, curPatch)
		fmt.Printf("✅ Latest tag: %s\n", current)
		fmt.Println()

		bumpType, err := resolveBumpType(reader, majorFlag, minorFlag, patchFlag, autoFlag)
		if err != nil {
			return err
		}
		newMajor, newMinor, newPatch := lib.BumpVersion(curMajor, curMinor, curPatch, bumpType)
		newTag = lib.FormatTag(prefix, newMajor, newMinor, newPatch)
	}

	fmt.Printf("\n✨ New tag: %s\n", newTag)
	fmt.Println()
	fmt.Printf("⚡ Ready to tag and push:\n   git tag %s\n   git push origin %s\n", newTag, newTag)
	fmt.Println()

	confirmed, err := confirmAction(reader, skipConfirm, "🚀 Proceed? [y/N]: ")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Aborted.")
		return nil
	}

	if err := lib.CreateLocalTag(newTag); err != nil {
		return err
	}
	if err := lib.PushTag(newTag); err != nil {
		return err
	}

	fmt.Printf("\n✅ Done! Tag %s created and pushed.\n", newTag)
	return nil
}
