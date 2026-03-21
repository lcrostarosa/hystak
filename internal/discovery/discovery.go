// Package discovery scans filesystem locations to find available Claude Code
// configuration items (MCPs, skills, hooks, permissions, env vars) from global,
// project, and registry scopes.
package discovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/registry"
)

// Source identifies where a discovered item originates from.
type Source int

const (
	SourceGlobal   Source = iota // ~/.claude/
	SourceProject                // project directory
	SourceRegistry               // hystak registry
)

// String returns the human-readable source name.
func (s Source) String() string {
	switch s {
	case SourceGlobal:
		return "global"
	case SourceProject:
		return "project"
	case SourceRegistry:
		return "registry"
	default:
		return "unknown"
	}
}

// DiscoveredMCP is an MCP server found during scanning.
type DiscoveredMCP struct {
	Name      string
	ServerDef model.ServerDef
	Source    Source
	IsManaged bool // true if hystak placed it
}

// DiscoveredSkill is a skill found during scanning.
type DiscoveredSkill struct {
	Name        string
	Path        string // absolute path to the SKILL.md
	Description string // parsed from first line if available
	Source      Source
	IsManaged   bool // true if path is a symlink
}

// DiscoveredHook is a hook found during scanning.
type DiscoveredHook struct {
	Name    string
	Event   string
	Matcher string
	Command string
	Timeout int
	Source  Source
}

// DiscoveredPermission is a permission rule found during scanning.
type DiscoveredPermission struct {
	Name   string
	Rule   string
	Type   string // "allow" or "deny"
	Source Source
}

// DiscoveredEnvVar is an environment variable found during scanning.
type DiscoveredEnvVar struct {
	Key    string
	Value  string
	Source Source
}

// DiscoveredPrompt is a prompt fragment found during scanning.
type DiscoveredPrompt struct {
	Name        string
	Description string
	Category    string
	Source      Source
}

// Items holds all items found by a scan.
type Items struct {
	MCPs        []DiscoveredMCP
	Skills      []DiscoveredSkill
	Hooks       []DiscoveredHook
	Permissions []DiscoveredPermission
	EnvVars     []DiscoveredEnvVar
	Prompts     []DiscoveredPrompt
}

// Engine scans filesystem locations for Claude Code configuration items.
type Engine struct {
	// claudeHome is the path to the Claude config directory (e.g. ~/.claude).
	claudeHome string
	// registry is the loaded hystak registry (nil to skip registry scan).
	registry *registry.Registry
}

// NewEngine creates a discovery engine.
// claudeHome is typically ~/.claude. registry may be nil to skip registry scanning.
func NewEngine(claudeHome string, reg *registry.Registry) *Engine {
	return &Engine{
		claudeHome: claudeHome,
		registry:   reg,
	}
}

// Scan discovers all available configuration items from global, project, and registry scopes.
// Errors from individual sources are silently skipped (graceful degradation).
func (e *Engine) Scan(projectPath string) *Items {
	items := &Items{}
	items.MCPs = e.ScanMCPs(projectPath)
	items.Skills = e.ScanSkills(projectPath)
	items.Hooks = e.ScanHooks(projectPath)
	items.Permissions = e.ScanPermissions(projectPath)
	items.EnvVars = e.ScanEnvVars(projectPath)
	items.Prompts = e.ScanPrompts()
	return items
}

// ScanMCPs discovers MCP servers from global claude.json, project .mcp.json, and the registry.
func (e *Engine) ScanMCPs(projectPath string) []DiscoveredMCP {
	var results []DiscoveredMCP

	// Global: ~/.claude.json → mcpServers
	globalPath := filepath.Join(filepath.Dir(e.claudeHome), ".claude.json")
	if mcps := readMCPsFromJSON(globalPath); mcps != nil {
		for name, srv := range mcps {
			results = append(results, DiscoveredMCP{
				Name:      name,
				ServerDef: srv,
				Source:    SourceGlobal,
			})
		}
	}

	// Project: <project>/.mcp.json → mcpServers
	if projectPath != "" {
		projPath := filepath.Join(projectPath, ".mcp.json")
		if mcps := readMCPsFromJSON(projPath); mcps != nil {
			for name, srv := range mcps {
				results = append(results, DiscoveredMCP{
					Name:      name,
					ServerDef: srv,
					Source:    SourceProject,
				})
			}
		}
	}

	// Registry: all servers from hystak registry
	if e.registry != nil {
		for _, srv := range e.registry.List() {
			results = append(results, DiscoveredMCP{
				Name:      srv.Name,
				ServerDef: srv,
				Source:    SourceRegistry,
				IsManaged: true,
			})
		}
	}

	return results
}

