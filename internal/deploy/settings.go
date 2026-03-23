package deploy

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/hystak/hystak/internal/config"
	"github.com/hystak/hystak/internal/model"
)

// Compile-time interface check.
var _ ResourceDeployer = (*SettingsDeployer)(nil)

// SettingsDeployer deploys hooks and permissions to .claude/settings.local.json (S-044).
type SettingsDeployer struct{}

func (d *SettingsDeployer) Kind() ResourceDeployerKind {
	return ResourceDeployerSettings
}

// settingsJSON is the JSON structure of settings.local.json.
type settingsJSON struct {
	Hooks       map[string][]hookEntryJSON `json:"hooks,omitempty"`
	Permissions *permissionsJSON           `json:"permissions,omitempty"`
}

type hookEntryJSON struct {
	Matcher string `json:"matcher,omitempty"`
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

type permissionsJSON struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

// Sync writes hooks and permissions to settings.local.json.
// Cleans up when all hooks/permissions are removed (CS-11 cleanup symmetry).
func (d *SettingsDeployer) Sync(projectPath string, cfg DeployConfig) error {
	dir := filepath.Join(projectPath, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating .claude directory: %w", err)
	}

	path := filepath.Join(dir, "settings.local.json")

	// If no hooks and no permissions, remove the file (cleanup symmetry)
	if len(cfg.Hooks) == 0 && len(cfg.Permissions) == 0 {
		err := os.Remove(path)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}

	settings := settingsJSON{}

	// Build hooks grouped by event
	if len(cfg.Hooks) > 0 {
		settings.Hooks = make(map[string][]hookEntryJSON)
		for _, h := range cfg.Hooks {
			entry := hookEntryJSON{
				Matcher: h.Matcher,
				Command: h.Command,
				Timeout: h.Timeout,
			}
			settings.Hooks[string(h.Event)] = append(settings.Hooks[string(h.Event)], entry)
		}
	}

	// Build permissions split into allow/deny
	if len(cfg.Permissions) > 0 {
		perms := &permissionsJSON{
			Allow: []string{},
			Deny:  []string{},
		}
		for _, p := range cfg.Permissions {
			switch p.Type {
			case model.PermissionAllow:
				perms.Allow = append(perms.Allow, p.Rule)
			case model.PermissionDeny:
				perms.Deny = append(perms.Deny, p.Rule)
			}
		}
		sort.Strings(perms.Allow)
		sort.Strings(perms.Deny)
		settings.Permissions = perms
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return config.AtomicWrite(path, data, 0o644)
}

// Preflight checks for user-owned settings.local.json (not managed by hystak).
func (d *SettingsDeployer) Preflight(projectPath string, cfg DeployConfig) []PreflightConflict {
	if len(cfg.Hooks) == 0 && len(cfg.Permissions) == 0 {
		return nil
	}
	path := filepath.Join(projectPath, ".claude", "settings.local.json")
	info, err := os.Lstat(path)
	if err != nil {
		return nil // file doesn't exist, no conflict
	}
	// Symlinks are not conflicts (S-048)
	if info.Mode()&os.ModeSymlink != 0 {
		return nil
	}
	// Regular file exists — check if it has hystak-managed content
	// For settings, any existing file could conflict
	return []PreflightConflict{{
		Path:    path,
		Kind:    ResourceDeployerSettings,
		Message: "settings.local.json already exists",
	}}
}

// ReadDeployed reads currently deployed hooks and permissions.
func (d *SettingsDeployer) ReadDeployed(projectPath string) (DeployConfig, error) {
	path := filepath.Join(projectPath, ".claude", "settings.local.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return DeployConfig{}, nil
		}
		return DeployConfig{}, err
	}

	var settings settingsJSON
	if err := json.Unmarshal(data, &settings); err != nil {
		return DeployConfig{}, err
	}

	var hooks []model.HookDef
	for event, entries := range settings.Hooks {
		for _, e := range entries {
			hooks = append(hooks, model.HookDef{
				Event:   model.HookEvent(event),
				Matcher: e.Matcher,
				Command: e.Command,
				Timeout: e.Timeout,
			})
		}
	}

	var perms []model.PermissionRule
	if settings.Permissions != nil {
		for _, rule := range settings.Permissions.Allow {
			perms = append(perms, model.PermissionRule{Rule: rule, Type: model.PermissionAllow})
		}
		for _, rule := range settings.Permissions.Deny {
			perms = append(perms, model.PermissionRule{Rule: rule, Type: model.PermissionDeny})
		}
	}

	return DeployConfig{Hooks: hooks, Permissions: perms}, nil
}
