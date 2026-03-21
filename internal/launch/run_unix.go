//go:build !windows

package launch

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// RunCommand launches the given executable as a child process in dir,
// forwarding stdio and signals. It returns the exit code when the child exits.
// Unlike Exec, the calling process survives, enabling post-exit actions.
//
// The child shares the parent's process group so it inherits foreground
// terminal ownership. This is required for interactive TUI children (like
// Claude Code) that read from the terminal — a background process group
// would receive SIGTTIN and be stopped by the kernel.
//
// The parent ignores SIGINT/SIGTERM while the child runs so that Ctrl+C
// is handled by the child (which is in the foreground group) rather than
// killing the parent prematurely.
func RunCommand(executable string, args []string, dir string) (int, error) {
	cmd := exec.Command(executable, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return -1, fmt.Errorf("starting %q: %w", executable, err)
	}

	// Ignore SIGINT and SIGTERM in the parent while the child runs.
	// The child is in our process group, so the terminal delivers signals
	// directly to it. We just need to avoid exiting before it does.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-sigCh:
				// Swallow — let the child handle it.
			case <-done:
				return
			}
		}
	}()

	err := cmd.Wait()
	close(done)
	signal.Stop(sigCh)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return -1, fmt.Errorf("waiting for %q: %w", executable, err)
	}
	return 0, nil
}