// ScanSkills discovers skills from global and project .claude/skills/ directories,
// plus all skills registered in the hystak registry.
func (e *Engine) ScanSkills(projectPath string) []DiscoveredSkill {
	var results []DiscoveredSkill

	// Global: ~/.claude/skills/*/SKILL.md
	globalSkillsDir := filepath.Join(e.claudeHome, "skills")
	results = append(results, scanSkillsDir(globalSkillsDir, SourceGlobal)...)

	// Project: <project>/.claude/skills/*/SKILL.md
	if projectPath != "" {
		projSkillsDir := filepath.Join(projectPath, ".claude", "skills")
		results = append(results, scanSkillsDir(projSkillsDir, SourceProject)...)
	}

	// Registry: all skills from hystak registry
	if e.registry != nil {
		for _, skill := range e.registry.ListSkills() {
			results = append(results, DiscoveredSkill{
				Name:        skill.Name,
				Path:        skill.Source,
				Description: skill.Description,
				Source:      SourceRegistry,
				IsManaged:   true,
			})
		}
	}

	return results
}

// ScanHooks discovers hooks from global and project settings files,
// plus all hooks registered in the hystak registry.
func (e *Engine) ScanHooks(projectPath string) []DiscoveredHook {
	var results []DiscoveredHook

	// Global: ~/.claude/settings.json → hooks
	globalSettings := filepath.Join(e.claudeHome, "settings.json")
	results = append(results, readHooksFromSettings(globalSettings, SourceGlobal)...)

	// Project: <project>/.claude/settings.local.json → hooks
	if projectPath != "" {
		projSettings := filepath.Join(projectPath, ".claude", "settings.local.json")
		results = append(results, readHooksFromSettings(projSettings, SourceProject)...)
	}

	// Registry: all hooks from hystak registry
	if e.registry != nil {
		for _, hook := range e.registry.ListHooks() {
			results = append(results, DiscoveredHook{
				Name:    hook.Name,
				Event:   hook.Event,
				Matcher: hook.Matcher,
				Command: hook.Command,
				Timeout: hook.Timeout,
				Source:  SourceRegistry,
			})
		}
	}

	return results
}

// ScanPermissions discovers permission rules from global and project settings files,
// plus all permissions registered in the hystak registry.
func (e *Engine) ScanPermissions(projectPath string) []DiscoveredPermission {
	var results []DiscoveredPermission

	// Global: ~/.claude/settings.json → permissions
	globalSettings := filepath.Join(e.claudeHome, "settings.json")
	results = append(results, readPermissionsFromSettings(globalSettings, SourceGlobal)...)

	// Project: <project>/.claude/settings.local.json → permissions
	if projectPath != "" {
		projSettings := filepath.Join(projectPath, ".claude", "settings.local.json")
		results = append(results, readPermissionsFromSettings(projSettings, SourceProject)...)
	}

	// Registry: all permissions from hystak registry
	if e.registry != nil {
		for _, perm := range e.registry.ListPermissions() {
			results = append(results, DiscoveredPermission{
				Name:   perm.Name,
				Rule:   perm.Rule,
				Type:   string(perm.Type),
				Source: SourceRegistry,
			})
		}
	}

	return results
}

// ScanEnvVars discovers environment variables from global and project settings files.
func (e *Engine) ScanEnvVars(projectPath string) []DiscoveredEnvVar {
	var results []DiscoveredEnvVar

	// Global: ~/.claude/settings.json → env
	globalSettings := filepath.Join(e.claudeHome, "settings.json")
	results = append(results, readEnvVarsFromSettings(globalSettings, SourceGlobal)...)

	// Project: <project>/.claude/settings.local.json → env
	if projectPath != "" {
		projSettings := filepath.Join(projectPath, ".claude", "settings.local.json")
		results = append(results, readEnvVarsFromSettings(projSettings, SourceProject)...)
	}

	return results
}

// ScanPrompts discovers prompt fragments from the hystak registry.
func (e *Engine) ScanPrompts() []DiscoveredPrompt {
	var results []DiscoveredPrompt
	if e.registry != nil {
		for _, prompt := range e.registry.ListPrompts() {
			results = append(results, DiscoveredPrompt{
				Name:        prompt.Name,
				Description: prompt.Description,
				Category:    prompt.Category,
				Source:      SourceRegistry,
			})
		}
	}
	return results
}

// --- MCP helpers ---

// readMCPsFromJSON reads mcpServers from a Claude Code JSON config file.
func readMCPsFromJSON(path string) map[string]model.ServerDef {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	serversRaw, ok := raw["mcpServers"]
	if !ok {
		return nil
	}

	// Parse into the Claude Code server format.
	var ccServers map[string]struct {
		Type    string            `json:"type"`
		Command string            `json:"command,omitempty"`
		Args    []string          `json:"args,omitempty"`
		Env     map[string]string `json:"env,omitempty"`
		URL     string            `json:"url,omitempty"`
		Headers map[string]string `json:"headers,omitempty"`
	}
	if err := json.Unmarshal(serversRaw, &ccServers); err != nil {
		return nil
	}

	result := make(map[string]model.ServerDef, len(ccServers))
	for name, ccs := range ccServers {
		result[name] = model.ServerDef{
			Name:      name,
			Transport: model.Transport(ccs.Type),
			Command:   ccs.Command,
			Args:      ccs.Args,
			Env:       ccs.Env,
			URL:       ccs.URL,
			Headers:   ccs.Headers,
		}
	}
	return result
}

