package cli

import (
	"fmt"
	"os"

	"github.com/lcrostarosa/hystak/internal/launch"
	"github.com/spf13/cobra"
)

func (a *cliApp) newRunCmd() *cobra.Command {
	var (
		noSync bool
		dryRun bool
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
			proj, ok := a.svc.Projects.Get(projectName)
			if !ok {
				return fmt.Errorf("project %q not found", projectName)
			}

			// Determine client executable.
			var execName string
			if len(args) > 1 {
				execName = args[1]
			} else {
				if len(proj.Clients) == 0 {
					return fmt.Errorf("project %q has no clients configured and no client was specified", projectName)
				}
				var err error
				execName, err = launch.DefaultExecutable(proj.Clients[0])
				if err != nil {
					return fmt.Errorf("cannot determine default client for project %q: %w", projectName, err)
				}
			}

			// Fail fast: resolve executable before syncing.
			execPath, err := launch.ResolveExecutable(execName)
			if err != nil {
				return err
			}

			// Determine working directory.
			workDir := proj.Path
			if workDir == "" || workDir == "~" {
				workDir, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("getting current directory: %w", err)
				}
			}

			// Validate working directory exists.
			if info, err := os.Stat(workDir); err != nil {
				return fmt.Errorf("project directory %q: %w", workDir, err)
			} else if !info.IsDir() {
				return fmt.Errorf("project path %q is not a directory", workDir)
			}

			// Sync unless --no-sync.
			if !noSync {
				results, err := a.svc.SyncProject(projectName)
				if err != nil {
					return fmt.Errorf("sync failed: %w", err)
				}
				printSyncResults(cmd, projectName, results)
			}

			if dryRun {
				fmt.Fprintf(cmd.ErrOrStderr(), "Would run: %s %v\n", execPath, extraArgs)
				fmt.Fprintf(cmd.ErrOrStderr(), "Directory: %s\n", workDir)
				return nil
			}

			return launch.Exec(execPath, extraArgs, workDir)
		},
	}

	cmd.Flags().BoolVar(&noSync, "no-sync", false, "skip the sync step")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would happen without executing")

	return cmd
}
