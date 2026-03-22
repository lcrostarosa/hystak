package launch

import (
	"fmt"
	"os"
	"os/exec"
)

// RunCommand spawns the given command as a child process in the specified
// working directory. Stdin, stdout, and stderr are connected to the parent.
// The parent ignores SIGINT/SIGTERM while the child runs so that the child
// handles signals itself.
func RunCommand(name string, args []string, dir string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	ignoreSignals()
	defer restoreSignals()

	return cmd.Run()
}

// LaunchClaudeCode runs the `claude` command in the given working directory.
func LaunchClaudeCode(workDir string) error {
	clientPath, err := FindClient(DefaultClientCommand())
	if err != nil {
		return err
	}
	return RunCommand(clientPath, nil, workDir)
}

// FindClient returns the path to the client binary, or an error if not found.
func FindClient(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("client %q not found in PATH: %w", name, err)
	}
	return path, nil
}

// DefaultClientCommand returns the default Claude Code command name.
func DefaultClientCommand() string {
	return "claude"
}
