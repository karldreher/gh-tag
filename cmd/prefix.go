package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/karldreher/gh-tag/lib"
	"github.com/spf13/cobra"
)

// prefixCmd is the package-level prefix subcommand, built once at package
// initialisation and registered with rootCmd via init.
var prefixCmd = newPrefixCmd()

// init registers prefixCmd with the root command.
func init() {
	rootCmd.AddCommand(prefixCmd)
}

// newPrefixCmd constructs a fresh prefix cobra.Command with its --edit flag
// bound to a local variable.
func newPrefixCmd() *cobra.Command {
	var edit bool
	cmd := &cobra.Command{
		Use:   "prefix",
		Short: "View or set the tag prefix (default: v)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runPrefixCmd(edit)
		},
	}
	cmd.Flags().BoolVar(&edit, "edit", false, "interactively set the tag prefix")
	return cmd
}

// runPrefixCmd implements the `gh tag prefix` subcommand. Without editFlag it
// prints the current prefix; with editFlag it prompts the user for a new value
// and persists it to ~/.gh-tag/config.json.
func runPrefixCmd(editFlag bool) error {
	current, err := lib.EffectivePrefix()
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
