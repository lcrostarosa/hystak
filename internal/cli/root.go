package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/hystak/hystak/internal/config"
	"github.com/hystak/hystak/internal/keyconfig"
	"github.com/hystak/hystak/internal/registry"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var configDir string

var rootCmd = &cobra.Command{
	Use:           "hystak",
	Short:         "Manage MCP server configurations for Claude Code",
	Long:          "hystak manages MCP server configurations from a central registry and deploys them to Claude Code project configs.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// S-075: Non-TTY detection — print help when stdout is not a terminal
		if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
			return cmd.Help()
		}
		// TODO: launch TUI when terminal is available
		return cmd.Help()
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if configDir != "" {
			config.OverrideDir(configDir)
		}

		// First-run flow: only for mutating commands (coding-standards rule 6)
		if cmd.Annotations["mutates"] != "true" {
			return nil
		}

		firstRun, err := config.IsFirstRun()
		if err != nil {
			return err
		}
		if !firstRun {
			// Also check if registry is empty (re-run scenario)
			reg, err := registry.LoadDefault()
			if err != nil {
				if !errors.Is(err, fs.ErrNotExist) {
					fmt.Fprintf(os.Stderr, "warning: loading registry: %v\n", err)
				}
				// Continue to first-run flow
			} else {
				_, keysErr := keyconfig.LoadDefault()
				if !reg.IsEmpty() || keysErr == nil {
					return nil
				}
			}
		}

		return runFirstRunFlow(cmd)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configDir, "config-dir", "", "override config directory path")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
