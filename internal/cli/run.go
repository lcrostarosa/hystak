package cli

import (
	"fmt"
	"os"

	hysterr "github.com/lcrostarosa/hystak/internal/errors"
	"github.com/lcrostarosa/hystak/internal/launch"
	"github.com/spf13/cobra"
)

func (a *cliApp) newRunCmd() *cobra.Command {
	var (
		noSync      bool
		dryRun      bool
		profileName string
	)

	cmd := &cobra.Command{
		Use:   "run <project> [client] [-- extra-args...]",
		Short: "Sync and launch a client in the project directory",
		Long: `Sync a project's MCP configs then launch a client (e.g. claude, opencode, cursor)
in the project directory, replacing the manual "hystak sync && cd && client" workflow.

If no client is specified, the default executable for the project's first client type is used.
Arguments after -- are forwarded to the client process.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Split args at -- separator.
			var extraArgs []string
			if dashAt := cmd.ArgsLenAtDash(); dashAt >= 0 {
				extraArgs = args[dashAt:]
				args = args[:dashAt]
			}

			if len(args) == 0 {
				return fmt.Errorf("project name required")
			}

			projectName := args[0]
			proj, ok := a.svc.GetProject(projectName)
			if !ok {
				return hysterr.ProjectNotFound(projectName)
			}

			// If --profile specified, set it as active before syncing.
			if profileName != "" {
				if err := a.svc.SetActiveProfile(projectName, profileName); err != nil {
					return fmt.Errorf("setting profile: %w", err)
				}
				// Reload project after profile change.
				proj, _ = a.svc.GetProject(projectName)
			}

			// If an explicit client is specified, use it directly instead of syncAndLaunch.
			if len(args) > 1 {
				execName := args[1]

				workDir := proj.Path
				if workDir == "" || workDir == "~" {
					var err error
					workDir, err = os.Getwd()
					if err != nil {
						return fmt.Errorf("getting current directory: %w", err)
					}
				}

				if info, err := os.Stat(workDir); err != nil {
					return fmt.Errorf("project directory %q: %w", workDir, err)
				} else if !info.IsDir() {
					return fmt.Errorf("project path %q is not a directory", workDir)
				}

				if !noSync {
					results, err := a.svc.SyncProject(projectName)
					if err != nil {
						return fmt.Errorf("sync failed: %w", err)
					}
					printSyncResults(cmd, projectName, results)
				}

				if dryRun {
					fmt.Fprintf(cmd.ErrOrStderr(), "Would run: %s %v\n", execName, extraArgs)
					fmt.Fprintf(cmd.ErrOrStderr(), "Directory: %s\n", workDir)
					return nil
				}

				execPath, err := launch.ResolveExecutable(execName)
				if err != nil {
					return err
				}

				return launch.Exec(execPath, extraArgs, workDir)
			}

			// Default client: use shared helper.
			return a.syncAndLaunch(cmd, proj, extraArgs, noSync, dryRun)
		},
	}

	cmd.Flags().BoolVar(&noSync, "no-sync", false, "skip the sync step")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would happen without executing")
	cmd.Flags().StringVar(&profileName, "profile", "", "use a specific profile for this run")

	return cmd
}
