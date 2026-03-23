package deploy

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hystak/hystak/internal/model"
)

// Compile-time interface check.
var _ ResourceDeployer = (*SkillsDeployer)(nil)

// SkillsDeployer deploys skills as symlinks to .claude/skills/<name>/SKILL.md (S-043).
// Symlinks are managed by hystak; regular files are user-owned and not overwritten.
type SkillsDeployer struct{}

func (d *SkillsDeployer) Kind() ResourceDeployerKind {
	return ResourceDeployerSkills
}

// Sync creates or updates symlinks for each skill, and removes stale ones.
func (d *SkillsDeployer) Sync(projectPath string, config DeployConfig) error {
	skillsDir := filepath.Join(projectPath, ".claude", "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return fmt.Errorf("creating skills directory: %w", err)
	}

	// Build set of expected skill names
	expected := make(map[string]bool, len(config.Skills))
	for _, skill := range config.Skills {
		expected[skill.Name] = true
		if err := d.deploySkill(skillsDir, skill); err != nil {
			return fmt.Errorf("deploying skill %q: %w", skill.Name, err)
		}
	}

	// Remove stale symlinks (managed by hystak, not in expected set)
	return d.removeStale(skillsDir, expected)
}

func (d *SkillsDeployer) deploySkill(skillsDir string, skill model.SkillDef) error {
	dir := filepath.Join(skillsDir, skill.Name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	linkPath := filepath.Join(dir, "SKILL.md")
	target := skill.Source

	// Check existing
	info, err := os.Lstat(linkPath)
	switch {
	case err == nil:
		// Exists — if it's a symlink, update it; if regular file, skip (user-owned)
		if info.Mode()&os.ModeSymlink == 0 {
			return nil // user-owned regular file, don't overwrite
		}
		// Remove old symlink to replace
		if err := os.Remove(linkPath); err != nil {
			return err
		}
	case errors.Is(err, fs.ErrNotExist):
		// doesn't exist, will create
	default:
		return err
	}

	if err := validateSymlinkTarget(target); err != nil {
		return err
	}
	return os.Symlink(target, linkPath)
}

func (d *SkillsDeployer) removeStale(skillsDir string, expected map[string]bool) error {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if expected[name] {
			continue
		}
		// Check if SKILL.md is a symlink (managed by hystak)
		linkPath := filepath.Join(skillsDir, name, "SKILL.md")
		info, err := os.Lstat(linkPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue // no SKILL.md, skip
			}
			return err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue // regular file, user-owned
		}
		// Remove managed symlink and empty directory
		if err := os.Remove(linkPath); err != nil {
			return err
		}
		// Try removing the directory (only succeeds if empty)
		_ = os.Remove(filepath.Join(skillsDir, name))
	}
	return nil
}

// Preflight checks for conflicts — regular files at SKILL.md paths.
func (d *SkillsDeployer) Preflight(projectPath string, config DeployConfig) []PreflightConflict {
	var conflicts []PreflightConflict
	for _, skill := range config.Skills {
		linkPath := filepath.Join(projectPath, ".claude", "skills", skill.Name, "SKILL.md")
		info, err := os.Lstat(linkPath)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink == 0 {
			conflicts = append(conflicts, PreflightConflict{
				Path:    linkPath,
				Kind:    ResourceDeployerSkills,
				Message: fmt.Sprintf("skill %q: regular file exists at %s (not a symlink)", skill.Name, linkPath),
			})
		}
	}
	return conflicts
}

// ReadDeployed reads currently deployed skills by scanning symlinks.
func (d *SkillsDeployer) ReadDeployed(projectPath string) (DeployConfig, error) {
	skillsDir := filepath.Join(projectPath, ".claude", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return DeployConfig{}, nil
		}
		return DeployConfig{}, err
	}

	var skills []model.SkillDef
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		linkPath := filepath.Join(skillsDir, e.Name(), "SKILL.md")
		info, err := os.Lstat(linkPath)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue // not managed
		}
		target, err := os.Readlink(linkPath)
		if err != nil {
			continue
		}
		skills = append(skills, model.SkillDef{
			Name:   e.Name(),
			Source: target,
		})
	}
	return DeployConfig{Skills: skills}, nil
}
