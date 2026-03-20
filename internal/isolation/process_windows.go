//go:build windows

package isolation

import "os"

// isProcessRunning checks if a process with the given PID is alive.
// On Windows, os.FindProcess always succeeds for valid PIDs, so we
// conservatively return true if the PID is positive. This means locks
// won't auto-expire on Windows, but users can manually release them.
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	_, err := os.FindProcess(pid)
	return err == nil
}
