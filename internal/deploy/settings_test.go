package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hystak/hystak/internal/model"
)

func TestSettingsDeployer_Sync(t *testing.T) {
	tmp := t.TempDir()
	d := &SettingsDeployer{}
	projDir := filepath.Join(tmp, "project")

	cfg := DeployConfig{
		Hooks: []model.HookDef{
			{Name: "lint", Event: model.HookEventPostToolUse, Matcher: "Edit", Command: "eslint", Timeout: 30},
		},
		Permissions: []model.PermissionRule{
			{Name: "allow-bash", Rule: "Bash(*)", Type: model.PermissionAllow},
			{Name: "deny-rm", Rule: "Bash(rm -rf /)", Type: model.PermissionDeny},
		},
	}

	if err := d.Sync(projDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Read and verify
	path := filepath.Join(projDir, ".claude", "settings.local.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var settings settingsJSON
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}

	if len(settings.Hooks["PostToolUse"]) != 1 {
		t.Errorf("PostToolUse hooks = %d, want 1", len(settings.Hooks["PostToolUse"]))
	}
	if settings.Permissions == nil {
		t.Fatal("permissions is nil")
	}
	if len(settings.Permissions.Allow) != 1 {
		t.Errorf("allow = %d, want 1", len(settings.Permissions.Allow))
	}
	if len(settings.Permissions.Deny) != 1 {
		t.Errorf("deny = %d, want 1", len(settings.Permissions.Deny))
	}
}

func TestSettingsDeployer_Sync_Cleanup(t *testing.T) {
	tmp := t.TempDir()
	d := &SettingsDeployer{}
	projDir := filepath.Join(tmp, "project")

	// First deploy with content
	cfg := DeployConfig{
		Hooks: []model.HookDef{{Name: "lint", Event: model.HookEventPostToolUse, Command: "eslint", Timeout: 30}},
	}
	if err := d.Sync(projDir, cfg); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(projDir, ".claude", "settings.local.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatal("file should exist after first sync")
	}

	// Re-deploy with empty config — file should be removed (CS-11)
	if err := d.Sync(projDir, DeployConfig{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should be removed when no hooks/permissions")
	}
}

func TestSettingsDeployer_ReadDeployed(t *testing.T) {
	tmp := t.TempDir()
	d := &SettingsDeployer{}
	projDir := filepath.Join(tmp, "project")

	cfg := DeployConfig{
		Hooks: []model.HookDef{
			{Event: model.HookEventPreToolUse, Matcher: "Bash", Command: "echo blocked", Timeout: 5},
		},
		Permissions: []model.PermissionRule{
			{Rule: "Bash(*)", Type: model.PermissionAllow},
		},
	}
	if err := d.Sync(projDir, cfg); err != nil {
		t.Fatal(err)
	}

	deployed, err := d.ReadDeployed(projDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(deployed.Hooks) != 1 {
		t.Errorf("deployed hooks = %d, want 1", len(deployed.Hooks))
	}
	if len(deployed.Permissions) != 1 {
		t.Errorf("deployed permissions = %d, want 1", len(deployed.Permissions))
	}
}
