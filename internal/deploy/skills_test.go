package deploy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hystak/hystak/internal/model"
)

func TestSkillsDeployer_Sync(t *testing.T) {
	tmp := t.TempDir()
	d := &SkillsDeployer{}

	// Create a source skill file
	sourceDir := filepath.Join(tmp, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(sourceDir, "SKILL.md")
	if err := os.WriteFile(sourcePath, []byte("# Review Skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	projDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := DeployConfig{
		Skills: []model.SkillDef{
			{Name: "review", Source: sourcePath},
		},
	}

	if err := d.Sync(projDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Verify symlink was created
	linkPath := filepath.Join(projDir, ".claude", "skills", "review", "SKILL.md")
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("symlink not created: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink, got regular file")
	}

	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatal(err)
	}
	if target != sourcePath {
		t.Errorf("symlink target = %q, want %q", target, sourcePath)
	}
}

func TestSkillsDeployer_Sync_RemovesStale(t *testing.T) {
	tmp := t.TempDir()
	d := &SkillsDeployer{}
	projDir := filepath.Join(tmp, "project")

	// Deploy a skill
	sourcePath := filepath.Join(tmp, "SKILL.md")
	if err := os.WriteFile(sourcePath, []byte("skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := DeployConfig{
		Skills: []model.SkillDef{{Name: "old-skill", Source: sourcePath}},
	}
	if err := d.Sync(projDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Re-deploy without the skill
	cfg.Skills = nil
	if err := d.Sync(projDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Verify symlink was removed
	linkPath := filepath.Join(projDir, ".claude", "skills", "old-skill", "SKILL.md")
	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Error("stale symlink should have been removed")
	}
}

func TestSkillsDeployer_Preflight_RegularFile(t *testing.T) {
	tmp := t.TempDir()
	d := &SkillsDeployer{}
	projDir := filepath.Join(tmp, "project")

	// Create a regular file at the skill path
	skillDir := filepath.Join(projDir, ".claude", "skills", "review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("user owned"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := DeployConfig{
		Skills: []model.SkillDef{{Name: "review", Source: "/source"}},
	}

	conflicts := d.Preflight(projDir, cfg)
	if len(conflicts) != 1 {
		t.Fatalf("conflicts = %d, want 1", len(conflicts))
	}
	if conflicts[0].Kind != ResourceDeployerSkills {
		t.Errorf("conflict kind = %q, want skills", conflicts[0].Kind)
	}
}

func TestSkillsDeployer_ReadDeployed(t *testing.T) {
	tmp := t.TempDir()
	d := &SkillsDeployer{}
	projDir := filepath.Join(tmp, "project")

	// Create a symlink skill
	sourcePath := filepath.Join(tmp, "SKILL.md")
	if err := os.WriteFile(sourcePath, []byte("skill"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := DeployConfig{
		Skills: []model.SkillDef{{Name: "review", Source: sourcePath}},
	}
	if err := d.Sync(projDir, cfg); err != nil {
		t.Fatal(err)
	}

	deployed, err := d.ReadDeployed(projDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(deployed.Skills) != 1 {
		t.Fatalf("deployed skills = %d, want 1", len(deployed.Skills))
	}
	if deployed.Skills[0].Name != "review" {
		t.Errorf("deployed skill name = %q, want review", deployed.Skills[0].Name)
	}
}
