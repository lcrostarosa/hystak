package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/hystak/hystak/internal/config"
	"github.com/hystak/hystak/internal/discovery"
	"github.com/hystak/hystak/internal/keyconfig"
	"github.com/hystak/hystak/internal/registry"
	"github.com/spf13/cobra"
)

// runFirstRunFlow executes the first-run setup: keybinding prompt (S-002)
// and existing config scanning (S-003).
func runFirstRunFlow(cmd *cobra.Command) error {
	// S-001: Ensure config directory exists
	if _, err := config.EnsureConfigDir(); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// S-002: Keybinding prompt
	profile, err := promptKeybindingProfile(cmd)
	if err != nil {
		return err
	}

	cfg := keyconfig.Config{Profile: profile}
	if err := keyconfig.SaveDefault(cfg); err != nil {
		return fmt.Errorf("saving keybinding config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Saved keybinding profile: %s\n\n", profile)

	// S-003: Scan existing configs
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	candidates, err := discovery.ScanAll(cwd)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: scanning configs: %v\n", err)
		return nil
	}

	if len(candidates) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No existing MCP servers found.")
		return nil
	}

	// Show discovered servers
	fmt.Fprintf(cmd.OutOrStdout(), "Found %d MCP server(s):\n", len(candidates))
	for _, c := range candidates {
		fmt.Fprintf(cmd.OutOrStdout(), "  [*] %s (%s) from %s\n", c.Name, c.Server.Transport, c.Source)
	}
	fmt.Fprintln(cmd.OutOrStdout())

	// Confirm import
	fmt.Fprint(cmd.OutOrStdout(), "Import these servers into registry? [Y]es / [N]o: ")
	reader := bufio.NewReader(cmd.InOrStdin())
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil // EOF = skip
	}

	choice := strings.TrimSpace(strings.ToLower(input))
	if choice != "y" && choice != "yes" && choice != "" {
		fmt.Fprintln(cmd.OutOrStdout(), "Skipped import.")
		return nil
	}

	// Import into registry
	reg, err := registry.LoadDefault()
	if err != nil {
		return fmt.Errorf("loading registry: %w", err)
	}

	imported := 0
	for _, c := range candidates {
		if _, exists := reg.Servers.Get(c.Name); exists {
			fmt.Fprintf(cmd.OutOrStdout(), "  Skipped %q (already in registry)\n", c.Name)
			continue
		}
		if err := reg.Servers.Add(c.Server); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "  Warning: adding %q: %v\n", c.Name, err)
			continue
		}
		imported++
	}

	if err := reg.SaveDefault(); err != nil {
		return fmt.Errorf("saving registry: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Imported %d server(s) into registry.\n", imported)
	return nil
}

// promptKeybindingProfile asks the user to choose a keybinding profile (S-002).
func promptKeybindingProfile(cmd *cobra.Command) (keyconfig.Profile, error) {
	fmt.Fprint(cmd.OutOrStdout(), "Navigation style? [A]rrows (recommended) / [V]im / [C]lassic: ")

	reader := bufio.NewReader(cmd.InOrStdin())
	input, err := reader.ReadString('\n')
	if err != nil {
		return keyconfig.ProfileArrows, nil // EOF = default
	}

	switch strings.TrimSpace(strings.ToLower(input)) {
	case "v", "vim":
		return keyconfig.ProfileVim, nil
	case "c", "classic":
		return keyconfig.ProfileClassic, nil
	default:
		return keyconfig.ProfileArrows, nil
	}
}
