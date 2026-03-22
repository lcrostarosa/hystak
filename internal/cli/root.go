package cli

import (
	"github.com/hystak/hystak/internal/config"
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
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configDir, "config-dir", "", "override config directory path")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
