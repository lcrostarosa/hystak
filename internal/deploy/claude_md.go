package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const legacyHystakSentinel = "<!-- managed by hystak -->"

// ClaudeMDDeployer creates a symlink for CLAUDE.md in the project root.
type ClaudeMDDeployer struct{}

// PreflightClaudeMD checks if CLAUDE.md exists and is not managed by hystak.
// Returns a conflict if CLAUDE.md is a regular file that hystak didn't place.
func (d *ClaudeMDDeployer) PreflightClaudeMD(projectPath string) *PreflightConflict {
	target := filepath.Join(projectPath, "CLAUDE.md")

	info, err := os.Lstat(target)
	if err != nil {
		return nil // doesn't exist, no conflict
	}

	// Symlink = managed by hystak, no conflict.
	if info.Mode()&os.ModeSymlink != 0 {
		return nil
	}

	// Regular file — check for legacy sentinel (also managed).
	content, err := os.ReadFile(target)
	if err != nil {
		return nil
	}
	if strings.HasPrefix(string(content), legacyHystakSentinel) {
		return nil // legacy managed file, no conflict
	}

	return &PreflightConflict{
		ResourceType: "claude_md",
		Name:         "CLAUDE.md",
		ExistingPath: target,
	}
}

// SyncClaudeMD deploys a template as a symlink at project/CLAUDE.md.
// Regular files (user-owned) are never overwritten unless they have the legacy sentinel.
func (d *ClaudeMDDeployer) SyncClaudeMD(projectPath, templateSource string) error {
	if templateSource == "" {
		return nil
	}

	source := expandHome(templateSource)
	if _, err := os.Stat(source); err != nil {
		return fmt.Errorf("template source %q does not exist: %w", source, err)
	}

	target := filepath.Join(projectPath, "CLAUDE.md")

	info, err := os.Lstat(target)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			// Existing symlink — check if it already points to the right source.
			existing, err := os.Readlink(target)
			if err == nil && existing == source {
				return nil // already correct
			}
			// Wrong target — remove and recreate.
			os.Remove(target)
		} else {
			// Regular file — check for legacy sentinel.
			content, err := os.ReadFile(target)
			if err != nil {
				return fmt.Errorf("reading existing CLAUDE.md: %w", err)
			}
			if strings.HasPrefix(string(content), legacyHystakSentinel) {
				// Legacy managed file — migrate to symlink.
				os.Remove(target)
			} else {
				// User-owned file — leave it alone.
				return nil
			}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("checking existing CLAUDE.md: %w", err)
	}

	if err := os.Symlink(source, target); err != nil {
		return fmt.Errorf("creating CLAUDE.md symlink: %w", err)
	}

	return nil
}

// IsClaudeMDManaged returns true if project/CLAUDE.md is managed by hystak
// (deployed as a symlink).
func (d *ClaudeMDDeployer) IsClaudeMDManaged(projectPath string) bool {
	return isSymlink(filepath.Join(projectPath, "CLAUDE.md"))
}
