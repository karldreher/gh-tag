package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/karldreher/gh-tag/lib"
	"github.com/spf13/cobra"
)

// effectivePrefix returns the tag prefix from config, defaulting to "v".
func effectivePrefix() (string, error) {
	cfg, err := lib.LoadConfig()
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}
	if cfg.Prefix == "" {
		return "v", nil
	}
	return cfg.Prefix, nil
}

// readBumpType returns "major", "minor", or "patch" — either from flags or
// by prompting the user interactively.
// Key map: M → major, m → minor, p → patch (also full words accepted).
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
	input := strings.TrimSpace(line)

	// Case-sensitive single-char keys: M=major, m=minor, p=patch.
	// Full words accepted case-insensitively as a convenience.
	switch input {
	case "M", "major", "Major":
		return "major", nil
	case "m", "minor", "Minor":
		return "minor", nil
	case "p", "P", "patch", "Patch":
		return "patch", nil
	default:
		return "", fmt.Errorf("invalid bump type: %q (use M=major, m=minor, p=patch)", input)
	}
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

// validateBumpFlags returns an error if more than one of the bump flags is set.
func validateBumpFlags(majorFlag, minorFlag, patchFlag bool) error {
	count := 0
	for _, f := range []bool{majorFlag, minorFlag, patchFlag} {
		if f {
			count++
		}
	}
	if count > 1 {
		return fmt.Errorf("--major, --minor, and --patch are mutually exclusive")
	}
	return nil
}

// runTagCmd is the handler for the root `gh tag` command.
func runTagCmd(cmd *cobra.Command, args []string) error {
	majorFlag, _ := cmd.Flags().GetBool("major")
	minorFlag, _ := cmd.Flags().GetBool("minor")
	patchFlag, _ := cmd.Flags().GetBool("patch")
	skipConfirm, _ := cmd.Flags().GetBool("confirm")

	if err := validateBumpFlags(majorFlag, minorFlag, patchFlag); err != nil {
		return err
	}

	prefix, err := effectivePrefix()
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

	curMajor, curMinor, curPatch, found := lib.FindLatestTag(tags, prefix)

	var newTag string

	if !found {
		if lib.HasTagsWithDifferentPrefix(tags, prefix) {
			// Tags exist on the remote but none match the configured prefix.
			// Warn the user so they don't accidentally create a tag with the
			// wrong prefix on a repo that already has a tagging convention.
			fmt.Printf("⚠️  Found %d tag(s) on remote, but none match prefix %q.\n", len(tags), prefix)
			fmt.Printf("   Run `gh tag prefix --edit` to change the prefix, or proceed to start a new %s series.\n\n", prefix)
		} else {
			fmt.Println("🆕 No existing tags found.")
			fmt.Println()
		}
		bumpType, err := readBumpType(reader, majorFlag, minorFlag, patchFlag)
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

		bumpType, err := readBumpType(reader, majorFlag, minorFlag, patchFlag)
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

// runPrefixCmd is the handler for `gh tag prefix`.
func runPrefixCmd(cmd *cobra.Command, args []string) error {
	editFlag, _ := cmd.Flags().GetBool("edit")

	current, err := effectivePrefix()
	if err != nil {
		return err
	}

	if !editFlag {
		fmt.Printf("Tag prefix: %s\n", current)
		return nil
	}

	fmt.Printf("Current prefix: %s\n", current)
	fmt.Printf("Enter new tag prefix [%s]: ", current)

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	input := strings.TrimSpace(line)
	if input == "" {
		input = current
	}

	cfg, err := lib.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	cfg.Prefix = input
	if err := lib.SaveConfig(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("✅ Tag prefix set to: %s\n", cfg.Prefix)
	return nil
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "gh-tag",
		Short: "🏷️  The missing tag command.",
		RunE:  runTagCmd,
	}
	rootCmd.Flags().Bool("major", false, "bump major version")
	rootCmd.Flags().Bool("minor", false, "bump minor version")
	rootCmd.Flags().Bool("patch", false, "bump patch version")
	rootCmd.Flags().Bool("confirm", false, "skip confirmation prompt")

	prefixCmd := &cobra.Command{
		Use:   "prefix",
		Short: "View or set the tag prefix (default: v)",
		RunE:  runPrefixCmd,
	}
	prefixCmd.Flags().Bool("edit", false, "interactively set the tag prefix")

	rootCmd.AddCommand(prefixCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
