//go:build windows

package launch

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
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

	if err := cmd.Start(); err != nil {
		return -1, fmt.Errorf("starting %q: %w", executable, err)
	}

	// Forward interrupt signal to the child.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-sigCh:
				if cmd.Process != nil {
					_ = cmd.Process.Signal(os.Interrupt)
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
