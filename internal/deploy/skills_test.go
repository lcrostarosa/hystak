package deploy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lcrostarosa/hystak/internal/model"
)

func TestSkillsDeployer_SyncSkills_CreatesSymlinks(t *testing.T) {
	projectDir := t.TempDir()

	// Create a skill source file.
	sourceDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "test-skill.md")
	if err := os.WriteFile(sourceFile, []byte("# Test Skill\nDo the thing."), 0o644); err != nil {
		t.Fatal(err)
	}

	deployer := &SkillsDeployer{}
	skills := []model.SkillDef{
		{Name: "test-skill", Source: sourceFile},
	}

	if err := deployer.SyncSkills(projectDir, skills); err != nil {
		t.Fatalf("SyncSkills: %v", err)
	}

	// Verify skill file is a symlink.
	skillPath := filepath.Join(projectDir, ".claude", "skills", "test-skill", "SKILL.md")
	info, err := os.Lstat(skillPath)
	if err != nil {
		t.Fatalf("skill file not created: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("SKILL.md should be a symlink, not a regular file")
	}

	// Verify symlink target.
	target, err := os.Readlink(skillPath)
	if err != nil {
		t.Fatalf("reading symlink: %v", err)
	}
	if target != sourceFile {
		t.Errorf("symlink target = %q, want %q", target, sourceFile)
	}

	// Verify content is readable through the symlink.
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("reading through symlink: %v", err)
	}
	if string(content) != "# Test Skill\nDo the thing." {
		t.Errorf("unexpected skill content: %q", string(content))
	}
}

func TestSkillsDeployer_RemovesManagedSymlinks(t *testing.T) {
	projectDir := t.TempDir()

	// Create a skill source file.
	sourceDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "old-skill.md")
	if err := os.WriteFile(sourceFile, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	deployer := &SkillsDeployer{}

	// First sync: add old-skill.
	if err := deployer.SyncSkills(projectDir, []model.SkillDef{
		{Name: "old-skill", Source: sourceFile},
	}); err != nil {
		t.Fatal(err)
	}

	// Verify symlink was created.
	skillFile := filepath.Join(projectDir, ".claude", "skills", "old-skill", "SKILL.md")
	if !isSymlink(skillFile) {
		t.Fatal("expected symlink after first sync")
	}

	// Second sync: remove old-skill (empty list).
	if err := deployer.SyncSkills(projectDir, nil); err != nil {
		t.Fatal(err)
	}

	// Verify old skill directory was removed.
	skillDir := filepath.Join(projectDir, ".claude", "skills", "old-skill")
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Error("expected old-skill directory to be removed")
	}
}

func TestSkillsDeployer_PreservesUnmanaged(t *testing.T) {
	projectDir := t.TempDir()

	// Create an unmanaged skill directory with a regular file.
	unmanagedDir := filepath.Join(projectDir, ".claude", "skills", "user-skill")
	if err := os.MkdirAll(unmanagedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unmanagedDir, "SKILL.md"), []byte("user content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a skill source.
	sourceDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "managed.md")
	if err := os.WriteFile(sourceFile, []byte("managed content"), 0o644); err != nil {
		t.Fatal(err)
	}

	deployer := &SkillsDeployer{}
	if err := deployer.SyncSkills(projectDir, []model.SkillDef{
		{Name: "managed-skill", Source: sourceFile},
	}); err != nil {
		t.Fatal(err)
	}

	// Verify unmanaged skill is preserved.
	content, err := os.ReadFile(filepath.Join(unmanagedDir, "SKILL.md"))
	if err != nil {
		t.Fatal("unmanaged skill was removed")
	}
	if string(content) != "user content" {
		t.Error("unmanaged skill content was modified")
	}
}

