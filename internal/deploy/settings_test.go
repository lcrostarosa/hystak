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
		t.Fatalf("PostToolUse hooks = %d, want 1", len(settings.Hooks["PostToolUse"]))
	}

	// Verify actual hook field values (Item 4: round-trip content check)
	hook := settings.Hooks["PostToolUse"][0]
	if hook.Matcher != "Edit" {
		t.Errorf("hook matcher = %q, want Edit", hook.Matcher)
	}
	if hook.Command != "eslint" {
		t.Errorf("hook command = %q, want eslint", hook.Command)
	}
	if hook.Timeout != 30 {
		t.Errorf("hook timeout = %d, want 30", hook.Timeout)
	}

	if settings.Permissions == nil {
		t.Fatal("permissions is nil")
	}
	if len(settings.Permissions.Allow) != 1 {
		t.Fatalf("allow = %d, want 1", len(settings.Permissions.Allow))
	}
	if len(settings.Permissions.Deny) != 1 {
		t.Fatalf("deny = %d, want 1", len(settings.Permissions.Deny))
	}

	// Verify actual permission rule strings
	if settings.Permissions.Allow[0] != "Bash(*)" {
		t.Errorf("allow[0] = %q, want Bash(*)", settings.Permissions.Allow[0])
	}
	if settings.Permissions.Deny[0] != "Bash(rm -rf /)" {
		t.Errorf("deny[0] = %q, want Bash(rm -rf /)", settings.Permissions.Deny[0])
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

	// Re-deploy with empty config -- file should be removed (CS-11)
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
		t.Fatalf("deployed hooks = %d, want 1", len(deployed.Hooks))
	}
	if len(deployed.Permissions) != 1 {
		t.Fatalf("deployed permissions = %d, want 1", len(deployed.Permissions))
	}

	// Verify hook content (Item 4: round-trip content check)
	h := deployed.Hooks[0]
	if h.Event != model.HookEventPreToolUse {
		t.Errorf("hook event = %q, want PreToolUse", h.Event)
	}
	if h.Matcher != "Bash" {
		t.Errorf("hook matcher = %q, want Bash", h.Matcher)
	}
	if h.Command != "echo blocked" {
		t.Errorf("hook command = %q, want 'echo blocked'", h.Command)
	}
	if h.Timeout != 5 {
		t.Errorf("hook timeout = %d, want 5", h.Timeout)
	}

	// Verify permission content
	p := deployed.Permissions[0]
	if p.Rule != "Bash(*)" {
		t.Errorf("permission rule = %q, want Bash(*)", p.Rule)
	}
	if p.Type != model.PermissionAllow {
		t.Errorf("permission type = %q, want allow", p.Type)
	}
}

func TestSettingsDeployer_Preflight(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, projDir string)
		wantConflict bool
	}{
		{
			name: "existing regular file",
			setup: func(t *testing.T, projDir string) {
				t.Helper()
				dir := filepath.Join(projDir, ".claude")
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "settings.local.json"), []byte(`{}`), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantConflict: true,
		},
		{
			name: "existing symlink",
			setup: func(t *testing.T, projDir string) {
				t.Helper()
				dir := filepath.Join(projDir, ".claude")
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(t.TempDir(), "settings.json")
				if err := os.WriteFile(target, []byte(`{}`), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, filepath.Join(dir, "settings.local.json")); err != nil {
					t.Fatal(err)
				}
			},
			wantConflict: false,
		},
		{
			name:         "no file",
			setup:        func(t *testing.T, projDir string) { t.Helper() },
			wantConflict: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			projDir := filepath.Join(tmp, "project")
			if err := os.MkdirAll(projDir, 0o755); err != nil {
				t.Fatal(err)
			}
			tt.setup(t, projDir)

			d := &SettingsDeployer{}
			cfg := DeployConfig{
				Hooks: []model.HookDef{{Name: "lint", Event: model.HookEventPostToolUse, Command: "eslint"}},
			}
			conflicts := d.Preflight(projDir, cfg)
			gotConflict := len(conflicts) > 0
			if gotConflict != tt.wantConflict {
				t.Errorf("conflict = %v, want %v", gotConflict, tt.wantConflict)
			}
		})
	}
}
