package isolation

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WorktreeManager creates and reuses git worktrees for isolation (S-064).
type WorktreeManager struct{}

// WorktreePath returns the worktree path for a project+profile combination.
func WorktreePath(projectPath, profileName string) string {
	return projectPath + ".hystak-wt-" + profileName
}

// Create creates a git worktree. Reuses an existing one if it already exists.
// Returns the worktree path. Errors if the project is not a git repo.
func (wm *WorktreeManager) Create(projectPath, profileName string) (string, error) {
	wtPath := WorktreePath(projectPath, profileName)

	// Check if worktree already exists
	info, err := os.Stat(wtPath)
	if err == nil && info.IsDir() {
		return wtPath, nil // reuse existing
	}

	// Verify project is a git repo
	gitDir := filepath.Join(projectPath, ".git")
	gInfo, err := os.Stat(gitDir)
	if err != nil || !gInfo.IsDir() {
		return "", fmt.Errorf("worktree isolation requires a git repository at %q", projectPath)
	}

	// Create worktree
	cmd := exec.Command("git", "worktree", "add", wtPath)
	cmd.Dir = projectPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("creating worktree: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return wtPath, nil
}

// Remove removes a git worktree.
func (wm *WorktreeManager) Remove(projectPath, profileName string) error {
	wtPath := WorktreePath(projectPath, profileName)

	cmd := exec.Command("git", "worktree", "remove", wtPath)
	cmd.Dir = projectPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("removing worktree: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// Exists checks whether a worktree directory exists for the given project+profile.
func (wm *WorktreeManager) Exists(projectPath, profileName string) bool {
	wtPath := WorktreePath(projectPath, profileName)
	info, err := os.Stat(wtPath)
	return err == nil && info.IsDir()
}