func TestSkillsDeployer_UpdatesSymlinkTarget(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	sourceA := filepath.Join(sourceDir, "version-a.md")
	sourceB := filepath.Join(sourceDir, "version-b.md")
	if err := os.WriteFile(sourceA, []byte("version A"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourceB, []byte("version B"), 0o644); err != nil {
		t.Fatal(err)
	}

	deployer := &SkillsDeployer{}

	// Deploy with source A.
	if err := deployer.SyncSkills(projectDir, []model.SkillDef{
		{Name: "my-skill", Source: sourceA},
	}); err != nil {
		t.Fatal(err)
	}

	skillFile := filepath.Join(projectDir, ".claude", "skills", "my-skill", "SKILL.md")
	target, _ := os.Readlink(skillFile)
	if target != sourceA {
		t.Fatalf("initial symlink target = %q, want %q", target, sourceA)
	}

	// Re-deploy with source B.
	if err := deployer.SyncSkills(projectDir, []model.SkillDef{
		{Name: "my-skill", Source: sourceB},
	}); err != nil {
		t.Fatal(err)
	}

	target, _ = os.Readlink(skillFile)
	if target != sourceB {
		t.Errorf("updated symlink target = %q, want %q", target, sourceB)
	}
}

func TestSkillsDeployer_SkipsCorrectSymlinks(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	sourceFile := filepath.Join(sourceDir, "stable.md")
	if err := os.WriteFile(sourceFile, []byte("stable"), 0o644); err != nil {
		t.Fatal(err)
	}

	deployer := &SkillsDeployer{}

	// Deploy once.
	if err := deployer.SyncSkills(projectDir, []model.SkillDef{
		{Name: "stable-skill", Source: sourceFile},
	}); err != nil {
		t.Fatal(err)
	}

	skillFile := filepath.Join(projectDir, ".claude", "skills", "stable-skill", "SKILL.md")
	info1, _ := os.Lstat(skillFile)

	// Deploy again — symlink should not be recreated.
	if err := deployer.SyncSkills(projectDir, []model.SkillDef{
		{Name: "stable-skill", Source: sourceFile},
	}); err != nil {
		t.Fatal(err)
	}

	info2, _ := os.Lstat(skillFile)

	// ModTime should be the same since the symlink wasn't recreated.
	if !info1.ModTime().Equal(info2.ModTime()) {
		t.Error("symlink was unnecessarily recreated")
	}
}

func TestSkillsDeployer_CleanMultipleSymlinks(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	sourceA := filepath.Join(sourceDir, "skill-a.md")
	sourceB := filepath.Join(sourceDir, "skill-b.md")
	if err := os.WriteFile(sourceA, []byte("Skill A"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourceB, []byte("Skill B"), 0o644); err != nil {
		t.Fatal(err)
	}

	deployer := &SkillsDeployer{}

	// Deploy two skills.
	if err := deployer.SyncSkills(projectDir, []model.SkillDef{
		{Name: "skill-a", Source: sourceA},
		{Name: "skill-b", Source: sourceB},
	}); err != nil {
		t.Fatal(err)
	}

	// Verify both created.
	skillsDir := filepath.Join(projectDir, ".claude", "skills")
	for _, name := range []string{"skill-a", "skill-b"} {
		p := filepath.Join(skillsDir, name, "SKILL.md")
		if !isSymlink(p) {
			t.Fatalf("skill %q not deployed as symlink", name)
		}
	}

	// Deploy empty list — should clean both.
	if err := deployer.SyncSkills(projectDir, nil); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"skill-a", "skill-b"} {
		p := filepath.Join(skillsDir, name)
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("skill directory %q should have been removed", name)
		}
	}
}

func TestSkillsDeployer_SourceMissing(t *testing.T) {
	projectDir := t.TempDir()
	deployer := &SkillsDeployer{}

	err := deployer.SyncSkills(projectDir, []model.SkillDef{
		{Name: "missing", Source: "/nonexistent/path/skill.md"},
	})
	if err == nil {
		t.Fatal("expected error for missing source")
	}
}

