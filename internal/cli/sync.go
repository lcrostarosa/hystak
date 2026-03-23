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

var syncCmd = &cobra.Command{
	Use:         "sync <project>",
	Short:       "Deploy project configs",
	Long:        "Resolve the active profile and deploy MCP servers to client config files.",
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{"mutates": "true"},
	RunE:        runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	svc, err := buildService()
	if err != nil {
		return err
	}

	results, err := svc.SyncProject(args[0])
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	for _, r := range results {
		if _, err := fmt.Fprintf(w, "%s\t%s\n", r.Name, r.Action); err != nil {
			return err
		}
	}
	return w.Flush()
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
