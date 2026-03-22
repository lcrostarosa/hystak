package discovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/registry"
)

// helper creates a temp dir tree and returns the base dir + cleanup func.
func setup(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// writeFile creates a file with the given content inside base.
func writeFile(t *testing.T, base string, relPath string, content string) {
	t.Helper()
	full := filepath.Join(base, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeJSON creates a JSON file inside base.
func writeJSON(t *testing.T, base string, relPath string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, base, relPath, string(data))
}

func TestScanMCPs_Global(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")
	_ = os.MkdirAll(claudeHome, 0o755)

	// Write ~/.claude.json with mcpServers
	globalConfig := map[string]any{
		"mcpServers": map[string]any{
			"github": map[string]any{
				"type":    "stdio",
				"command": "gh-mcp",
				"args":    []string{"--token", "abc"},
			},
			"browser": map[string]any{
				"type": "sse",
				"url":  "http://localhost:3000",
			},
		},
	}
	writeJSON(t, base, ".claude.json", globalConfig)

	engine := NewEngine(claudeHome, nil)
	mcps := engine.ScanMCPs("")

	if len(mcps) != 2 {
		t.Fatalf("expected 2 global MCPs, got %d", len(mcps))
	}

	found := map[string]DiscoveredMCP{}
	for _, m := range mcps {
		found[m.Name] = m
	}

	gh, ok := found["github"]
	if !ok {
		t.Fatal("expected github MCP")
	}
	if gh.Source != SourceGlobal {
		t.Errorf("expected source Global, got %v", gh.Source)
	}
	if gh.ServerDef.Command != "gh-mcp" {
		t.Errorf("expected command gh-mcp, got %q", gh.ServerDef.Command)
	}
	if gh.ServerDef.Transport != model.TransportStdio {
		t.Errorf("expected transport stdio, got %q", gh.ServerDef.Transport)
	}

	browser := found["browser"]
	if browser.ServerDef.URL != "http://localhost:3000" {
		t.Errorf("expected URL http://localhost:3000, got %q", browser.ServerDef.URL)
	}
}

func TestScanMCPs_Project(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")
	_ = os.MkdirAll(claudeHome, 0o755)

	projectPath := filepath.Join(base, "myproject")
	_ = os.MkdirAll(projectPath, 0o755)

	projConfig := map[string]any{
		"mcpServers": map[string]any{
			"local-mcp": map[string]any{
				"type":    "stdio",
				"command": "local-mcp-server",
			},
		},
	}
	writeJSON(t, projectPath, ".mcp.json", projConfig)

	engine := NewEngine(claudeHome, nil)
	mcps := engine.ScanMCPs(projectPath)

	if len(mcps) != 1 {
		t.Fatalf("expected 1 project MCP, got %d", len(mcps))
	}
	if mcps[0].Source != SourceProject {
		t.Errorf("expected source Project, got %v", mcps[0].Source)
	}
	if mcps[0].Name != "local-mcp" {
		t.Errorf("expected name local-mcp, got %q", mcps[0].Name)
	}
}

func newRegistry() *registry.Registry {
	return &registry.Registry{
		Servers:     make(map[string]model.ServerDef),
		Skills:      make(map[string]model.SkillDef),
		Hooks:       make(map[string]model.HookDef),
		Permissions: make(map[string]model.PermissionRule),
		Templates:   make(map[string]model.TemplateDef),
		Tags:        make(map[string][]string),
	}
}

func TestScanMCPs_Registry(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")
	_ = os.MkdirAll(claudeHome, 0o755)

	reg := newRegistry()
	_ = reg.Add(model.ServerDef{
		Name:      "registry-mcp",
		Transport: model.TransportStdio,
		Command:   "reg-cmd",
	})

	engine := NewEngine(claudeHome, reg)
	mcps := engine.ScanMCPs("")

	var regMCP *DiscoveredMCP
	for _, m := range mcps {
		if m.Source == SourceRegistry {
			regMCP = &m
			break
		}
	}
	if regMCP == nil {
		t.Fatal("expected registry MCP")
	}
	if regMCP.Name != "registry-mcp" {
		t.Errorf("expected name registry-mcp, got %q", regMCP.Name)
	}
	if !regMCP.IsManaged {
		t.Error("expected registry MCPs to be marked as managed")
	}
}

func TestScanMCPs_Deduplication(t *testing.T) {
	// Same MCP in global + project → both shown with correct source
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")
	_ = os.MkdirAll(claudeHome, 0o755)

	projectPath := filepath.Join(base, "myproject")
	mcpConfig := map[string]any{
		"mcpServers": map[string]any{
			"github": map[string]any{
				"type":    "stdio",
				"command": "gh-mcp",
			},
		},
	}
	writeJSON(t, base, ".claude.json", mcpConfig)
	writeJSON(t, projectPath, ".mcp.json", mcpConfig)

	engine := NewEngine(claudeHome, nil)
	mcps := engine.ScanMCPs(projectPath)

	if len(mcps) != 2 {
		t.Fatalf("expected 2 MCPs (global + project), got %d", len(mcps))
	}

	sources := map[Source]bool{}
	for _, m := range mcps {
		sources[m.Source] = true
	}
	if !sources[SourceGlobal] || !sources[SourceProject] {
		t.Error("expected both global and project sources")
	}
}

func TestScanMCPs_MissingDir(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")
	// Don't create any files

	engine := NewEngine(claudeHome, nil)
	mcps := engine.ScanMCPs("")

	if len(mcps) != 0 {
		t.Fatalf("expected 0 MCPs for missing dir, got %d", len(mcps))
	}
}

func TestScanMCPs_MalformedJSON(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")
	_ = os.MkdirAll(claudeHome, 0o755)

	// Write malformed JSON
	writeFile(t, base, ".claude.json", "{invalid json")

	engine := NewEngine(claudeHome, nil)
	mcps := engine.ScanMCPs("")

	// Should gracefully return empty, not error
	if len(mcps) != 0 {
		t.Fatalf("expected 0 MCPs for malformed JSON, got %d", len(mcps))
	}
}

func TestScanSkills_Global(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")

	writeFile(t, base, ".claude/skills/code-review/SKILL.md", "# Code Review\nReviews code for quality.")
	writeFile(t, base, ".claude/skills/testing/SKILL.md", "---\nname: testing\n---\n# Testing\nRun tests.")

	engine := NewEngine(claudeHome, nil)
	skills := engine.ScanSkills("")

	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}

	found := map[string]DiscoveredSkill{}
	for _, s := range skills {
		found[s.Name] = s
	}

	cr, ok := found["code-review"]
	if !ok {
		t.Fatal("expected code-review skill")
	}
	if cr.Source != SourceGlobal {
		t.Errorf("expected source Global, got %v", cr.Source)
	}
	if cr.Description != "Code Review" {
		t.Errorf("expected description 'Code Review', got %q", cr.Description)
	}
	if cr.IsManaged {
		t.Error("regular file should not be marked as managed")
	}

	ts := found["testing"]
	if ts.Description != "Testing" {
		t.Errorf("expected description 'Testing' (past frontmatter), got %q", ts.Description)
	}
}

func TestScanSkills_Project(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")
	_ = os.MkdirAll(claudeHome, 0o755)

	projectPath := filepath.Join(base, "myproject")
	writeFile(t, projectPath, ".claude/skills/local-skill/SKILL.md", "# Local Skill")

	engine := NewEngine(claudeHome, nil)
	skills := engine.ScanSkills(projectPath)

	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Source != SourceProject {
		t.Errorf("expected source Project, got %v", skills[0].Source)
	}
}

func TestScanSkills_SymlinkDetected(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")

	// Create a real skill as the source.
	sourceDir := filepath.Join(base, "hystak-skills", "managed-skill")
	_ = os.MkdirAll(sourceDir, 0o755)
	_ = os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte("# Managed"), 0o644)

	// Create a symlink in the project skills directory.
	projectPath := filepath.Join(base, "myproject")
	skillDir := filepath.Join(projectPath, ".claude", "skills", "managed-skill")
	_ = os.MkdirAll(skillDir, 0o755)
	_ = os.Symlink(filepath.Join(sourceDir, "SKILL.md"), filepath.Join(skillDir, "SKILL.md"))

	engine := NewEngine(claudeHome, nil)
	skills := engine.ScanSkills(projectPath)

	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if !skills[0].IsManaged {
		t.Error("expected symlinked skill to be marked as managed")
	}
}

