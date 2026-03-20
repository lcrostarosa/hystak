//go:build !windows

package isolation

import (
	"os"
	"syscall"
)

// isProcessRunning checks if a process with the given PID is alive.
// On Unix, Signal(0) checks existence without sending a signal.
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	return err == nil
}
