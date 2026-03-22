package cli

import (
	"github.com/hystak/hystak/internal/config"
	"github.com/hystak/hystak/internal/keyconfig"
	"github.com/hystak/hystak/internal/registry"
	"github.com/spf13/cobra"
)

var configDir string

var rootCmd = &cobra.Command{
	Use:           "hystak",
	Short:         "Manage MCP server configurations for Claude Code",
	Long:          "hystak manages MCP server configurations from a central registry and deploys them to Claude Code project configs.",
	SilenceUsage:  true,
	SilenceErrors: true,
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
				return nil // non-blocking
			}
			_, keysErr := keyconfig.LoadDefault()
			if !reg.IsEmpty() || keysErr == nil {
				return nil
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