func TestScanSkills_MissingDir(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")
	// Don't create skills dir

	engine := NewEngine(claudeHome, nil)
	skills := engine.ScanSkills("")

	if len(skills) != 0 {
		t.Fatalf("expected 0 skills, got %d", len(skills))
	}
}

func TestScanSkills_NoSKILLFile(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")

	// Create skill dir without SKILL.md
	_ = os.MkdirAll(filepath.Join(claudeHome, "skills", "empty-skill"), 0o755)

	engine := NewEngine(claudeHome, nil)
	skills := engine.ScanSkills("")

	if len(skills) != 0 {
		t.Fatalf("expected 0 skills (no SKILL.md), got %d", len(skills))
	}
}

func TestScanHooks_Global(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")

	settings := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []map[string]any{
				{
					"matcher": "Bash",
					"hooks": []map[string]any{
						{
							"type":    "command",
							"command": "echo 'pre-bash'",
							"timeout": 5000,
						},
					},
				},
			},
			"PostToolUse": []map[string]any{
				{
					"hooks": []map[string]any{
						{
							"type":    "command",
							"command": "echo 'post-all'",
						},
					},
				},
			},
		},
	}
	writeJSON(t, claudeHome, "settings.json", settings)

	engine := NewEngine(claudeHome, nil)
	hooks := engine.ScanHooks("")

	if len(hooks) != 2 {
		t.Fatalf("expected 2 hooks, got %d", len(hooks))
	}

	found := map[string]DiscoveredHook{}
	for _, h := range hooks {
		found[h.Event+":"+h.Matcher] = h
	}

	preBash, ok := found["PreToolUse:Bash"]
	if !ok {
		t.Fatal("expected PreToolUse:Bash hook")
	}
	if preBash.Command != "echo 'pre-bash'" {
		t.Errorf("expected command echo 'pre-bash', got %q", preBash.Command)
	}
	if preBash.Timeout != 5000 {
		t.Errorf("expected timeout 5000, got %d", preBash.Timeout)
	}
	if preBash.Source != SourceGlobal {
		t.Errorf("expected source Global, got %v", preBash.Source)
	}

	postAll, ok := found["PostToolUse:"]
	if !ok {
		t.Fatal("expected PostToolUse (no matcher) hook")
	}
	if postAll.Command != "echo 'post-all'" {
		t.Errorf("expected command echo 'post-all', got %q", postAll.Command)
	}
}

