package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const legacyHystakSentinel = "<!-- managed by hystak -->"

// ClaudeMDDeployer creates a symlink or composed file for CLAUDE.md in the project root.
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

	// Regular file — check for sentinel (also managed).
	content, err := os.ReadFile(target)
	if err != nil {
		return nil
	}
	if strings.HasPrefix(string(content), legacyHystakSentinel) {
		return nil // managed file, no conflict
	}

	return &PreflightConflict{
		ResourceType: "claude_md",
		Name:         "CLAUDE.md",
		ExistingPath: target,
	}
}

// SyncClaudeMD deploys CLAUDE.md to the project root.
//
// When promptSources is empty, the template is deployed as a symlink (original behavior).
// When promptSources is non-empty, a composed file is generated with sentinel header,
// template content (if any), and prompt fragment contents in order.
// Regular files (user-owned) are never overwritten unless they have the managed sentinel.
func (d *ClaudeMDDeployer) SyncClaudeMD(projectPath, templateSource string, promptSources []string) error {
	if templateSource == "" && len(promptSources) == 0 {
		return nil
	}

	// Composition mode: generate a file from template + prompts.
	if len(promptSources) > 0 {
		return d.syncComposed(projectPath, templateSource, promptSources)
	}

	// Symlink mode: template only, no prompts.
	return d.syncSymlink(projectPath, templateSource)
}

// syncSymlink deploys a template as a symlink (original behavior).
func (d *ClaudeMDDeployer) syncSymlink(projectPath, templateSource string) error {
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
			// Regular file — check for managed sentinel.
			content, err := os.ReadFile(target)
			if err != nil {
				return fmt.Errorf("reading existing CLAUDE.md: %w", err)
			}
			if strings.HasPrefix(string(content), legacyHystakSentinel) {
				// Managed file — replace with symlink.
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

// syncComposed generates CLAUDE.md from template + prompt fragments.
func (d *ClaudeMDDeployer) syncComposed(projectPath, templateSource string, promptSources []string) error {
	var buf strings.Builder
	buf.WriteString(legacyHystakSentinel)
	buf.WriteString("\n\n")

	// Include template content if set.
	if templateSource != "" {
		source := expandHome(templateSource)
		content, err := os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("reading template source %q: %w", source, err)
		}
		buf.Write(content)
		buf.WriteString("\n\n")
	}

	// Append each prompt fragment.
	for _, ps := range promptSources {
		source := expandHome(ps)
		content, err := os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("reading prompt source %q: %w", source, err)
		}
		buf.Write(content)
		buf.WriteString("\n\n")
	}

	composed := buf.String()
	target := filepath.Join(projectPath, "CLAUDE.md")

	// Handle existing file at target.
	info, err := os.Lstat(target)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			// Previous symlink — remove to write composed file.
			os.Remove(target)
		} else {
			// Regular file — check for managed sentinel.
			existing, err := os.ReadFile(target)
			if err != nil {
				return fmt.Errorf("reading existing CLAUDE.md: %w", err)
			}
			if !strings.HasPrefix(string(existing), legacyHystakSentinel) {
				// User-owned file — leave it alone.
				return nil
			}
			// Managed file — check if content is already correct.
			if string(existing) == composed {
				return nil // already up to date
			}
			// Will overwrite below.
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("checking existing CLAUDE.md: %w", err)
	}

	if err := os.WriteFile(target, []byte(composed), 0o644); err != nil {
		return fmt.Errorf("writing composed CLAUDE.md: %w", err)
	}

	return nil
}

// IsClaudeMDManaged returns true if project/CLAUDE.md is managed by hystak
// (deployed as a symlink or as a generated file with the managed sentinel).
func (d *ClaudeMDDeployer) IsClaudeMDManaged(projectPath string) bool {
	target := filepath.Join(projectPath, "CLAUDE.md")

	if isSymlink(target) {
		return true
	}

	content, err := os.ReadFile(target)
	if err != nil {
		return false
	}
	return strings.HasPrefix(string(content), legacyHystakSentinel)
}
