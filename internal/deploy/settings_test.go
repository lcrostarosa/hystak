package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lcrostarosa/hystak/internal/model"
)

func TestSettingsDeployer_SyncSettings(t *testing.T) {
	projectDir := t.TempDir()

	deployer := &SettingsDeployer{}

	hooks := []model.HookDef{
		{Name: "lint", Event: "PreToolUse", Matcher: "Bash", Command: "golangci-lint run ./...", Timeout: 30000},
	}
	permissions := []model.PermissionRule{
		{Name: "allow-bash", Rule: "Bash(*)", Type: "allow"},
		{Name: "deny-env", Rule: "Read(.env)", Type: "deny"},
	}

	if err := deployer.SyncSettings(projectDir, hooks, permissions); err != nil {
		t.Fatalf("SyncSettings: %v", err)
	}

	settingsPath := filepath.Join(projectDir, ".claude", "settings.local.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("reading settings: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("parsing settings: %v", err)
	}

	// Verify hooks.
	if _, ok := raw["hooks"]; !ok {
		t.Fatal("expected 'hooks' key in settings")
	}

	var hooksMap map[string][]hookMatcher
	if err := json.Unmarshal(raw["hooks"], &hooksMap); err != nil {
		t.Fatalf("parsing hooks: %v", err)
	}

	matchers, ok := hooksMap["PreToolUse"]
	if !ok || len(matchers) == 0 {
		t.Fatal("expected PreToolUse hooks")
	}
	if matchers[0].Matcher != "Bash" {
		t.Errorf("matcher = %q, want 'Bash'", matchers[0].Matcher)
	}
	if len(matchers[0].Hooks) != 1 {
		t.Fatalf("expected 1 hook entry, got %d", len(matchers[0].Hooks))
	}
	if matchers[0].Hooks[0].Command != "golangci-lint run ./..." {
		t.Errorf("hook command = %q", matchers[0].Hooks[0].Command)
	}
	if matchers[0].Hooks[0].Timeout != 30000 {
		t.Errorf("hook timeout = %d, want 30000", matchers[0].Hooks[0].Timeout)
	}

	// Verify permissions.
	var permsMap map[string][]string
	if err := json.Unmarshal(raw["permissions"], &permsMap); err != nil {
		t.Fatalf("parsing permissions: %v", err)
	}
	if len(permsMap["allow"]) != 1 || permsMap["allow"][0] != "Bash(*)" {
		t.Errorf("allow permissions = %v", permsMap["allow"])
	}
	if len(permsMap["deny"]) != 1 || permsMap["deny"][0] != "Read(.env)" {
		t.Errorf("deny permissions = %v", permsMap["deny"])
	}
}

func TestSettingsDeployer_PreservesExistingKeys(t *testing.T) {
	projectDir := t.TempDir()

	settingsDir := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write existing settings with a custom key.
	existing := `{"customKey": "customValue"}`
	settingsPath := filepath.Join(settingsDir, "settings.local.json")
	if err := os.WriteFile(settingsPath, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	deployer := &SettingsDeployer{}
	hooks := []model.HookDef{
		{Name: "test", Event: "PostToolUse", Command: "echo done"},
	}

	if err := deployer.SyncSettings(projectDir, hooks, nil); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}

	if _, ok := raw["customKey"]; !ok {
		t.Error("customKey was not preserved")
	}
	if _, ok := raw["hooks"]; !ok {
		t.Error("hooks were not written")
	}
}

func TestSettingsDeployer_NoOp(t *testing.T) {
	projectDir := t.TempDir()

	deployer := &SettingsDeployer{}
	if err := deployer.SyncSettings(projectDir, nil, nil); err != nil {
		t.Fatal(err)
	}

	// No file should be created.
	settingsPath := filepath.Join(projectDir, ".claude", "settings.local.json")
	if _, err := os.Stat(settingsPath); !os.IsNotExist(err) {
		t.Error("settings file should not be created when no hooks/permissions")
	}
}
