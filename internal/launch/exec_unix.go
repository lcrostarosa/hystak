//go:build !windows

package launch

import (
	"fmt"
	"os"
	"syscall"
)

// Exec replaces the current process with the given executable, running in dir.
func Exec(executable string, args []string, dir string) error {
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("changing to directory %q: %w", dir, err)
	}

	argv := append([]string{executable}, args...)
	return syscall.Exec(executable, argv, os.Environ())
}