func TestSkillsDeployer_MigratesLegacyMarker(t *testing.T) {
	projectDir := t.TempDir()

	// Simulate legacy state: file copies + marker.
	skillsDir := filepath.Join(projectDir, ".claude", "skills")
	legacyDir := filepath.Join(skillsDir, "legacy-skill")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "SKILL.md"), []byte("legacy copy"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write the legacy marker.
	markerPath := filepath.Join(skillsDir, legacyManagedSkillsMarker)
	if err := os.WriteFile(markerPath, []byte("legacy-skill\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a source for the replacement symlink.
	sourceDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "legacy-skill.md")
	if err := os.WriteFile(sourceFile, []byte("new version"), 0o644); err != nil {
		t.Fatal(err)
	}

	deployer := &SkillsDeployer{}
	if err := deployer.SyncSkills(projectDir, []model.SkillDef{
		{Name: "legacy-skill", Source: sourceFile},
	}); err != nil {
		t.Fatal(err)
	}

	// Marker should be removed.
	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Error("legacy marker should have been removed")
	}

	// Skill should now be a symlink.
	skillFile := filepath.Join(skillsDir, "legacy-skill", "SKILL.md")
	if !isSymlink(skillFile) {
		t.Error("legacy skill should have been converted to symlink")
	}

	target, _ := os.Readlink(skillFile)
	if target != sourceFile {
		t.Errorf("symlink target = %q, want %q", target, sourceFile)
	}
}

func TestSkillsDeployer_MigratesLegacyMarker_RemovesStale(t *testing.T) {
	projectDir := t.TempDir()

	// Simulate legacy state with a skill that is no longer in the list.
	skillsDir := filepath.Join(projectDir, ".claude", "skills")
	staleDir := filepath.Join(skillsDir, "stale-skill")
	if err := os.MkdirAll(staleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staleDir, "SKILL.md"), []byte("stale copy"), 0o644); err != nil {
		t.Fatal(err)
	}

	markerPath := filepath.Join(skillsDir, legacyManagedSkillsMarker)
	if err := os.WriteFile(markerPath, []byte("stale-skill\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	deployer := &SkillsDeployer{}
	// Sync with empty list — stale legacy copy should be removed.
	if err := deployer.SyncSkills(projectDir, nil); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(staleDir, "SKILL.md")); !os.IsNotExist(err) {
		t.Error("stale legacy skill should have been removed during migration")
	}
	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Error("legacy marker should have been removed")
	}
}

func TestPreflightSkills_ConflictForRegularFile(t *testing.T) {
	projectDir := t.TempDir()

	// Create a regular file skill (unmanaged).
	skillDir := filepath.Join(projectDir, ".claude", "skills", "my-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("user content"), 0o644); err != nil {
		t.Fatal(err)
	}

	deployer := &SkillsDeployer{}
	conflicts := deployer.PreflightSkills(projectDir, []model.SkillDef{
		{Name: "my-skill", Source: "/some/source"},
	})

	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Name != "my-skill" {
		t.Errorf("conflict name = %q, want %q", conflicts[0].Name, "my-skill")
	}
}

func TestPreflightSkills_NoConflictForSymlink(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	sourceFile := filepath.Join(sourceDir, "skill.md")
	if err := os.WriteFile(sourceFile, []byte("managed"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a symlinked skill (managed).
	skillDir := filepath.Join(projectDir, ".claude", "skills", "my-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sourceFile, filepath.Join(skillDir, "SKILL.md")); err != nil {
		t.Fatal(err)
	}

	deployer := &SkillsDeployer{}
	conflicts := deployer.PreflightSkills(projectDir, []model.SkillDef{
		{Name: "my-skill", Source: sourceFile},
	})

	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts for symlinked skill, got %d", len(conflicts))
	}
}

func TestIsSkillManaged(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	sourceFile := filepath.Join(sourceDir, "skill.md")
	if err := os.WriteFile(sourceFile, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	deployer := &SkillsDeployer{}

	// Deploy skill as symlink.
	if err := deployer.SyncSkills(projectDir, []model.SkillDef{
		{Name: "test-skill", Source: sourceFile},
	}); err != nil {
		t.Fatal(err)
	}

	if !deployer.IsSkillManaged(projectDir, "test-skill") {
		t.Error("deployed skill should be managed")
	}

	if deployer.IsSkillManaged(projectDir, "nonexistent") {
		t.Error("nonexistent skill should not be managed")
	}
}