func TestScanHooks_Project(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")
	_ = os.MkdirAll(claudeHome, 0o755)

	projectPath := filepath.Join(base, "myproject")
	settings := map[string]any{
		"hooks": map[string]any{
			"UserPromptSubmit": []map[string]any{
				{
					"hooks": []map[string]any{
						{
							"type":    "command",
							"command": "echo 'submit'",
						},
					},
				},
			},
		},
	}
	writeJSON(t, projectPath, ".claude/settings.local.json", settings)

	engine := NewEngine(claudeHome, nil)
	hooks := engine.ScanHooks(projectPath)

	if len(hooks) != 1 {
		t.Fatalf("expected 1 hook, got %d", len(hooks))
	}
	if hooks[0].Source != SourceProject {
		t.Errorf("expected source Project, got %v", hooks[0].Source)
	}
}

func TestScanPermissions_Global(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")

	settings := map[string]any{
		"permissions": map[string]any{
			"allow": []string{"Bash(*)", "WebFetch(domain:github.com)"},
			"deny":  []string{"DeleteFile(*)"},
		},
	}
	writeJSON(t, claudeHome, "settings.json", settings)

	engine := NewEngine(claudeHome, nil)
	perms := engine.ScanPermissions("")

	if len(perms) != 3 {
		t.Fatalf("expected 3 permissions, got %d", len(perms))
	}

	allowCount, denyCount := 0, 0
	for _, p := range perms {
		switch p.Type {
		case "allow":
			allowCount++
		case "deny":
			denyCount++
		}
		if p.Source != SourceGlobal {
			t.Errorf("expected source Global, got %v", p.Source)
		}
	}
	if allowCount != 2 {
		t.Errorf("expected 2 allow rules, got %d", allowCount)
	}
	if denyCount != 1 {
		t.Errorf("expected 1 deny rule, got %d", denyCount)
	}
}

