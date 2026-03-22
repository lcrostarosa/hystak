package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lcrostarosa/hystak/internal/model"
)

// Compile-time interface compliance checks.
var (
	_ ResourceDeployer = (*SkillsDeployer)(nil)
	_ ResourceDeployer = (*SettingsDeployer)(nil)
	_ ResourceDeployer = (*ClaudeMDDeployer)(nil)
)

func TestSkillsDeployer_Kind(t *testing.T) {
	d := &SkillsDeployer{}
	if d.Kind() != DeployerKindSkill {
		t.Errorf("Kind() = %q, want %q", d.Kind(), DeployerKindSkill)
	}
}

func TestSettingsDeployer_Kind(t *testing.T) {
	d := &SettingsDeployer{}
	if d.Kind() != DeployerKindSettings {
		t.Errorf("Kind() = %q, want %q", d.Kind(), DeployerKindSettings)
	}
}

func TestClaudeMDDeployer_Kind(t *testing.T) {
	d := &ClaudeMDDeployer{}
	if d.Kind() != DeployerKindClaudeMD {
		t.Errorf("Kind() = %q, want %q", d.Kind(), DeployerKindClaudeMD)
	}
}

func TestSkillsDeployer_ReadDeployed_Empty(t *testing.T) {
	d := &SkillsDeployer{}
	cfg, err := d.ReadDeployed(t.TempDir())
	if err != nil {
		t.Fatalf("ReadDeployed: %v", err)
	}
	if len(cfg.Skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(cfg.Skills))
	}
}

func TestSkillsDeployer_ReadDeployed_Symlink(t *testing.T) {
	d := &SkillsDeployer{}
	projectDir := t.TempDir()

	// Create a source skill file.
	sourceDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "review.md")
	_ = os.WriteFile(sourceFile, []byte("# Code Review"), 0o644)

	// Create a symlinked skill in the project.
	skillDir := filepath.Join(projectDir, ".claude", "skills", "review")
	_ = os.MkdirAll(skillDir, 0o755)
	_ = os.Symlink(sourceFile, filepath.Join(skillDir, "SKILL.md"))

	cfg, err := d.ReadDeployed(projectDir)
	if err != nil {
		t.Fatalf("ReadDeployed: %v", err)
	}
	if len(cfg.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(cfg.Skills))
	}
	if cfg.Skills[0].Name != "review" {
		t.Errorf("Name = %q", cfg.Skills[0].Name)
	}
	if cfg.Skills[0].Source != sourceFile {
		t.Errorf("Source = %q, want %q", cfg.Skills[0].Source, sourceFile)
	}
}

func TestSettingsDeployer_ReadDeployed_Empty(t *testing.T) {
	d := &SettingsDeployer{}
	cfg, err := d.ReadDeployed(t.TempDir())
	if err != nil {
		t.Fatalf("ReadDeployed: %v", err)
	}
	if len(cfg.Hooks) != 0 || len(cfg.Permissions) != 0 {
		t.Error("expected empty config for non-existent settings")
	}
}

func TestSettingsDeployer_ReadDeployed_WithData(t *testing.T) {
	d := &SettingsDeployer{}
	projectDir := t.TempDir()
	claudeDir := filepath.Join(projectDir, ".claude")
	_ = os.MkdirAll(claudeDir, 0o755)

	settings := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []map[string]any{
				{
					"matcher": "Bash",
					"hooks":   []map[string]any{{"type": "command", "command": "echo pre"}},
				},
			},
		},
		"permissions": map[string]any{
			"allow": []string{"Bash(*)"},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	_ = os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), data, 0o644)

	cfg, err := d.ReadDeployed(projectDir)
	if err != nil {
		t.Fatalf("ReadDeployed: %v", err)
	}
	if len(cfg.Hooks) != 1 {
		t.Fatalf("expected 1 hook, got %d", len(cfg.Hooks))
	}
	if cfg.Hooks[0].Event != "PreToolUse" {
		t.Errorf("Event = %q", cfg.Hooks[0].Event)
	}
	if len(cfg.Permissions) != 1 {
		t.Fatalf("expected 1 permission, got %d", len(cfg.Permissions))
	}
	if cfg.Permissions[0].Rule != "Bash(*)" {
		t.Errorf("Rule = %q", cfg.Permissions[0].Rule)
	}
}

func TestClaudeMDDeployer_ReadDeployed_Symlink(t *testing.T) {
	d := &ClaudeMDDeployer{}
	projectDir := t.TempDir()

	sourceFile := filepath.Join(t.TempDir(), "template.md")
	_ = os.WriteFile(sourceFile, []byte("# Project"), 0o644)
	_ = os.Symlink(sourceFile, filepath.Join(projectDir, "CLAUDE.md"))

	cfg, err := d.ReadDeployed(projectDir)
	if err != nil {
		t.Fatalf("ReadDeployed: %v", err)
	}
	if cfg.TemplateSource != sourceFile {
		t.Errorf("TemplateSource = %q, want %q", cfg.TemplateSource, sourceFile)
	}
}

func TestClaudeMDDeployer_ReadDeployed_NotManaged(t *testing.T) {
	d := &ClaudeMDDeployer{}
	projectDir := t.TempDir()

	// Regular file without sentinel — not managed.
	_ = os.WriteFile(filepath.Join(projectDir, "CLAUDE.md"), []byte("user content"), 0o644)

	cfg, err := d.ReadDeployed(projectDir)
	if err != nil {
		t.Fatalf("ReadDeployed: %v", err)
	}
	if cfg.TemplateSource != "" {
		t.Errorf("expected empty TemplateSource for unmanaged file, got %q", cfg.TemplateSource)
	}
}

func TestDeployConfig_SkillsDeployer_SyncViaInterface(t *testing.T) {
	// Verify that Sync() via the interface delegates correctly.
	d := &SkillsDeployer{}
	projectDir := t.TempDir()

	sourceFile := filepath.Join(t.TempDir(), "test.md")
	_ = os.WriteFile(sourceFile, []byte("# Test"), 0o644)

	cfg := DeployConfig{
		Skills: []model.SkillDef{
			{Name: "test-skill", Source: sourceFile},
		},
	}

	if err := d.Sync(projectDir, cfg); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	// Verify symlink was created.
	skillFile := filepath.Join(projectDir, ".claude", "skills", "test-skill", "SKILL.md")
	info, err := os.Lstat(skillFile)
	if err != nil {
		t.Fatalf("skill file not created: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink")
	}
}
