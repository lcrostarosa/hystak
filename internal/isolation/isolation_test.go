package isolation

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// initGitRepo creates a minimal git repo in dir with one commit.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %s: %v", args, out, err)
		}
	}
	run("init")
	run("commit", "--allow-empty", "-m", "initial")
}

// ---------- WorktreeManager ----------

func TestWorktreeCreate(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	wm := NewWorktreeManager()
	wtPath, err := wm.Create(dir, "frontend")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if !wm.Exists(dir, "frontend") {
		t.Fatal("worktree should exist after Create")
	}

	// Verify it's a real directory.
	info, err := os.Stat(wtPath)
	if err != nil {
		t.Fatalf("Stat worktree: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("worktree path should be a directory")
	}

	// Verify the .git file exists (worktrees have .git as a file, not a dir).
	gitFile := filepath.Join(wtPath, ".git")
	finfo, err := os.Stat(gitFile)
	if err != nil {
		t.Fatalf("worktree should have .git file: %v", err)
	}
	if finfo.IsDir() {
		t.Fatal("worktree .git should be a file, not a directory")
	}

	// Cleanup.
	t.Cleanup(func() { _ = wm.Remove(dir, "frontend") })
}

func TestWorktreeCreateIdempotent(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	wm := NewWorktreeManager()
	path1, err := wm.Create(dir, "backend")
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}

	path2, err := wm.Create(dir, "backend")
	if err != nil {
		t.Fatalf("second Create: %v", err)
	}

	if path1 != path2 {
		t.Fatalf("idempotent Create should return same path: %q vs %q", path1, path2)
	}

	t.Cleanup(func() { _ = wm.Remove(dir, "backend") })
}

func TestWorktreeRemove(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	wm := NewWorktreeManager()
	_, err := wm.Create(dir, "test")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := wm.Remove(dir, "test"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if wm.Exists(dir, "test") {
		t.Fatal("worktree should not exist after Remove")
	}
}

func TestWorktreeList(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	wm := NewWorktreeManager()

	// No worktrees yet.
	infos, err := wm.List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(infos) != 0 {
		t.Fatalf("expected 0 worktrees, got %d", len(infos))
	}

	// Create two worktrees.
	_, _ = wm.Create(dir, "alpha")
	_, _ = wm.Create(dir, "beta")

	infos, err = wm.List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(infos))
	}

	names := map[string]bool{}
	for _, info := range infos {
		names[info.ProfileName] = true
	}
	if !names["alpha"] || !names["beta"] {
		t.Fatalf("expected alpha and beta, got %v", names)
	}

	t.Cleanup(func() {
		_ = wm.Remove(dir, "alpha")
		_ = wm.Remove(dir, "beta")
	})
}

func TestWorktreeTwoWorktreesIndependent(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	wm := NewWorktreeManager()
	path1, _ := wm.Create(dir, "one")
	path2, _ := wm.Create(dir, "two")

	// Write a file in each worktree.
	os.WriteFile(filepath.Join(path1, "wt1.txt"), []byte("one"), 0o644)
	os.WriteFile(filepath.Join(path2, "wt2.txt"), []byte("two"), 0o644)

	// Each worktree should only have its own file.
	if _, err := os.Stat(filepath.Join(path1, "wt2.txt")); err == nil {
		t.Fatal("wt2.txt should not exist in worktree one")
	}
	if _, err := os.Stat(filepath.Join(path2, "wt1.txt")); err == nil {
		t.Fatal("wt1.txt should not exist in worktree two")
	}

	t.Cleanup(func() {
		_ = wm.Remove(dir, "one")
		_ = wm.Remove(dir, "two")
	})
}

func TestWorktreeNonGitRepo(t *testing.T) {
	dir := t.TempDir()

	wm := NewWorktreeManager()
	_, err := wm.Create(dir, "test")
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
}

func TestWorktreePath(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "myproject")
	os.Mkdir(projDir, 0o755)

	wm := NewWorktreeManager()
	p := wm.Path(projDir, "frontend")

	// Path should end with the expected suffix.
	if !strings.HasSuffix(p, "myproject.hystak-wt-frontend") {
		t.Fatalf("unexpected path: %s", p)
	}
}

// ---------- LockManager ----------

func TestLockAcquireRelease(t *testing.T) {
	dir := t.TempDir()
	lm := NewLockManager()

	if err := lm.Acquire(dir); err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	// Lock file should exist.
	data, err := os.ReadFile(filepath.Join(dir, lockFileName))
	if err != nil {
		t.Fatalf("reading lock file: %v", err)
	}
	pid, _ := strconv.Atoi(string(data))
	if pid != os.Getpid() {
		t.Fatalf("lock PID = %d, want %d", pid, os.Getpid())
	}

	// Should be locked.
	locked, lockPid, err := lm.IsLocked(dir)
	if err != nil {
		t.Fatalf("IsLocked: %v", err)
	}
	if !locked {
		t.Fatal("should be locked")
	}
	if lockPid != os.Getpid() {
		t.Fatalf("locked PID = %d, want %d", lockPid, os.Getpid())
	}

	// Release.
	if err := lm.Release(dir); err != nil {
		t.Fatalf("Release: %v", err)
	}

	locked, _, _ = lm.IsLocked(dir)
	if locked {
		t.Fatal("should not be locked after release")
	}
}

func TestLockPreventsDoubleAcquire(t *testing.T) {
	dir := t.TempDir()
	lm := NewLockManager()

	if err := lm.Acquire(dir); err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	defer lm.Release(dir)

	err := lm.Acquire(dir)
	if err == nil {
		t.Fatal("expected error on double Acquire")
	}
}

func TestLockStaleDetection(t *testing.T) {
	dir := t.TempDir()
	lm := NewLockManager()

	// Write a lock file with a PID that doesn't exist.
	// PID 99999999 is extremely unlikely to be running.
	os.WriteFile(filepath.Join(dir, lockFileName), []byte("99999999"), 0o644)

	locked, _, err := lm.IsLocked(dir)
	if err != nil {
		t.Fatalf("IsLocked: %v", err)
	}
	if locked {
		t.Fatal("stale lock should be auto-released")
	}

	// Lock file should have been removed.
	if _, err := os.Stat(filepath.Join(dir, lockFileName)); !os.IsNotExist(err) {
		t.Fatal("stale lock file should be removed")
	}
}

func TestLockMalformedFile(t *testing.T) {
	dir := t.TempDir()
	lm := NewLockManager()

	// Write a malformed lock file.
	os.WriteFile(filepath.Join(dir, lockFileName), []byte("not-a-pid"), 0o644)

	locked, _, err := lm.IsLocked(dir)
	if err != nil {
		t.Fatalf("IsLocked: %v", err)
	}
	if locked {
		t.Fatal("malformed lock should not report as locked")
	}

	// Lock file should have been removed.
	if _, err := os.Stat(filepath.Join(dir, lockFileName)); !os.IsNotExist(err) {
		t.Fatal("malformed lock file should be removed")
	}
}

func TestLockNoFile(t *testing.T) {
	dir := t.TempDir()
	lm := NewLockManager()

	locked, _, err := lm.IsLocked(dir)
	if err != nil {
		t.Fatalf("IsLocked: %v", err)
	}
	if locked {
		t.Fatal("should not be locked when no lock file exists")
	}
}

func TestLockReleaseIdempotent(t *testing.T) {
	dir := t.TempDir()
	lm := NewLockManager()

	// Release without acquire should not error.
	if err := lm.Release(dir); err != nil {
		t.Fatalf("Release without acquire: %v", err)
	}
}
