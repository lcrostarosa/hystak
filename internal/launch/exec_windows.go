//go:build windows

package launch

import (
	"fmt"
	"os"
	"os/exec"
)

// Exec launches the given executable in dir, forwarding stdio and exit code.
// On Windows, syscall.Exec (execve) is not available, so we spawn a child process instead.
func Exec(executable string, args []string, dir string) error {
	cmd := exec.Command(executable, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("running %q: %w", executable, err)
	}
	os.Exit(0)
	return nil // unreachable
}
