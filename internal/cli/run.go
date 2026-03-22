package cli

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/hystak/hystak/internal/launch"
	"github.com/hystak/hystak/internal/service"
	"github.com/spf13/cobra"
)

var (
	runProfile string
	runNoSync  bool
	runDryRun  bool
)

var runCmd = &cobra.Command{
	Use:         "run <project> [-- extra-args...]",
	Short:       "Sync and launch Claude Code",
	Long:        "Resolve the active profile, deploy configs, and launch the client. Post-exit loop offers relaunch/configure/quit.",
	Args:        cobra.ArbitraryArgs,
	Annotations: map[string]string{"mutates": "true"},
	RunE:        runRun,
}

func init() {
	runCmd.Flags().StringVar(&runProfile, "profile", "", "use a specific profile instead of active")
	runCmd.Flags().BoolVar(&runNoSync, "no-sync", false, "launch without syncing")
	runCmd.Flags().BoolVar(&runDryRun, "dry-run", false, "show sync plan and launch command without executing")
	rootCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("project name is required")
	}

	projectName := args[0]
	var extraArgs []string
	dashIdx := cmd.ArgsLenAtDash()
	if dashIdx >= 0 {
		extraArgs = args[dashIdx:]
	}

	svc, err := buildService()
	if err != nil {
		return err
	}

	// Fail fast if claude not in PATH
	clientCmd := launch.DefaultClientCommand()
	if !runDryRun {
		if _, err := launch.FindClient(clientCmd); err != nil {
			return err
		}
	}

	// Override active profile if --profile flag is set
	if runProfile != "" {
		if err := svc.SetActiveProfile(projectName, runProfile); err != nil {
			return fmt.Errorf("setting profile %q: %w", runProfile, err)
		}
	}

	proj, ok := svc.GetProject(projectName)
	if !ok {
		return fmt.Errorf("project %q not found", projectName)
	}

	if runDryRun {
		return dryRun(cmd, svc, projectName, clientCmd, proj.Path, extraArgs)
	}

	// Post-exit loop (S-054)
	for {
		if !runNoSync {
			results, err := svc.SyncProject(projectName)
			if err != nil {
				return fmt.Errorf("sync failed: %w", err)
			}
			if err := printSyncResults(cmd, results); err != nil {
				return err
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Launching %s in %s...\n", clientCmd, proj.Path)

		launchArgs := extraArgs
		err := launch.RunCommand(clientCmd, launchArgs, proj.Path)

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				return fmt.Errorf("launch failed: %w", err)
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\nClaude exited (code %d). What next?\n", exitCode)
		fmt.Fprintln(cmd.OutOrStdout(), "  [R]elaunch  [C]onfigure  [Q]uit")

		action := readPostExitChoice(cmd)

		switch action {
		case "r":
			runNoSync = false // re-sync on relaunch
			continue
		case "c":
			fmt.Fprintln(cmd.OutOrStdout(), "  Configure mode not yet available (requires TUI).")
			continue
		default:
			return nil
		}
	}
}

func dryRun(cmd *cobra.Command, svc *service.Service, projectName, clientCmd, projectPath string, extraArgs []string) error {
	results, err := svc.DryRunSync(projectName)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Sync plan:")
	if err := printSyncResults(cmd, results); err != nil {
		return err
	}

	launchParts := []string{clientCmd}
	launchParts = append(launchParts, extraArgs...)
	fmt.Fprintf(cmd.OutOrStdout(), "\nWould launch: %s\n", strings.Join(launchParts, " "))
	fmt.Fprintf(cmd.OutOrStdout(), "Working dir:  %s\n", projectPath)
	return nil
}

func printSyncResults(cmd *cobra.Command, results []service.SyncResult) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	for _, r := range results {
		if _, err := fmt.Fprintf(w, "  %s\t%s\n", r.Name, r.Action); err != nil {
			return err
		}
	}
	return w.Flush()
}

// readPostExitChoice reads from cmd.InOrStdin() for testability.
func readPostExitChoice(cmd *cobra.Command) string {
	fmt.Fprint(cmd.OutOrStdout(), "  Choice: ")

	reader := bufio.NewReader(cmd.InOrStdin())
	input, err := reader.ReadString('\n')
	if err != nil {
		return "q" // EOF or error = quit
	}
	return strings.TrimSpace(strings.ToLower(input))
}
