package isolation

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const lockFileName = ".hystak.lock"

// LockManager manages per-project lock files for preventing concurrent sessions.
type LockManager struct{}

// NewLockManager creates a new LockManager.
func NewLockManager() *LockManager {
	return &LockManager{}
}

func lockPath(projectPath string) string {
	return filepath.Join(projectPath, lockFileName)
}

// Acquire creates a lock file with the current PID.
// Returns an error if the project is already locked by a running process.
func (m *LockManager) Acquire(projectPath string) error {
	locked, pid, err := m.IsLocked(projectPath)
	if err != nil {
		return err
	}
	if locked {
		return fmt.Errorf("project is locked by PID %d", pid)
	}

	data := []byte(strconv.Itoa(os.Getpid()))
	if err := os.WriteFile(lockPath(projectPath), data, 0o644); err != nil {
		return fmt.Errorf("writing lock file: %w", err)
	}

	return nil
}

// Release removes the lock file.
func (m *LockManager) Release(projectPath string) error {
	path := lockPath(projectPath)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing lock file: %w", err)
	}
	return nil
}

// IsLocked checks if the project is locked by a running process.
// A stale lock (PID no longer running) is automatically released.
// Returns (locked, pid, error).
func (m *LockManager) IsLocked(projectPath string) (bool, int, error) {
	data, err := os.ReadFile(lockPath(projectPath))
	if err != nil {
		if os.IsNotExist(err) {
			return false, 0, nil
		}
		return false, 0, fmt.Errorf("reading lock file: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		// Malformed lock file — remove it.
		_ = os.Remove(lockPath(projectPath))
		return false, 0, nil
	}

	// Check if the process is still running.
	if isProcessRunning(pid) {
		return true, pid, nil
	}

	// Stale lock — remove it.
	_ = os.Remove(lockPath(projectPath))
	return false, 0, nil
}
