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

	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Saved keybinding profile: %s\n\n", profile); err != nil {
		return err
	}

	// S-003: Scan existing configs
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	candidates, err := discovery.ScanAll(cwd)
	if err != nil {
		if _, wErr := fmt.Fprintf(cmd.ErrOrStderr(), "Warning: scanning configs: %v\n", err); wErr != nil {
			return wErr
		}
		return nil
	}

	if len(candidates) == 0 {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "No existing MCP servers found."); err != nil {
			return err
		}
		return nil
	}

	// Show discovered servers
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Found %d MCP server(s):\n", len(candidates)); err != nil {
		return err
	}
	for _, c := range candidates {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  [*] %s (%s) from %s\n", c.Name, c.Server.Transport, c.Source); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout()); err != nil {
		return err
	}

	// Confirm import
	if _, err := fmt.Fprint(cmd.OutOrStdout(), "Import these servers into registry? [Y]es / [N]o: "); err != nil {
		return err
	}
	reader := bufio.NewReader(cmd.InOrStdin())
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil // EOF = skip
	}

	choice := strings.TrimSpace(strings.ToLower(input))
	if choice != "y" && choice != "yes" && choice != "" {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Skipped import."); err != nil {
			return err
		}
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
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  Skipped %q (already in registry)\n", c.Name); err != nil {
				return err
			}
			continue
		}
		if err := reg.Servers.Add(c.Server); err != nil {
			if _, wErr := fmt.Fprintf(cmd.ErrOrStderr(), "  Warning: adding %q: %v\n", c.Name, err); wErr != nil {
				return wErr
			}
			continue
		}
		imported++
	}

	if err := reg.SaveDefault(); err != nil {
		return fmt.Errorf("saving registry: %w", err)
	}

	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Imported %d server(s) into registry.\n", imported); err != nil {
		return err
	}
	return nil
}

// promptKeybindingProfile asks the user to choose a keybinding profile (S-002).
func promptKeybindingProfile(cmd *cobra.Command) (keyconfig.Profile, error) {
	if _, err := fmt.Fprint(cmd.OutOrStdout(), "Navigation style? [A]rrows (recommended) / [V]im / [C]lassic: "); err != nil {
		return keyconfig.ProfileArrows, err
	}

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
