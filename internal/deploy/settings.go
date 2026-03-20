package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lcrostarosa/hystak/internal/model"
)

// SettingsDeployer writes hooks and permissions to .claude/settings.local.json.
type SettingsDeployer struct{}

// hookEntry represents a single hook in the Claude Code hooks format.
type hookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

// hookMatcher represents a matcher group in the hooks format.
type hookMatcher struct {
	Matcher string      `json:"matcher,omitempty"`
	Hooks   []hookEntry `json:"hooks"`
}

// SyncSettings writes hooks and permissions to settings.local.json.
func (d *SettingsDeployer) SyncSettings(projectPath string, hooks []model.HookDef, permissions []model.PermissionRule) error {
	if len(hooks) == 0 && len(permissions) == 0 {
		return nil
	}

	settingsDir := filepath.Join(projectPath, ".claude")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		return fmt.Errorf("creating .claude directory: %w", err)
	}

	settingsPath := filepath.Join(settingsDir, "settings.local.json")

	// Read existing settings to preserve non-managed keys.
	var raw map[string]json.RawMessage
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("reading %s: %w", settingsPath, err)
		}
		raw = make(map[string]json.RawMessage)
	} else {
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("parsing %s: %w", settingsPath, err)
		}
	}

	// Build hooks section grouped by event.
	if len(hooks) > 0 {
		hooksMap := buildHooksMap(hooks)
		hooksJSON, err := json.Marshal(hooksMap)
		if err != nil {
			return fmt.Errorf("marshaling hooks: %w", err)
		}
		raw["hooks"] = hooksJSON
	}

	// Build permissions section.
	if len(permissions) > 0 {
		permsMap := buildPermissionsMap(permissions)
		permsJSON, err := json.Marshal(permsMap)
		if err != nil {
			return fmt.Errorf("marshaling permissions: %w", err)
		}
		raw["permissions"] = permsJSON
	}

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, append(out, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", settingsPath, err)
	}

	return nil
}

// PreflightSettings checks for hook/permission conflicts before deployment.
// Returns conflicts when hooks or permissions keys exist in settings.local.json
// but were not placed by hystak (hystak does not currently track a managed marker
// for settings, so any pre-existing key is treated as a potential conflict).
func (d *SettingsDeployer) PreflightSettings(projectPath string, hooks []model.HookDef, permissions []model.PermissionRule) []PreflightConflict {
	settingsPath := filepath.Join(projectPath, ".claude", "settings.local.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil // no existing file, no conflicts
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	var conflicts []PreflightConflict

	// A conflict exists when the key is already present and hystak intends to write it.
	if _, exists := raw["hooks"]; exists && len(hooks) > 0 {
		conflicts = append(conflicts, PreflightConflict{
			ResourceType: "hook",
			Name:         "hooks",
			ExistingPath: settingsPath,
		})
	}

	if _, exists := raw["permissions"]; exists && len(permissions) > 0 {
		conflicts = append(conflicts, PreflightConflict{
			ResourceType: "permission",
			Name:         "permissions",
			ExistingPath: settingsPath,
		})
	}

	return conflicts
}

// buildHooksMap groups hooks by event and matcher into the Claude Code format.
func buildHooksMap(hooks []model.HookDef) map[string][]hookMatcher {
	// Group by event+matcher.
	type key struct {
		event   string
		matcher string
	}
	groups := make(map[key][]hookEntry)
	var order []key

	for _, h := range hooks {
		k := key{event: h.Event, matcher: h.Matcher}
		if _, exists := groups[k]; !exists {
			order = append(order, k)
		}
		entry := hookEntry{
			Type:    "command",
			Command: h.Command,
			Timeout: h.Timeout,
		}
		groups[k] = append(groups[k], entry)
	}

	result := make(map[string][]hookMatcher)
	for _, k := range order {
		result[k.event] = append(result[k.event], hookMatcher{
			Matcher: k.matcher,
			Hooks:   groups[k],
		})
	}

	return result
}

// buildPermissionsMap splits permissions into allow/deny lists.
func buildPermissionsMap(permissions []model.PermissionRule) map[string][]string {
	result := make(map[string][]string)

	for _, p := range permissions {
		typ := p.EffectiveType()
		result[typ] = append(result[typ], p.Rule)
	}

	return result
}
