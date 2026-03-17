package cli

import (
	"fmt"
	"strings"

	"github.com/rbbydotdev/hystak/internal/model"
	"github.com/spf13/cobra"
)

func newOverrideCmd() *cobra.Command {
	var (
		envFlags  []string
		argsFlag  []string
	)

	cmd := &cobra.Command{
		Use:   "override <project> <server>",
		Short: "Set per-project overrides for a server",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]
			serverName := args[1]

			override := model.ServerOverride{}
			hasOverride := false

			if len(envFlags) > 0 {
				override.Env = make(map[string]string, len(envFlags))
				for _, e := range envFlags {
					parts := strings.SplitN(e, "=", 2)
					if len(parts) != 2 {
						return fmt.Errorf("invalid env format %q: expected KEY=VAL", e)
					}
					override.Env[parts[0]] = parts[1]
				}
				hasOverride = true
			}

			if len(argsFlag) > 0 {
				override.Args = argsFlag
				hasOverride = true
			}

			if !hasOverride {
				return fmt.Errorf("at least one override flag required (--env or --args)")
			}

			if err := svc.Projects.SetOverride(projectName, serverName, override); err != nil {
				return err
			}

			if err := svc.SaveProjects(); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Override set for %s in project %s.\n", serverName, projectName)
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&envFlags, "env", nil, "environment variable override (KEY=VAL, repeatable)")
	cmd.Flags().StringSliceVar(&argsFlag, "args", nil, "argument override (comma-separated)")

	return cmd
}
