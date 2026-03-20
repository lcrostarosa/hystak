package isolation

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WorktreeInfo describes an active git worktree managed by hystak.
type WorktreeInfo struct {
	ProfileName string
	Path        string
	Branch      string
}

// WorktreeManager creates and manages git worktrees for session isolation.
type WorktreeManager struct{}

// NewWorktreeManager creates a new WorktreeManager.
func NewWorktreeManager() *WorktreeManager {
	return &WorktreeManager{}
}

// worktreePath returns the predictable path for a hystak worktree.
// Worktrees are placed as sibling directories: <projectPath>.hystak-wt-<profileName>
func worktreePath(projectPath, profileName string) string {
	abs := resolveAbsPath(projectPath)
	return abs + ".hystak-wt-" + profileName
}

// resolveAbsPath returns the absolute, symlink-resolved path.
// This ensures consistent paths on systems where temp dirs use symlinks
// (e.g., macOS /var → /private/var).
func resolveAbsPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs
	}
	return resolved
}

// Path returns the worktree directory path without creating it.
func (m *WorktreeManager) Path(projectPath, profileName string) string {
	return worktreePath(projectPath, profileName)
}

// Create creates a git worktree for the given profile.
// If the worktree already exists, it is returned without modification.
// Returns the worktree path.
func (m *WorktreeManager) Create(projectPath, profileName string) (string, error) {
	if err := validateGitRepo(projectPath); err != nil {
		return "", err
	}

	wtPath := worktreePath(projectPath, profileName)

	// If worktree already exists, return it.
	if m.Exists(projectPath, profileName) {
		return wtPath, nil
	}

	// Create the worktree detached at HEAD.
	cmd := exec.Command("git", "worktree", "add", "--detach", wtPath)
	cmd.Dir = projectPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("creating worktree: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return wtPath, nil
}

// Exists checks if a worktree exists for the given profile.
func (m *WorktreeManager) Exists(projectPath, profileName string) bool {
	wtPath := worktreePath(projectPath, profileName)
	info, err := os.Stat(wtPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// Remove removes a worktree for the given profile.
func (m *WorktreeManager) Remove(projectPath, profileName string) error {
	wtPath := worktreePath(projectPath, profileName)

	cmd := exec.Command("git", "worktree", "remove", "--force", wtPath)
	cmd.Dir = projectPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("removing worktree: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return nil
}

// List returns all hystak-managed worktrees for a project.
// Worktrees are identified by the .hystak-wt- path prefix convention.
func (m *WorktreeManager) List(projectPath string) ([]WorktreeInfo, error) {
	if err := validateGitRepo(projectPath); err != nil {
		return nil, err
	}

	abs := resolveAbsPath(projectPath)
	prefix := abs + ".hystak-wt-"

	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = projectPath
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing worktrees: %w", err)
	}

	var infos []WorktreeInfo
	var currentPath, currentBranch string
	detached := false

	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			// Flush previous entry.
			if currentPath != "" && strings.HasPrefix(currentPath, prefix) {
				infos = append(infos, WorktreeInfo{
					ProfileName: strings.TrimPrefix(currentPath, prefix),
					Path:        currentPath,
					Branch:      currentBranch,
				})
			}
			currentPath = strings.TrimPrefix(line, "worktree ")
			currentBranch = ""
			detached = false
		case strings.HasPrefix(line, "branch "):
			currentBranch = strings.TrimPrefix(line, "branch ")
		case line == "detached":
			detached = true
			_ = detached // used for context, branch will be empty
		}
	}

	// Flush last entry.
	if currentPath != "" && strings.HasPrefix(currentPath, prefix) {
		infos = append(infos, WorktreeInfo{
			ProfileName: strings.TrimPrefix(currentPath, prefix),
			Path:        currentPath,
			Branch:      currentBranch,
		})
	}

	return infos, nil
}

// validateGitRepo checks that the given path is inside a git repository.
func validateGitRepo(path string) error {
	gitDir := filepath.Join(path, ".git")
	_, err := os.Stat(gitDir)
	if err != nil {
		return fmt.Errorf("%q is not a git repository (no .git found)", path)
	}
	return nil
}
