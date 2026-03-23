package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/hystak/hystak/internal/backup"
	"github.com/hystak/hystak/internal/deploy"
	"github.com/hystak/hystak/internal/model"
	"github.com/hystak/hystak/internal/profile"
	"github.com/hystak/hystak/internal/project"
	"github.com/hystak/hystak/internal/registry"
	"github.com/hystak/hystak/internal/service"
	"github.com/spf13/cobra"
)

var (
	syncAll     bool
	syncProfile string
	syncDryRun  bool
	syncForce   bool
)

var syncCmd = &cobra.Command{
	Use:         "sync [project]",
	Short:       "Deploy project configs",
	Long:        "Resolve the active profile and deploy MCP servers to client config files.",
	Args:        cobra.MaximumNArgs(1),
	Annotations: map[string]string{"mutates": "true"},
	RunE:        runSync,
}

func init() {
	syncCmd.Flags().BoolVar(&syncAll, "all", false, "sync all projects (S-034)")
	syncCmd.Flags().StringVar(&syncProfile, "profile", "", "use a specific profile (S-035)")
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "show sync plan without writing (S-036)")
	syncCmd.Flags().BoolVar(&syncForce, "force", false, "skip preflight conflict checks")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	if !syncAll && len(args) == 0 {
		return fmt.Errorf("project name is required (or use --all)")
	}

	svc, err := buildService()
	if err != nil {
		return err
	}

	if syncAll {
		return runSyncAll(cmd, svc)
	}

	projectName := args[0]

	// S-035: Override active profile if --profile flag is set
	if syncProfile != "" {
		if err := svc.SetActiveProfile(projectName, syncProfile); err != nil {
			return fmt.Errorf("setting profile %q: %w", syncProfile, err)
		}
	}

	// S-046: Preflight conflict check (unless --force)
	if !syncForce && !syncDryRun {
		conflicts, prefErr := svc.PreflightCheck(projectName)
		if prefErr != nil {
			return fmt.Errorf("preflight check: %w", prefErr)
		}
		if len(conflicts) > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "Preflight conflicts (%d):\n", len(conflicts))
			for _, c := range conflicts {
				fmt.Fprintf(cmd.ErrOrStderr(), "  %s: %s\n", c.Path, c.Message)
			}
			return fmt.Errorf("resolve conflicts before syncing (or use --force to skip)")
		}
	}

	var results []service.SyncResult
	if syncDryRun {
		results, err = svc.DryRunSync(projectName)
	} else {
		results, err = svc.SyncProject(projectName)
	}
	if err != nil {
		return err
	}

	if syncDryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "Dry run (no changes written):")
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	for _, r := range results {
		if _, err := fmt.Fprintf(w, "%s\t%s\n", r.Name, r.Action); err != nil {
			return err
		}
	}
	return w.Flush()
}

// runSyncAll syncs every registered project (S-034).
func runSyncAll(cmd *cobra.Command, svc *service.Service) error {
	projects := svc.ListProjects()
	if len(projects) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No projects registered.")
		return nil
	}

	for _, p := range projects {
		fmt.Fprintf(cmd.OutOrStdout(), "--- %s ---\n", p.Name)

		var results []service.SyncResult
		var err error
		if syncDryRun {
			results, err = svc.DryRunSync(p.Name)
		} else {
			results, err = svc.SyncProject(p.Name)
		}
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "  error: %v\n", err)
			continue
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		for _, r := range results {
			if _, err := fmt.Fprintf(w, "  %s\t%s\n", r.Name, r.Action); err != nil {
				return err
			}
		}
		if err := w.Flush(); err != nil {
			return err
		}
	}
	return nil
}

// buildService creates a fully wired Service from default config paths.
func buildService() (*service.Service, error) {
	reg, err := registry.LoadDefault()
	if err != nil {
		return nil, fmt.Errorf("loading registry: %w", err)
	}

	projStore, err := project.LoadDefault()
	if err != nil {
		return nil, fmt.Errorf("loading projects: %w", err)
	}

	profMgr := profile.NewDefaultManager()

	dep, ok := deploy.NewDeployer(model.ClientClaudeCode)
	if !ok {
		return nil, fmt.Errorf("no deployer for %s", model.ClientClaudeCode)
	}

	svc := service.New(reg, projStore, profMgr, dep)
	svc.WithBackup(backup.NewDefaultManager())
	svc.WithResourceDeployers(
		&deploy.SkillsDeployer{},
		&deploy.SettingsDeployer{},
		&deploy.ClaudeMDDeployer{},
	)
	return svc, nil
}

// buildServiceReadOnly creates a Service for read-only operations.
// Errors to stderr if non-critical components fail.
func buildServiceReadOnly() (*service.Service, error) {
	reg, err := registry.LoadDefault()
	if err != nil {
		return nil, fmt.Errorf("loading registry: %w", err)
	}

	projStore, err := project.LoadDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: loading projects: %v\n", err)
		projStore = project.NewStore()
	}

	profMgr := profile.NewDefaultManager()

	dep, ok := deploy.NewDeployer(model.ClientClaudeCode)
	if !ok {
		return nil, fmt.Errorf("no deployer for %s", model.ClientClaudeCode)
	}

	return service.New(reg, projStore, profMgr, dep), nil
}
