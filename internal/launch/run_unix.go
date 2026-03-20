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
func RunCommand(executable string, args []string, dir string) (int, error) {
	cmd := exec.Command(executable, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Give the child its own process group so Ctrl+C goes to it, not us.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return -1, fmt.Errorf("starting %q: %w", executable, err)
	}

	// Forward SIGINT and SIGTERM to the child process group.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case sig := <-sigCh:
				if cmd.Process != nil {
					// Send to the child's process group (negative PID).
					_ = syscall.Kill(-cmd.Process.Pid, sig.(syscall.Signal))
				}
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