func TestScanPermissions_Project(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")
	_ = os.MkdirAll(claudeHome, 0o755)

	projectPath := filepath.Join(base, "myproject")
	settings := map[string]any{
		"permissions": map[string]any{
			"allow": []string{"Read(*)"},
		},
	}
	writeJSON(t, projectPath, ".claude/settings.local.json", settings)

	engine := NewEngine(claudeHome, nil)
	perms := engine.ScanPermissions(projectPath)

	if len(perms) != 1 {
		t.Fatalf("expected 1 permission, got %d", len(perms))
	}
	if perms[0].Source != SourceProject {
		t.Errorf("expected source Project, got %v", perms[0].Source)
	}
}

func TestScanEnvVars_Global(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")

	settings := map[string]any{
		"env": map[string]string{
			"NODE_ENV":  "development",
			"LOG_LEVEL": "debug",
		},
	}
	writeJSON(t, claudeHome, "settings.json", settings)

	engine := NewEngine(claudeHome, nil)
	envVars := engine.ScanEnvVars("")

	if len(envVars) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(envVars))
	}

	found := map[string]string{}
	for _, e := range envVars {
		found[e.Key] = e.Value
		if e.Source != SourceGlobal {
			t.Errorf("expected source Global, got %v", e.Source)
		}
	}
	if found["NODE_ENV"] != "development" {
		t.Errorf("expected NODE_ENV=development, got %q", found["NODE_ENV"])
	}
	if found["LOG_LEVEL"] != "debug" {
		t.Errorf("expected LOG_LEVEL=debug, got %q", found["LOG_LEVEL"])
	}
}

func TestScanEnvVars_Project(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")
	_ = os.MkdirAll(claudeHome, 0o755)

	projectPath := filepath.Join(base, "myproject")
	settings := map[string]any{
		"env": map[string]string{
			"API_KEY": "secret",
		},
	}
	writeJSON(t, projectPath, ".claude/settings.local.json", settings)

	engine := NewEngine(claudeHome, nil)
	envVars := engine.ScanEnvVars(projectPath)

	if len(envVars) != 1 {
		t.Fatalf("expected 1 env var, got %d", len(envVars))
	}
	if envVars[0].Source != SourceProject {
		t.Errorf("expected source Project, got %v", envVars[0].Source)
	}
}

