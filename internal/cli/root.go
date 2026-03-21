package cli

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"github.com/lcrostarosa/hystak/internal/config"
	"github.com/lcrostarosa/hystak/internal/profile"
	"github.com/lcrostarosa/hystak/internal/service"
	"github.com/lcrostarosa/hystak/internal/tui"
	"github.com/spf13/cobra"
)

// cliApp holds the shared service instance for all subcommands.
type cliApp struct {
	svc     *service.Service
	version string
}

// newRootCmd builds the full command tree.
func newRootCmd(version, commit, date string) *cobra.Command {
	var cfgDir string
	var configure string
	app := &cliApp{version: version}

	root := &cobra.Command{
		Use:   "hystak [-- claude-args...]",
		Short: "Claude Code launcher with profile management",
		Long: `hystak manages MCP server configurations, skills, hooks, and permissions
from a central registry, syncs them to client config files, and launches
Claude Code with the selected profile.

Run with no arguments to pick a profile interactively, or use subcommands
for non-interactive workflows.

Arguments after -- are forwarded to the claude process.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cfgDir == "" {
				cfgDir = config.ConfigDir()
			}

			// Migrate from legacy ~/.config/hystak/ if needed.
			if warning, err := config.Migrate(); err != nil {
				return fmt.Errorf("config migration: %w", err)
			} else if warning != "" {
				fmt.Fprintln(cmd.ErrOrStderr(), warning)
			}

			if err := config.EnsureConfigDir(); err != nil {
				return fmt.Errorf("creating config directory: %w", err)
			}
			var err error
			app.svc, err = service.New(cfgDir)
			if err != nil {
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Split args at -- separator.
			var extraArgs []string
			if dashAt := cmd.ArgsLenAtDash(); dashAt >= 0 {
				extraArgs = args[dashAt:]
				args = args[:dashAt]
			}

			if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
				return cmd.Help()
			}

			// --configure flag: open wizard in hub mode for the specified project.
			if configure != "" {
				return app.runWizardAndLaunch(cmd, configure, tui.LWModeHub, extraArgs)
			}

			// First-time setup wizard.
			if app.svc.IsEmpty() {
				wizard := tui.NewWizardModel(app.svc)
				wp := tea.NewProgram(wizard, tea.WithAltScreen())
				if _, err := wp.Run(); err != nil {
					return err
				}
			}

			// Show picker.
			picker := tui.NewPickerModel(app.svc, app.version)
			p := tea.NewProgram(picker, tea.WithAltScreen())
			result, err := p.Run()
			if err != nil {
				return err
			}

			pickerResult := result.(tui.PickerModel).Result()
			if pickerResult == nil {
				// User cancelled.
				return nil
			}

			if pickerResult.Manage {
				return app.runManageTUI(cmd, extraArgs)
			}

			// Configure mode: open wizard in hub mode (check before nil-project fallthrough).
			if pickerResult.Configure && pickerResult.Project != nil {
				proj := *pickerResult.Project
				return app.runWizardAndLaunch(cmd, proj.Name, tui.LWModeHub, extraArgs)
			}

			if pickerResult.Project == nil {
				// Launch without profile.
				return launchBare(cmd, extraArgs)
			}

			proj := *pickerResult.Project

			// First-time launch: walk through all steps sequentially.
			// Returning project: open hub mode so user can review/change profile.
			mode := tui.LWModeHub
			if !app.svc.HasLaunched(proj.Name) {
				mode = tui.LWModeSequential
			}
			return app.runWizardAndLaunch(cmd, proj.Name, mode, extraArgs)
		},
		SilenceUsage: true,
	}

	root.PersistentFlags().StringVar(&cfgDir, "config-dir", "", "config directory (default: ~/.hystak)")
	root.Flags().StringVar(&configure, "configure", "", "open launch wizard for the specified project")

	root.AddCommand(app.newListCmd())
	root.AddCommand(app.newSyncCmd())
	root.AddCommand(app.newImportCmd())
	root.AddCommand(app.newOverrideCmd())
	root.AddCommand(app.newDiffCmd())
	root.AddCommand(app.newRunCmd())
	root.AddCommand(app.newBackupCmd())
	root.AddCommand(app.newRestoreCmd())
	root.AddCommand(app.newManageCmd())
	root.AddCommand(app.newSetupCmd())
	root.AddCommand(app.newProfileCmd())
	root.AddCommand(newVersionCmd(version, commit, date))

	return root
}

// runManageTUI launches the management TUI and handles a launch request if emitted.
func (a *cliApp) runManageTUI(cmd *cobra.Command, extraArgs []string) error {
	m := tui.NewApp(a.svc)
	mp := tea.NewProgram(m, tea.WithAltScreen())
	result, err := mp.Run()
	if err != nil {
		return err
	}
	if app, ok := result.(tui.AppModel); ok {
		if proj := app.LaunchRequest(); proj != nil {
			return a.syncAndLaunch(cmd, *proj, extraArgs, false, false)
		}
	}
	return nil
}

// runWizardAndLaunch runs the launch wizard for a project, saves the resulting profile,
// and optionally syncs and launches.
func (a *cliApp) runWizardAndLaunch(cmd *cobra.Command, projectName string, mode tui.LaunchWizardMode, extraArgs []string) error {
	proj, ok := a.svc.GetProject(projectName)
	if !ok {
		return fmt.Errorf("project %q not found", projectName)
	}

	// Run discovery for this project.
	discovered := a.svc.Discover(proj.Path)

	// Load existing profile for pre-population if the project has been launched before.
	var existingProfile *profile.Profile
	if activeName, _ := a.svc.GetActiveProfile(projectName); activeName != "" {
		// loadProfile is internal, so reconstruct from project profile data.
		if pp, ok := proj.Profiles[activeName]; ok {
			existingProfile = &profile.Profile{
				Name:        activeName,
				Description: pp.Description,
				MCPs:        pp.MCPs,
				Skills:      pp.Skills,
				Hooks:       pp.Hooks,
				Permissions: pp.Permissions,
				EnvVars:     pp.EnvVars,
				ClaudeMD:    pp.ClaudeMD,
				Isolation:   profile.IsolationStrategy(pp.Isolation),
			}
		}
	}

	// Run the launch wizard as a standalone Bubble Tea program.
	wiz := tui.NewLaunchWizardModel(&proj, mode, discovered, existingProfile)
	wrapper := newWizardWrapper(wiz)
	p := tea.NewProgram(wrapper, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return err
	}

	wr := result.(wizardWrapper)
	if wr.cancelled {
		return nil
	}

	prof := wr.profile
	profileName := "default"

	// Save the profile to the project.
	if err := a.svc.SaveProjectProfile(projectName, profileName, prof); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	// Set as active profile.
	if err := a.svc.SetActiveProfile(projectName, profileName); err != nil {
		return fmt.Errorf("setting active profile: %w", err)
	}

	// Mark project as launched.
	if err := a.svc.MarkLaunched(projectName); err != nil {
		return fmt.Errorf("marking launched: %w", err)
	}

	if !wr.launch {
		fmt.Fprintln(cmd.OutOrStdout(), "Profile saved.")
		return nil
	}

	// Sync and launch.
	// Reload project since we just modified it.
	proj, _ = a.svc.GetProject(projectName)
	return a.syncAndLaunch(cmd, proj, extraArgs, false, false)
}

// wizardWrapper wraps LaunchWizardModel to implement tea.Model for standalone use.
type wizardWrapper struct {
	inner     tui.LaunchWizardModel
	profile   profile.Profile
	launch    bool
	cancelled bool
}

func newWizardWrapper(m tui.LaunchWizardModel) wizardWrapper {
	return wizardWrapper{inner: m}
}

func (w wizardWrapper) Init() tea.Cmd {
	return w.inner.Init()
}

func (w wizardWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tui.LaunchWizardCompleteMsg:
		complete := msg.(tui.LaunchWizardCompleteMsg)
		w.profile = complete.Profile
		w.launch = complete.Launch
		return w, tea.Quit

	case tui.LaunchWizardCancelledMsg:
		w.cancelled = true
		return w, tea.Quit
	}

	var cmd tea.Cmd
	w.inner, cmd = w.inner.Update(msg)
	return w, cmd
}

func (w wizardWrapper) View() string {
	return w.inner.View()
}

// newTeaProgram creates a Bubble Tea program for the wizard wrapper.
func newTeaProgram(wrapper wizardWrapper) *tea.Program {
	return tea.NewProgram(wrapper, tea.WithAltScreen())
}

// Execute runs the CLI with the given version info.
func Execute(version, commit, date string) {
	if err := newRootCmd(version, commit, date).Execute(); err != nil {
		os.Exit(1)
	}
}
