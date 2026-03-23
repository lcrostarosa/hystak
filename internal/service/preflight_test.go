package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hystak/hystak/internal/config"
	"github.com/hystak/hystak/internal/deploy"
	"github.com/hystak/hystak/internal/model"
	"github.com/hystak/hystak/internal/profile"
	"github.com/hystak/hystak/internal/project"
	"github.com/hystak/hystak/internal/registry"
)

func TestService_PreflightCheck_NoConflicts(t *testing.T) {
	svc, _ := setupTestService(t)
	svc.WithResourceDeployers(
		&deploy.SkillsDeployer{},
		&deploy.SettingsDeployer{},
		&deploy.ClaudeMDDeployer{},
	)

	conflicts, err := svc.PreflightCheck("myproject")
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 0 {
		t.Errorf("conflicts = %d, want 0", len(conflicts))
	}
}

func TestService_PreflightCheck_SkillConflict(t *testing.T) {
	tmp := t.TempDir()
	config.OverrideDir(tmp)
	t.Cleanup(func() { config.OverrideDir("") })

	projDir := filepath.Join(tmp, "proj")
	mkdirAll(t, projDir)

	reg := registry.New()
	if err := reg.Skills.Add(model.SkillDef{Name: "review", Source: "/skills/review.md"}); err != nil {
		t.Fatal(err)
	}

	projStore := project.NewStore()
	if err := projStore.Add(model.Project{Name: "proj", Path: projDir, ActiveProfile: "dev"}); err != nil {
		t.Fatal(err)
	}

	profDir := filepath.Join(tmp, "profiles")
	mkdirAll(t, profDir)
	profMgr := profile.NewManager(profDir)
	if err := profMgr.Save(model.ProjectProfile{
		Name:   "dev",
		Skills: []string{"review"},
	}); err != nil {
		t.Fatal(err)
	}

	// Create a regular file at the skill path (conflict)
	skillDir := filepath.Join(projDir, ".claude", "skills", "review")
	mkdirAll(t, skillDir)
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("user file"), 0o644); err != nil {
		t.Fatal(err)
	}

	dep := &deploy.ClaudeCodeDeployer{}
	svc := New(reg, projStore, profMgr, dep)
	svc.WithResourceDeployers(&deploy.SkillsDeployer{})

	conflicts, err := svc.PreflightCheck("proj")
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("conflicts = %d, want 1", len(conflicts))
	}
	if conflicts[0].Kind != deploy.ResourceDeployerSkills {
		t.Errorf("kind = %q, want skills", conflicts[0].Kind)
	}
}

func TestService_PreflightCheck_NoProfile(t *testing.T) {
	tmp := t.TempDir()
	config.OverrideDir(tmp)
	t.Cleanup(func() { config.OverrideDir("") })

	reg := registry.New()
	projStore := project.NewStore()
	if err := projStore.Add(model.Project{Name: "proj", Path: filepath.Join(tmp, "proj")}); err != nil {
		t.Fatal(err)
	}

	profDir := filepath.Join(tmp, "profiles")
	mkdirAll(t, profDir)
	profMgr := profile.NewManager(profDir)
	dep := &deploy.ClaudeCodeDeployer{}
	svc := New(reg, projStore, profMgr, dep)

	conflicts, err := svc.PreflightCheck("proj")
	if err != nil {
		t.Fatal(err)
	}
	if conflicts != nil {
		t.Errorf("conflicts should be nil for no-profile project, got %d", len(conflicts))
	}
}

func TestService_SyncWithConflicts_Pending_Errors(t *testing.T) {
	svc, _ := setupTestService(t)

	conflicts := []SyncConflict{
		{Resolution: ConflictPending},
	}
	_, err := svc.SyncWithConflicts("myproject", conflicts)
	if err == nil {
		t.Fatal("expected error for pending conflict")
	}
}
