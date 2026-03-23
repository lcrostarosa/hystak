package isolation

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

func TestLockManager_AcquireRelease(t *testing.T) {
	tmp := t.TempDir()
	lm := &LockManager{}

	if err := lm.Acquire(tmp); err != nil {
		t.Fatal(err)
	}

	// Lock file should contain our PID
	lockPath := filepath.Join(tmp, lockFileName)
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		t.Fatal(err)
	}
	if pid != os.Getpid() {
		t.Errorf("lock PID = %d, want %d", pid, os.Getpid())
	}

	// IsLocked should return true
	locked, holderPID, err := lm.IsLocked(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if !locked {
		t.Error("expected locked")
	}
	if holderPID != os.Getpid() {
		t.Errorf("holder PID = %d, want %d", holderPID, os.Getpid())
	}

	// Release
	if err := lm.Release(tmp); err != nil {
		t.Fatal(err)
	}

	// Lock file should be gone
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("lock file should be removed after release")
	}
}

func TestLockManager_StaleLockCleanup(t *testing.T) {
	tmp := t.TempDir()
	lm := &LockManager{}

	// Write a stale lock with a dead PID
	deadPID := deadPID(t)
	lockPath := filepath.Join(tmp, lockFileName)
	if err := os.WriteFile(lockPath, []byte(strconv.Itoa(deadPID)), 0o644); err != nil {
		t.Fatal(err)
	}

	// Acquire should succeed (stale lock cleaned)
	if err := lm.Acquire(tmp); err != nil {
		t.Fatalf("expected stale lock to be cleaned, got: %v", err)
	}

	// Verify our PID is now in the lock
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	pid, _ := strconv.Atoi(string(data))
	if pid != os.Getpid() {
		t.Errorf("lock PID = %d, want %d (after stale cleanup)", pid, os.Getpid())
	}

	if err := lm.Release(tmp); err != nil {
		t.Fatal(err)
	}
}

func TestLockManager_IsLocked_NoLock(t *testing.T) {
	tmp := t.TempDir()
	lm := &LockManager{}

	locked, _, err := lm.IsLocked(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if locked {
		t.Error("expected not locked when no lock file")
	}
}

// deadPID returns a PID that is guaranteed to not be running.
func deadPID(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	return cmd.Process.Pid
}
