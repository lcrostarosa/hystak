package isolation

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// LockManager implements lock-based isolation (S-065).
// Creates .hystak.lock with PID, detects held locks, cleans stale locks.
type LockManager struct{}

// lockFileName is the lock file name within a project directory.
const lockFileName = ".hystak.lock"

// LockPath returns the lock file path for a project.
func LockPath(projectPath string) string {
	return filepath.Join(projectPath, lockFileName)
}

// Acquire attempts to acquire a lock for the project. Returns an error
// if the lock is held by a running process. Cleans stale locks automatically.
func (lm *LockManager) Acquire(projectPath string) error {
	lockPath := LockPath(projectPath)

	data, err := os.ReadFile(lockPath)
	switch {
	case err == nil:
		// Lock file exists — check if holder is alive
		pid, parseErr := strconv.Atoi(strings.TrimSpace(string(data)))
		if parseErr == nil && isProcessRunning(pid) {
			return fmt.Errorf("lock held by PID %d at %s", pid, lockPath)
		}
		// Stale lock — clean it up
		if err := os.Remove(lockPath); err != nil {
			return fmt.Errorf("cleaning stale lock: %w", err)
		}
	case errors.Is(err, fs.ErrNotExist):
		// No lock, proceed
	default:
		return fmt.Errorf("reading lock file: %w", err)
	}

	// Write our PID
	pid := os.Getpid()
	if err := os.WriteFile(lockPath, []byte(strconv.Itoa(pid)), 0o644); err != nil {
		return fmt.Errorf("writing lock file: %w", err)
	}
	return nil
}

// Release removes the lock file if it's owned by the current process.
func (lm *LockManager) Release(projectPath string) error {
	lockPath := LockPath(projectPath)

	data, err := os.ReadFile(lockPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil // already released
		}
		return err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		// Corrupt lock file — remove it
		return os.Remove(lockPath)
	}

	if pid != os.Getpid() {
		return fmt.Errorf("lock owned by PID %d, not current process %d", pid, os.Getpid())
	}

	return os.Remove(lockPath)
}

// IsLocked reports whether the project lock is held by a running process.
func (lm *LockManager) IsLocked(projectPath string) (bool, int, error) {
	lockPath := LockPath(projectPath)

	data, err := os.ReadFile(lockPath)
	switch {
	case err == nil:
		pid, parseErr := strconv.Atoi(strings.TrimSpace(string(data)))
		if parseErr != nil {
			return false, 0, nil // corrupt, treat as unlocked
		}
		if isProcessRunning(pid) {
			return true, pid, nil
		}
		return false, 0, nil // stale
	case errors.Is(err, fs.ErrNotExist):
		return false, 0, nil
	default:
		return false, 0, err
	}
}

// isProcessRunning checks if a process with the given PID is alive
// using kill(pid, 0) which checks existence without sending a signal.
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
