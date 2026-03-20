package deploy

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lcrostarosa/hystak/internal/model"
)

const legacyManagedSkillsMarker = ".hystak-managed"

// SkillsDeployer syncs skill files to .claude/skills/<name>/SKILL.md using symlinks.
type SkillsDeployer struct{}

// SyncSkills creates symlinks for each skill and removes stale managed symlinks.
// Unmanaged skill directories (containing regular files, not symlinks) are preserved.
func (d *SkillsDeployer) SyncSkills(projectPath string, skills []model.SkillDef) error {
	skillsDir := filepath.Join(projectPath, ".claude", "skills")

	// Migrate from legacy marker-based tracking if present.
	d.migrateLegacyMarker(skillsDir)

	if len(skills) == 0 {
		return d.cleanManagedSymlinks(skillsDir, nil)
	}

	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return fmt.Errorf("creating skills directory: %w", err)
	}

	currentSet := make(map[string]bool, len(skills))
	for _, skill := range skills {
		currentSet[skill.Name] = true

		source := expandHome(skill.Source)
		if _, err := os.Stat(source); err != nil {
			return fmt.Errorf("skill source %q does not exist: %w", source, err)
		}

		skillDir := filepath.Join(skillsDir, skill.Name)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			return fmt.Errorf("creating skill directory %q: %w", skill.Name, err)
		}

		target := filepath.Join(skillDir, "SKILL.md")

		// If a symlink already exists, check if it points to the right source.
		if info, err := os.Lstat(target); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				existing, err := os.Readlink(target)
				if err == nil && existing == source {
					continue // already correct
				}
				// Wrong target — remove and recreate.
				os.Remove(target)
			} else {
				// Regular file — this is a conflict (legacy copy or user-owned).
				// Replace legacy copies (from old marker-based deploys).
				os.Remove(target)
			}
		}

		if err := os.Symlink(source, target); err != nil {
			return fmt.Errorf("creating symlink for skill %q: %w", skill.Name, err)
		}
	}

	return d.cleanManagedSymlinks(skillsDir, currentSet)
}

// cleanManagedSymlinks removes skill directories that contain a symlinked SKILL.md
// but are not in the current set.
func (d *SkillsDeployer) cleanManagedSymlinks(skillsDir string, current map[string]bool) error {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading skills directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if current[name] {
			continue
		}

		skillFile := filepath.Join(skillsDir, name, "SKILL.md")
		if isSymlink(skillFile) {
			os.RemoveAll(filepath.Join(skillsDir, name))
		}
		// Non-symlink directories are unmanaged — leave them.
	}

	return nil
}

// migrateLegacyMarker reads the old .hystak-managed marker file and removes
// file-copied skill directories that it tracked (they'll be re-created as symlinks
// if still in the current skill list). Deletes the marker afterward.
func (d *SkillsDeployer) migrateLegacyMarker(skillsDir string) {
	markerPath := filepath.Join(skillsDir, legacyManagedSkillsMarker)
	names, err := readLines(markerPath)
	if err != nil {
		return // no marker, nothing to migrate
	}

	for _, name := range names {
		skillFile := filepath.Join(skillsDir, name, "SKILL.md")
		// Only remove if it's a regular file (legacy copy), not a symlink.
		info, err := os.Lstat(skillFile)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink == 0 {
			os.RemoveAll(filepath.Join(skillsDir, name))
		}
	}

	os.Remove(markerPath)
}

// PreflightSkills checks for skill conflicts before deployment.
// Returns conflicts for skills that exist as regular files (not symlinks) in the project.
func (d *SkillsDeployer) PreflightSkills(projectPath string, skills []model.SkillDef) []PreflightConflict {
	skillsDir := filepath.Join(projectPath, ".claude", "skills")

	var conflicts []PreflightConflict
	for _, skill := range skills {
		target := filepath.Join(skillsDir, skill.Name, "SKILL.md")
		info, err := os.Lstat(target)
		if err != nil {
			continue // doesn't exist, no conflict
		}
		// Symlinks are managed by hystak — no conflict.
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		// Regular file exists — conflict.
		conflicts = append(conflicts, PreflightConflict{
			ResourceType: "skill",
			Name:         skill.Name,
			ExistingPath: target,
		})
	}
	return conflicts
}

// IsSkillManaged returns true if the skill at the given project path is managed
// by hystak (deployed as a symlink).
func (d *SkillsDeployer) IsSkillManaged(projectPath, skillName string) bool {
	skillFile := filepath.Join(projectPath, ".claude", "skills", skillName, "SKILL.md")
	return isSymlink(skillFile)
}
