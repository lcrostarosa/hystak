package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "hystak",
	Short:         "Manage MCP server configurations for Claude Code",
	Long:          "hystak manages MCP server configurations from a central registry and deploys them to Claude Code project configs.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