// --- Skills helpers ---

// scanSkillsDir scans a directory for skill subdirectories containing SKILL.md.
func scanSkillsDir(dir string, source Source) []DiscoveredSkill {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var results []DiscoveredSkill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(dir, entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillPath); err != nil {
			continue
		}

		skill := DiscoveredSkill{
			Name:   entry.Name(),
			Path:   skillPath,
			Source: source,
		}

		// Check if it's a symlink (managed by hystak).
		if linfo, err := os.Lstat(skillPath); err == nil {
			skill.IsManaged = linfo.Mode()&os.ModeSymlink != 0
		}

		// Parse first non-empty, non-frontmatter line as description.
		skill.Description = parseSkillDescription(skillPath)

		results = append(results, skill)
	}
	return results
}

// parseSkillDescription extracts a brief description from the SKILL.md file.
// It reads past YAML frontmatter (---) and returns the first non-empty content line.
func parseSkillDescription(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	inFrontmatter := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			inFrontmatter = !inFrontmatter
			continue
		}
		if inFrontmatter {
			continue
		}
		if trimmed == "" {
			continue
		}
		// Strip markdown heading prefix.
		trimmed = strings.TrimLeft(trimmed, "# ")
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// --- Settings (hooks, permissions, env) helpers ---

// settingsFile represents the structure of a Claude Code settings JSON file.
type settingsFile struct {
	Hooks       map[string]json.RawMessage `json:"hooks,omitempty"`
	Permissions *permissionsBlock          `json:"permissions,omitempty"`
	Env         map[string]string          `json:"env,omitempty"`
}

type permissionsBlock struct {
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

// hookMatcherJSON matches the Claude Code hooks format.
type hookMatcherJSON struct {
	Matcher string          `json:"matcher,omitempty"`
	Hooks   []hookEntryJSON `json:"hooks"`
}

type hookEntryJSON struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

// readSettingsFile reads and parses a Claude Code settings JSON file.
func readSettingsFile(path string) *settingsFile {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var sf settingsFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil
	}
	return &sf
}

// readHooksFromSettings extracts hooks from a settings file.
func readHooksFromSettings(path string, source Source) []DiscoveredHook {
	sf := readSettingsFile(path)
	if sf == nil || len(sf.Hooks) == 0 {
		return nil
	}

	var results []DiscoveredHook
	for event, raw := range sf.Hooks {
		var matchers []hookMatcherJSON
		if err := json.Unmarshal(raw, &matchers); err != nil {
			continue
		}
		for _, m := range matchers {
			for i, h := range m.Hooks {
				name := buildHookName(event, m.Matcher, i)
				results = append(results, DiscoveredHook{
					Name:    name,
					Event:   event,
					Matcher: m.Matcher,
					Command: h.Command,
					Timeout: h.Timeout,
					Source:  source,
				})
			}
		}
	}
	return results
}

// buildHookName creates a synthetic name for a discovered hook.
func buildHookName(event, matcher string, index int) string {
	name := event
	if matcher != "" {
		name += ":" + matcher
	}
	if index > 0 {
		name += ":" + strings.Repeat("i", index) // simple suffix for duplicates
	}
	return name
}

// readPermissionsFromSettings extracts permission rules from a settings file.
func readPermissionsFromSettings(path string, source Source) []DiscoveredPermission {
	sf := readSettingsFile(path)
	if sf == nil || sf.Permissions == nil {
		return nil
	}

	var results []DiscoveredPermission
	for _, rule := range sf.Permissions.Allow {
		results = append(results, DiscoveredPermission{
			Name:   rule,
			Rule:   rule,
			Type:   "allow",
			Source: source,
		})
	}
	for _, rule := range sf.Permissions.Deny {
		results = append(results, DiscoveredPermission{
			Name:   rule,
			Rule:   rule,
			Type:   "deny",
			Source: source,
		})
	}
	return results
}

// readEnvVarsFromSettings extracts environment variables from a settings file.
func readEnvVarsFromSettings(path string, source Source) []DiscoveredEnvVar {
	sf := readSettingsFile(path)
	if sf == nil || len(sf.Env) == 0 {
		return nil
	}

	var results []DiscoveredEnvVar
	for key, val := range sf.Env {
		results = append(results, DiscoveredEnvVar{
			Key:    key,
			Value:  val,
			Source: source,
		})
	}
	return results
}
