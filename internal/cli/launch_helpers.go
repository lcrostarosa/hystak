package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/lcrostarosa/hystak/internal/isolation"
	"github.com/lcrostarosa/hystak/internal/launch"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/profile"
	"github.com/lcrostarosa/hystak/internal/service"
	"github.com/spf13/cobra"
)

// syncAndLaunch syncs a project's configs and exec's the client.
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

	return launch.Exec(execPath, extraArgs, workDir)
}

// launchBare resolves "claude" on PATH and exec's in the current directory.
func launchBare(extraArgs []string) error {
	execPath, err := launch.ResolveExecutable("claude")
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	return launch.Exec(execPath, extraArgs, cwd)
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