func TestScan_FullIntegration(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")
	projectPath := filepath.Join(base, "myproject")

	// Global: MCPs in ~/.claude.json
	writeJSON(t, base, ".claude.json", map[string]any{
		"mcpServers": map[string]any{
			"global-mcp": map[string]any{"type": "stdio", "command": "gm"},
		},
	})

	// Global: skills
	writeFile(t, claudeHome, "skills/global-skill/SKILL.md", "# Global Skill")

	// Global: hooks + permissions + env
	writeJSON(t, claudeHome, "settings.json", map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []map[string]any{{
				"hooks": []map[string]any{{"type": "command", "command": "echo hook"}},
			}},
		},
		"permissions": map[string]any{
			"allow": []string{"Bash(*)"},
		},
		"env": map[string]string{"DEBUG": "1"},
	})

	// Project: MCPs
	writeJSON(t, projectPath, ".mcp.json", map[string]any{
		"mcpServers": map[string]any{
			"project-mcp": map[string]any{"type": "stdio", "command": "pm"},
		},
	})

	// Project: skills
	writeFile(t, projectPath, ".claude/skills/project-skill/SKILL.md", "# Project Skill")

	// Project: hooks + permissions + env
	writeJSON(t, projectPath, ".claude/settings.local.json", map[string]any{
		"hooks": map[string]any{
			"PostToolUse": []map[string]any{{
				"hooks": []map[string]any{{"type": "command", "command": "echo post"}},
			}},
		},
		"permissions": map[string]any{
			"deny": []string{"DeleteFile(*)"},
		},
		"env": map[string]string{"API_KEY": "test"},
	})

	// Registry: one MCP
	reg := newRegistry()
	_ = reg.Add(model.ServerDef{Name: "registry-mcp", Transport: model.TransportStdio, Command: "rm"})

	engine := NewEngine(claudeHome, reg)
	items := engine.Scan(projectPath)

	// MCPs: 1 global + 1 project + 1 registry = 3
	if len(items.MCPs) != 3 {
		t.Errorf("expected 3 MCPs, got %d", len(items.MCPs))
	}

	// Skills: 1 global + 1 project = 2
	if len(items.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(items.Skills))
	}

	// Hooks: 1 global + 1 project = 2
	if len(items.Hooks) != 2 {
		t.Errorf("expected 2 hooks, got %d", len(items.Hooks))
	}

	// Permissions: 1 allow (global) + 1 deny (project) = 2
	if len(items.Permissions) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(items.Permissions))
	}

	// Env vars: 1 global + 1 project = 2
	if len(items.EnvVars) != 2 {
		t.Errorf("expected 2 env vars, got %d", len(items.EnvVars))
	}
}

func TestScan_EmptyFilesystem(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")
	// Don't create anything

	engine := NewEngine(claudeHome, nil)
	items := engine.Scan("")

	if len(items.MCPs) != 0 {
		t.Errorf("expected 0 MCPs, got %d", len(items.MCPs))
	}
	if len(items.Skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(items.Skills))
	}
	if len(items.Hooks) != 0 {
		t.Errorf("expected 0 hooks, got %d", len(items.Hooks))
	}
	if len(items.Permissions) != 0 {
		t.Errorf("expected 0 permissions, got %d", len(items.Permissions))
	}
	if len(items.EnvVars) != 0 {
		t.Errorf("expected 0 env vars, got %d", len(items.EnvVars))
	}
}

func TestScan_MalformedSettings(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")

	// Write malformed settings.json but valid .mcp.json
	writeFile(t, claudeHome, "settings.json", "{broken json")
	writeJSON(t, base, ".claude.json", map[string]any{
		"mcpServers": map[string]any{
			"ok-mcp": map[string]any{"type": "stdio", "command": "ok"},
		},
	})

	engine := NewEngine(claudeHome, nil)
	items := engine.Scan("")

	// MCPs from valid file should still be found
	if len(items.MCPs) != 1 {
		t.Errorf("expected 1 MCP despite malformed settings, got %d", len(items.MCPs))
	}
	// Hooks/permissions/env from malformed settings should be empty
	if len(items.Hooks) != 0 {
		t.Errorf("expected 0 hooks from malformed settings, got %d", len(items.Hooks))
	}
}

