package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lcrostarosa/hystak/internal/isolation"
	"github.com/lcrostarosa/hystak/internal/launch"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/profile"
	"github.com/lcrostarosa/hystak/internal/service"
	"github.com/lcrostarosa/hystak/internal/tui"
	"github.com/spf13/cobra"
)

// postExitAction represents what the user wants after Claude exits.
type postExitAction int

const (
	actionRelaunch  postExitAction = iota
	actionConfigure
	actionQuit
)

// syncAndLaunch syncs a project's configs and launches the client with a
// post-exit reconfiguration loop. When Claude exits, the user is prompted to
// relaunch, reconfigure, or quit.
func (a *cliApp) syncAndLaunch(cmd *cobra.Command, proj model.Project, extraArgs []string, noSync, dryRun bool) error {
	// Determine client executable.
	if len(proj.Clients) == 0 {
		return fmt.Errorf("project %q has no clients configured", proj.Name)
	}
	execName, err := launch.DefaultExecutable(proj.Clients[0])
	if err != nil {
		return fmt.Errorf("cannot determine default client for project %q: %w", proj.Name, err)
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

	// Check isolation strategy from the active profile.
	isoStrategy := getIsolation(proj)
	deployPath := workDir

	switch isoStrategy {
	case profile.IsolationWorktree:
		wm := isolation.NewWorktreeManager()
		profileName := proj.ActiveProfile
		if profileName == "" {
			profileName = "default"
		}
		wtPath, err := wm.Create(workDir, profileName)
		if err != nil {
			return fmt.Errorf("creating worktree for profile %q: %w", profileName, err)
		}
		deployPath = wtPath
		workDir = wtPath

	case profile.IsolationLock:
		lm := isolation.NewLockManager()
		if err := lm.Acquire(workDir); err != nil {
			return fmt.Errorf("cannot launch: %s", err)
		}
		// Lock is released via PID-based stale detection when the process exits.
	}

	// Sync unless --no-sync.
	if !noSync {
		if err := a.syncDeploy(cmd, proj, deployPath); err != nil {
			return err
		}
	}

	if dryRun {
		var w io.Writer = os.Stderr
		if cmd != nil {
			w = cmd.ErrOrStderr()
		}
		fmt.Fprintf(w, "Would run: %s %v\n", execName, extraArgs)
		fmt.Fprintf(w, "Directory: %s\n", workDir)
		if isoStrategy == profile.IsolationWorktree {
			fmt.Fprintf(w, "Isolation: worktree at %s\n", workDir)
		} else if isoStrategy == profile.IsolationLock {
			fmt.Fprintf(w, "Isolation: lock\n")
		}
		return nil
	}

	// Resolve executable only when actually launching.
	execPath, err := launch.ResolveExecutable(execName)
	if err != nil {
		return err
	}

	// Launch with reconfiguration loop.
	return a.launchLoop(cmd, proj, execPath, extraArgs, workDir, deployPath, noSync)
}

// launchLoop runs Claude as a child process, and on exit offers
// Relaunch / Configure / Quit.
func (a *cliApp) launchLoop(cmd *cobra.Command, proj model.Project, execPath string, extraArgs []string, workDir, deployPath string, noSync bool) error {
	isRelaunch := false

	for {
		launchArgs := extraArgs
		if isRelaunch {
			launchArgs = appendContinue(extraArgs)
		}

		exitCode, err := launch.RunCommand(execPath, launchArgs, workDir)
		if err != nil {
			return err
		}

		// Get the reader for post-exit prompt.
		var reader io.Reader = os.Stdin
		if cmd != nil {
			reader = cmd.InOrStdin()
		}
		var w io.Writer = os.Stderr
		if cmd != nil {
			w = cmd.ErrOrStderr()
		}

		fmt.Fprintf(w, "\nClaude exited (code %d).\n", exitCode)
		action := promptPostExit(reader, w)

		switch action {
		case actionQuit:
			return nil

		case actionRelaunch:
			// Re-sync before relaunch.
			if !noSync {
				if err := a.syncDeploy(cmd, proj, deployPath); err != nil {
					fmt.Fprintf(w, "Warning: sync failed: %v\n", err)
				}
			}
			isRelaunch = true

		case actionConfigure:
			if err := a.reconfigure(cmd, proj); err != nil {
				fmt.Fprintf(w, "Reconfigure failed: %v\n", err)
				continue
			}
			// Reload project after reconfiguration.
			if updated, ok := a.svc.GetProject(proj.Name); ok {
				proj = updated
			}
			// Re-sync after reconfiguration.
			if !noSync {
				if err := a.syncDeploy(cmd, proj, deployPath); err != nil {
					fmt.Fprintf(w, "Warning: sync failed: %v\n", err)
				}
			}
			isRelaunch = true
		}
	}
}

// reconfigure opens the launch wizard in hub mode for the given project.
func (a *cliApp) reconfigure(cmd *cobra.Command, proj model.Project) error {
	discovered := a.svc.Discover(proj.Path)

	var existingProfile *profile.Profile
	if activeName, _ := a.svc.GetActiveProfile(proj.Name); activeName != "" {
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

	wiz := tui.NewLaunchWizardModel(&proj, tui.LWModeHub, discovered, existingProfile)
	wrapper := newWizardWrapper(wiz)

	// Use inline import to avoid circular deps — tea is already imported in root.go.
	// We rely on newWizardWrapper being in the same package.
	return a.runWizardSaveProfile(cmd, proj.Name, wrapper)
}

// runWizardSaveProfile runs a wizard wrapper and saves the result.
func (a *cliApp) runWizardSaveProfile(cmd *cobra.Command, projectName string, wrapper wizardWrapper) error {
	// Import tea locally to run program.
	p := newTeaProgram(wrapper)
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

	if err := a.svc.SaveProjectProfile(projectName, profileName, prof); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}
	if err := a.svc.SetActiveProfile(projectName, profileName); err != nil {
		return fmt.Errorf("setting active profile: %w", err)
	}

	return nil
}

// syncDeploy syncs project config to the deploy path.
func (a *cliApp) syncDeploy(cmd *cobra.Command, proj model.Project, deployPath string) error {
	var results []service.SyncResult
	var syncErr error
	if deployPath != proj.Path {
		results, syncErr = a.svc.SyncProjectToPath(proj.Name, deployPath)
	} else {
		results, syncErr = a.svc.SyncProject(proj.Name)
	}
	if syncErr != nil {
		return fmt.Errorf("sync failed: %w", syncErr)
	}
	if cmd != nil {
		printSyncResults(cmd, proj.Name, results)
	}
	return nil
}

// promptPostExit shows the post-exit prompt and reads the user's choice.
func promptPostExit(reader io.Reader, w io.Writer) postExitAction {
	fmt.Fprintf(w, "[R]elaunch / [C]onfigure / [Q]uit: ")

	scanner := bufio.NewScanner(reader)
	if scanner.Scan() {
		line := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if len(line) > 0 {
			switch line[0] {
			case 'r':
				return actionRelaunch
			case 'c':
				return actionConfigure
			}
		}
	}
	return actionQuit
}

// appendContinue appends --continue to args if not already present.
func appendContinue(args []string) []string {
	for _, a := range args {
		if a == "--continue" {
			return args
		}
	}
	result := make([]string, len(args), len(args)+1)
	copy(result, args)
	return append(result, "--continue")
}

// launchBare resolves "claude" on PATH and runs it in the current directory
// with a post-exit reconfiguration loop (relaunch or quit only).
func launchBare(cmd *cobra.Command, extraArgs []string) error {
	execPath, err := launch.ResolveExecutable("claude")
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	isRelaunch := false
	for {
		launchArgs := extraArgs
		if isRelaunch {
			launchArgs = appendContinue(extraArgs)
		}

		exitCode, err := launch.RunCommand(execPath, launchArgs, cwd)
		if err != nil {
			return err
		}

		var reader io.Reader = os.Stdin
		if cmd != nil {
			reader = cmd.InOrStdin()
		}
		var w io.Writer = os.Stderr
		if cmd != nil {
			w = cmd.ErrOrStderr()
		}

		fmt.Fprintf(w, "\nClaude exited (code %d).\n", exitCode)
		fmt.Fprintf(w, "[R]elaunch / [Q]uit: ")

		scanner := bufio.NewScanner(reader)
		if scanner.Scan() {
			line := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if len(line) > 0 && line[0] == 'r' {
				isRelaunch = true
				continue
			}
		}
		return nil
	}
}

// getIsolation returns the isolation strategy from the project's active profile.
func getIsolation(proj model.Project) profile.IsolationStrategy {
	if proj.ActiveProfile == "" {
		return profile.IsolationNone
	}
	pp, ok := proj.Profiles[proj.ActiveProfile]
	if !ok {
		return profile.IsolationNone
	}
	switch profile.IsolationStrategy(pp.Isolation) {
	case profile.IsolationWorktree, profile.IsolationLock:
		return profile.IsolationStrategy(pp.Isolation)
	default:
		return profile.IsolationNone
	}
}