func TestParseSkillDescription_Frontmatter(t *testing.T) {
	base := setup(t)
	content := "---\nname: test\ndescription: meta\n---\n\n# My Skill\n\nDetails here."
	writeFile(t, base, "SKILL.md", content)

	desc := parseSkillDescription(filepath.Join(base, "SKILL.md"))
	if desc != "My Skill" {
		t.Errorf("expected 'My Skill', got %q", desc)
	}
}

func TestParseSkillDescription_NoFrontmatter(t *testing.T) {
	base := setup(t)
	content := "# Simple Skill\n\nSome content."
	writeFile(t, base, "SKILL.md", content)

	desc := parseSkillDescription(filepath.Join(base, "SKILL.md"))
	if desc != "Simple Skill" {
		t.Errorf("expected 'Simple Skill', got %q", desc)
	}
}

func TestParseSkillDescription_Empty(t *testing.T) {
	base := setup(t)
	writeFile(t, base, "SKILL.md", "")

	desc := parseSkillDescription(filepath.Join(base, "SKILL.md"))
	if desc != "" {
		t.Errorf("expected empty description, got %q", desc)
	}
}

func TestParseSkillDescription_MissingFile(t *testing.T) {
	desc := parseSkillDescription("/nonexistent/SKILL.md")
	if desc != "" {
		t.Errorf("expected empty description for missing file, got %q", desc)
	}
}

func TestSourceString(t *testing.T) {
	tests := []struct {
		source Source
		want   string
	}{
		{SourceGlobal, "global"},
		{SourceProject, "project"},
		{SourceRegistry, "registry"},
		{Source(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.source.String(); got != tt.want {
			t.Errorf("Source(%d).String() = %q, want %q", tt.source, got, tt.want)
		}
	}
}

func TestScanHooks_MalformedHookEntry(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")

	// Write hooks with an invalid format (not an array of matchers)
	settings := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": "not-an-array",
		},
	}
	writeJSON(t, claudeHome, "settings.json", settings)

	engine := NewEngine(claudeHome, nil)
	hooks := engine.ScanHooks("")

	// Should gracefully skip malformed hook entries
	if len(hooks) != 0 {
		t.Errorf("expected 0 hooks for malformed entry, got %d", len(hooks))
	}
}

func TestScanHooks_MultipleHooksPerMatcher(t *testing.T) {
	base := setup(t)
	claudeHome := filepath.Join(base, ".claude")

	settings := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []map[string]any{
				{
					"matcher": "Bash",
					"hooks": []map[string]any{
						{"type": "command", "command": "echo first"},
						{"type": "command", "command": "echo second"},
					},
				},
			},
		},
	}
	writeJSON(t, claudeHome, "settings.json", settings)

	engine := NewEngine(claudeHome, nil)
	hooks := engine.ScanHooks("")

	if len(hooks) != 2 {
		t.Fatalf("expected 2 hooks (multiple per matcher), got %d", len(hooks))
	}
	if hooks[0].Command != "echo first" {
		t.Errorf("expected first hook command, got %q", hooks[0].Command)
	}
	if hooks[1].Command != "echo second" {
		t.Errorf("expected second hook command, got %q", hooks[1].Command)
	}
}

func TestBuildHookName(t *testing.T) {
	tests := []struct {
		event, matcher string
		index          int
		want           string
	}{
		{"PreToolUse", "Bash", 0, "PreToolUse:Bash"},
		{"PreToolUse", "", 0, "PreToolUse"},
		{"PostToolUse", "Bash", 1, "PostToolUse:Bash:i"},
		{"PostToolUse", "Bash", 2, "PostToolUse:Bash:ii"},
	}
	for _, tt := range tests {
		got := buildHookName(tt.event, tt.matcher, tt.index)
		if got != tt.want {
			t.Errorf("buildHookName(%q, %q, %d) = %q, want %q",
				tt.event, tt.matcher, tt.index, got, tt.want)
		}
	}
}
