# hystak Architecture & Design Reference

## What is hystak

hystak is a Go CLI/TUI tool that manages MCP (Model Context Protocol) server configurations from a central registry and deploys them to MCP client config files. It controls which MCP servers Claude Code connects to, what hooks fire, what permissions are granted, what skills are available, and what CLAUDE.md instructions Claude follows — all from a single binary with zero runtime dependencies.

---

## Features

### CLI Commands

| Command | Purpose |
|---------|---------|
| `hystak` | Interactive launcher with profile picker (TUI if terminal) |
| `hystak manage` | Full management TUI for all resources |
| `hystak list` | Tabular server listing |
| `hystak sync [project]` | Deploy project configs (`--all`, `--profile`) |
| `hystak diff <project>` | Show drift as unified diff |
| `hystak import <path>` | Import servers from client config with conflict resolution |
| `hystak run <project>` | Sync + launch client (`--no-sync`, `--dry-run`, `--profile`, `-- <args>`) |
| `hystak backup` | Snapshot client configs (`--all`, `--list`) |
| `hystak restore <project>` | Interactive or `--index` based restore |
| `hystak profile list/export/import` | Profile management |
| `hystak setup` | First-run setup wizard |
| `hystak version` | Version info |

### Core Capabilities

- **Registry Management** — Central catalog of MCP servers, skills, hooks, permissions, templates, prompts with full CRUD
- **Project Management** — Per-project server assignments with override merging
- **Profiles** — Named loadouts (MCPs/skills/hooks/permissions/prompts/env/claudeMD/isolation) per project
- **Tags** — Named server groups for bulk assignment
- **Sync/Deploy** — Deploy to client config files (`.mcp.json` project scope, `~/.claude.json` global scope)
- **Semantic Drift Detection** — Compares only deployed-relevant fields, ignores formatting/metadataa
- **Import** — Pull existing servers from client configs with interactive conflict resolution
- **Auto-Discovery** — Scan `~/.claude/` and project dirs to find MCPs, skills, hooks, permissions, env vars, prompts
- **Backup & Restore** — Timestamp-indexed snapshots with configurable retention
- **Worktree Isolation** — Git worktrees for concurrent Claude Code sessions
- **Profile Sharing** — Export/import profiles as YAML
- **Built-in Catalog** — Curated MCPs, skills, hooks, permissions for quick setup

### TUI Features

- 8 tabs: Profiles, Tools, MCPs, Skills, Hooks, Permissions, Templates, Prompts
- Modal overlays: Form, Confirm, Diff, Import, Discovery, Conflict Resolution, Launch Wizard
- Two-pane profile view (project list + 5-section detail)
- Customizable keybindings via `keys.yaml`
- First-run setup wizard

---

## Package Map (18 packages)

```
internal/
├── backup/      — Config snapshot engine
├── catalog/     — Built-in curated server/skill/hook catalog
├── cli/         — Cobra command tree (12 commands)
├── config/      — XDG paths, user config, legacy migration
├── deploy/      — Deployer + ResourceDeployer implementations
├── discovery/   — Auto-detection engine for MCPs/skills/hooks/permissions/env/prompts
├── errors/      — Custom error types (ProjectNotFound, ServerNotFound, etc.)
├── isolation/   — Worktree & lock-based concurrency
├── keyconfig/   — Keybinding configuration from keys.yaml
├── launch/      — Client process execution (platform-specific)
├── model/       — 9 domain types, all implementing Resource interface
├── profile/     — Profile manager (global profiles, export/import)
├── project/     — Project store with resolution algorithm
├── registry/    — Generic Store[T] + Registry (servers/skills/hooks/perms/templates/prompts/tags)
├── service/     — Orchestration layer (sync, diff, import, discover, backup)
└── tui/         — Bubble Tea app (8 tabs, 12 modes, 7 form types)
```

### Layered Architecture

```
main.go → cli/ (Cobra root dispatcher)
    ├── no args + terminal → TUI picker → profile selection → sync → launch
    ├── manage             → full management TUI
    ├── launch wizard      → discovery + profile builder
    └── subcommand        → CLI handlers
            ↓
        service/ (orchestration & business logic)
            ↓
    ┌───────┬────────────┬─────────────┬──────────┬────────────┬──────────────┐
 registry/ project/ deploy/ profile/ discovery/ backup/ isolation/ launch/
 (YAML)    (YAML)   (client                                              (exec)
           assign)  deployers)
```

### Package Responsibilities

| Layer | Owns | Does NOT own |
|-------|------|-------------|
| `model/` | Type definitions | No I/O, no logic |
| `registry/` | YAML persistence + tag integrity | No deployment, no business rules |
| `project/` | Project persistence + resolution algorithm | No client awareness |
| `deploy/` | Client-specific config translation | No registry access, no business decisions |
| `service/` | Orchestration (sync, diff, import, discover) | No file format knowledge, no UI |
| `cli/` | Argument parsing + output formatting | No business logic |
| `tui/` | User interaction + visual state | No business logic |

---

## Design Patterns

### DRY: Generic `Store[T, PT]`

One 117-line file (`registry/store.go`) serves all 6 resource types. Without it, you'd write `Add`, `Get`, `Update`, `Delete`, `List`, `Items`, `SetItems` six times.

**The 4-line interface that makes it work** (`model/resource.go`):
```go
type Resource interface {
    ResourceName() string
    SetResourceName(name string)
}
```

**The generic type with `Resource` constraint** (`registry/store.go`):
```go
type Store[T any, PT interface {
    model.Resource
    *T
}] struct {
    items    map[string]T
    kind     string
    sortFunc func(a, b T) bool
}
```

**Every model type satisfies it with 2 lines** (`model/server.go`):
```go
func (s *ServerDef) ResourceName() string    { return s.Name }
func (s *ServerDef) SetResourceName(n string) { s.Name = n }
```

**6 stores instantiated from the same generic** (`registry/registry.go`):
```go
func empty() *Registry {
    return &Registry{
        Servers:     NewStore[model.ServerDef, *model.ServerDef]("server"),
        Skills:      NewStore[model.SkillDef, *model.SkillDef]("skill"),
        Hooks:       NewStore[model.HookDef, *model.HookDef]("hook"),
        Permissions: NewStore[model.PermissionRule, *model.PermissionRule]("permission"),
        Templates:   NewStore[model.TemplateDef, *model.TemplateDef]("template"),
        Prompts:     NewStore[model.PromptDef, *model.PromptDef]("prompt").WithSort(...),
        Tags:        make(map[string][]string),
    }
}
```

The `kind` string field gives type-specific error messages (`"server 'foo' not found"` vs `"skill 'bar' not found"`) without branching.

### DRY: Unified ResourceDeployer Loop

One interface for skills, settings, and CLAUDE.md (`deploy/resource_deployer.go`):
```go
type ResourceDeployer interface {
    Kind() ResourceDeployerKind
    Sync(projectPath string, config DeployConfig) error
    Preflight(projectPath string, config DeployConfig) []PreflightConflict
    ReadDeployed(projectPath string) (DeployConfig, error)
}
```

The sync loop that consumes it — 3 deployers, 4 lines (`service/service.go`):
```go
for _, rd := range s.resourceDeployers {
    if err := rd.Sync(proj.Path, dcfg); err != nil {
        return nil, fmt.Errorf("syncing %s for project %q: %w", rd.Kind(), proj.Name, err)
    }
}
```

Same pattern for preflight conflict detection:
```go
for _, rd := range s.resourceDeployers {
    for _, c := range rd.Preflight(proj.Path, dcfg) {
        conflicts = append(conflicts, SyncConflict{...})
    }
}
```

Adding a new resource deployer = implement the interface + append to the slice. Zero changes to sync/preflight/drift logic.

### SRP: Service Doesn't Know File Formats

The service resolves names to data, then hands off to deployers that own the translation:

```go
func (s *Service) buildDeployConfig(cfg effectiveConfig) deploy.DeployConfig {
    skills := make([]model.SkillDef, 0, len(cfg.skills))
    for _, name := range cfg.skills {
        if skill, ok := s.registry.Skills.Get(name); ok {
            skills = append(skills, skill)
        }
    }
    // ... same for hooks, permissions, templates, prompts
    return deploy.DeployConfig{Skills: skills, Hooks: hooks, ...}
}
```

The service never touches `.mcp.json`, `settings.local.json`, or symlinks. Each deployer owns its own file format entirely:
- `SkillsDeployer` knows about symlinks
- `SettingsDeployer` knows about `settings.local.json`
- `ClaudeCodeDeployer` knows about `.mcp.json` JSON schema


### SRP: Semantic Equality Ignores Metadata

`Equal` compares only deployed-relevant fields (`model/server.go`):
```go
func (a ServerDef) Equal(b ServerDef) bool {
    return a.Transport == b.Transport &&
        a.Command == b.Command &&
        a.URL == b.URL &&
        slices.Equal(a.Args, b.Args) &&
        maps.Equal(a.Env, b.Env) &&
        maps.Equal(a.Headers, b.Headers)
}
```

`Name` and `Description` are deliberately excluded — they're registry metadata, not deployment state. This is what makes drift detection semantic rather than textual.

### Extensibility: Factory + Interface for Clients

Adding a new client means one map entry (`deploy/deployer.go`):
```go
var deployerFactories = map[model.ClientType]func() Deployer{
    model.ClientClaudeCode: func() Deployer { return &ClaudeCodeDeployer{} },
    // future: model.ClientCursor: func() Deployer { return &CursorDeployer{} },
}
```

The sync loop is already client-agnostic (`service/service.go`):
```go
for _, ct := range proj.Clients {
    deployer, ok := s.deployers[ct]
    // ... Bootstrap, ReadServers, WriteServers — all via interface
}
```

A new client needs: one struct implementing 5 methods + one factory entry. Sync, diff, import, backup all work immediately.

### Extensibility: Dual YAML Format via Custom Marshaling

Bare string or map, decided at marshal time (`model/project.go`):
```go
func (a MCPAssignment) MarshalYAML() (interface{}, error) {
    if a.Overrides == nil {
        return a.Name, nil       // "- github"
    }
    return map[string]mcpAssignmentValue{
        a.Name: {Overrides: a.Overrides},
    }, nil                       // "- github: {overrides: {env: ...}}"
}
```

Unmarshal detects node kind:
```go
func (a *MCPAssignment) UnmarshalYAML(value *yaml.Node) error {
    if value.Kind == yaml.ScalarNode {
        a.Name = value.Value     // bare string
        return nil
    }
    if value.Kind == yaml.MappingNode {
        a.Name = value.Content[0].Value
        // decode overrides from value
        return nil
    }
    return fmt.Errorf("expected string or map, got %v", value.Kind)
}
```

### Extensibility: `DeployConfig` as a Shared Envelope

```go
type DeployConfig struct {
    Skills         []model.SkillDef
    Hooks          []model.HookDef
    Permissions    []model.PermissionRule
    TemplateSource string
    PromptSources  []string
}
```

Every `ResourceDeployer` receives the full config but reads only its fields. Adding a new field means adding it to the struct — existing deployers ignore it.

### Summary of All Patterns

| Pattern | Where | Details |
|---------|-------|---------|
| Layered Architecture | Entire app | CLI → Service → Data/Deploy layers |
| Strategy / Interface Polymorphism | `deploy/` | `Deployer` for clients; `ResourceDeployer` for resources |
| Generics + Type Constraints | `registry/store.go` | `Store[T, PT]` constrained by `Resource` interface |
| Factory Pattern | `deploy/deployer.go` | `NewDeployer(ct)` returns client-specific deployer |
| Elm Architecture | `tui/` | Bubble Tea Model-Update-View with custom `tea.Msg` types |
| Message Passing | `tui/app.go` | Child-to-parent communication via typed messages |
| Composition Over Inheritance | Everywhere | `Service` composes Registry, Store, deployers, profiles, backups |
| Dual YAML Format | `model/project.go` | Custom Marshal/Unmarshal for `MCPAssignment` |
| Override Merge Rules | `model/server.go` | env/headers: map merge; args: replace; command/url: replace if non-nil |
| Conflict Resolution | `service/`, `tui/` | `ImportCandidate` + `SyncConflict` with resolution enums |
| Symlink-based Management | `deploy/skills.go` | Symlinks = managed; regular files = user-owned |
| Sentinel Markers | `deploy/claude_md.go` | `<!-- managed by hystak -->` marks generated files |
| XDG Compliance | `config/` | Config path resolution with legacy migration |
| Platform Abstraction | `launch/` | `exec_unix.go` / `exec_windows.go` build-tag separation |

---

## The Core Extensibility Pipeline

The codebase follows a consistent **interface → generic container → uniform orchestration** pipeline:

```
Resource interface
    → Store[T, PT] (one generic store for all types)
        → Registry (one serialization layer for all stores)
            → Service (one orchestration layer)

Deployer interface
    → deployerFactories map (one factory for all clients)
        → Service.SyncProject (one sync path for all clients)

ResourceDeployer interface
    → []ResourceDeployer slice (one loop for all resource types)
        → Service.SyncProject (same sync path)
```

New types and new clients slot into existing loops rather than requiring new code paths. The core principle: **extend by implementing interfaces, not by modifying orchestration**.

---

## Testing

### Framework

- **Go stdlib `testing` package** — no external test frameworks
- **`charmbracelet/x/exp/teatest`** — snapshot/golden testing for Bubble Tea TUI
- **VHS tape scripts** — E2E visual output testing

### Distribution

- **39 test files**, **638 test functions** across all packages
- Unit tests in every `internal/` package
- TUI tests using `teatest.TestModel` with keyboard simulation
- E2E tests in `e2e/tapes/` (help, list_servers, tui_startup)

### Patterns

| Pattern | Usage |
|---------|-------|
| Table-driven subtests | `t.Run()` throughout |
| Isolated temp dirs | `t.TempDir()` for file-based tests |
| Interface mocks | `mockDeployer` with compile-time check `var _ deploy.Deployer = (*mockDeployer)(nil)` |
| Golden file comparison | `teatest.RequireEqualOutput()` for TUI snapshots |
| YAML round-trip tests | `TestServerDefYAMLRoundTrip_Stdio` etc. |
| Seeded fixtures | `setupTestConfig()` creates registry/projects YAML in temp dirs |
| Race detection | `make test-race` runs `go test -race ./...` |
| Coverage reporting | `make test-cover` → `coverage.out` + HTML |

---

## Dependencies (7 direct)

| Dependency | Purpose |
|------------|---------|
| `spf13/cobra` v1.10.2 | CLI framework |
| `gopkg.in/yaml.v3` | YAML config I/O |
| `charmbracelet/bubbletea` v1.3.10 | TUI framework (Elm architecture) |
| `charmbracelet/bubbles` v1.0.0 | TUI components (inputs, lists) |
| `charmbracelet/lipgloss` v1.1.0 | Terminal styling |
| `charmbracelet/x/exp/teatest` | TUI snapshot testing |
| `mattn/go-isatty` v0.0.20 | TTY detection |

---

## Build & Release

- **Go 1.25.6**, CGO disabled
- **goreleaser** — Multi-platform builds (Linux/macOS amd64+arm64, Windows amd64)
- **ldflags** inject `version`, `commit`, `date` at build time
- **Makefile targets**: build, test, test-race, test-update, test-cover, lint, e2e, snapshot, clean

---

## User Journey

### Phase 1: First Run

```
$ hystak                          (no args, terminal detected)
    │
    ├─ PersistentPreRunE
    │   ├─ Migrate legacy config (~/.config/hystak → ~/.hystak)
    │   ├─ Create service (loads registry.yaml + projects.yaml)
    │   ├─ Auto-discover: scan ~/.claude.json + project .mcp.json files
    │   └─ Load keybindings
    │
    ├─ service.IsEmpty() == true → Launch Setup Wizard
    │
    └─ WIZARD (8 steps):
        1. Welcome screen
        2. Keybinding profile (Arrows / Vim / Classic)
        3. Scan results — discovered MCPs from existing configs, toggle to import
        4. Catalog browser — built-in MCPs, skills, hooks, permissions
        5. Project form — name + path (auto-fills from cwd)
        6. Summary — review all selections
        7. Apply — imports servers, creates project, assigns resources
        8. Done → drops into Management TUI
```

### Phase 2: Management TUI

```
┌─ Profiles ─ Tools ─ MCPs ─ Skills ─ Hooks ─ Permissions ─ Templates ─ Prompts ─┐
│                                                                                  │
│  Profiles tab: two-pane layout                                                   │
│    Left: project list with active profile indicator                              │
│    Right: 5 sections (MCPs, Skills, Hooks, Permissions, Template)                │
│                                                                                  │
│  Tools tab: 4 actions                                                            │
│    [Sync] [Diff] [Discover] [Launch]                                             │
│                                                                                  │
│  Other tabs: CRUD lists with form overlays                                       │
│                                                                                  │
│  Modal overlays: Form, Confirm, Diff viewer, Importer, Conflict resolver         │
└──────────────────────────────────────────────────────────────────────────────────┘
```

### Phase 3: Launch

When user triggers Launch from TUI or runs `hystak run <project>`:

```
syncAndLaunch()
    │
    ├─ Resolve isolation strategy
    │   ├─ none: deploy to project path directly
    │   ├─ worktree: create <project>.hystak-wt-<profile>/, deploy there
    │   └─ lock: acquire .hystak.lock, deploy to project path
    │
    ├─ SyncProject()  ← deploys all configs (see "What SyncProject Writes" below)
    │
    ├─ launch.RunCommand(claude, workDir)  ← spawns Claude Code as child process
    │   └─ Parent ignores SIGINT/SIGTERM while child runs
    │
    └─ POST-EXIT LOOP:
        ┌─────────────────────────────────────────┐
        │  Claude exited. What next?              │
        │  [R]elaunch  [C]onfigure  [Q]uit        │
        └─────────────────────────────────────────┘
        R → re-sync + relaunch
        C → open Launch Wizard (hub mode) → edit profile → re-sync + relaunch
        Q → exit hystak
```

### Phase 4: Daily Use

```
$ hystak                 → Management TUI (auto-discovers new MCPs on startup)
$ hystak run myproject   → Sync + launch Claude Code directly
$ hystak sync myproject  → Deploy configs without launching
$ hystak diff myproject  → Show drift between registry and deployed
$ hystak import .mcp.json → Pull servers from existing config into registry
```

---

## What SyncProject Writes (and How It Affects Claude Code)

### 1. MCP Servers → `.mcp.json` / `~/.claude.json`

```
Registry (ServerDef)              Claude Code config
─────────────────────    →    ─────────────────────────────
name: github                  .mcp.json:
transport: stdio              {
command: npx                    "mcpServers": {
args: [-y, @modelcontextprotocol/     "github": {
       server-github]                   "type": "stdio",
env:                                    "command": "npx",
  GITHUB_TOKEN: ${GITHUB_TOKEN}         "args": ["-y", "@modelcontextprotocol/server-github"],
                                        "env": { "GITHUB_TOKEN": "${GITHUB_TOKEN}" }
                                      }
                                    }
                                  }
```

**Written by:** `ClaudeCodeDeployer.WriteServers()` in `deploy/claude_code.go`

**Where:**
- Project scope: `<project>/.mcp.json`
- Global scope: `~/.claude.json`

**Effect on Claude Code:** These are the MCP servers Claude Code connects to at startup. Each server provides tools (GitHub operations, filesystem access, database queries, etc.) that Claude can invoke during conversations.

### 2. Hooks + Permissions → `.claude/settings.local.json`

```
Registry (HookDef)                      settings.local.json
──────────────────────    →    ─────────────────────────────────
name: lint-on-bash                {
event: PreToolUse                   "hooks": {
matcher: Bash                         "PreToolUse": [{
command: eslint --fix                   "matcher": "Bash",
timeout: 5000                           "hooks": [{
                                          "type": "command",
Registry (PermissionRule)                 "command": "eslint --fix",
──────────────────────                    "timeout": 5000
name: allow-bash                        }]
rule: Bash(*)                         }]
type: allow                         },
                                    "permissions": {
name: allow-github                    "allow": ["Bash(*)", "WebFetch(domain:github.com)"],
rule: WebFetch(domain:github.com)     "deny": []
type: allow                         }
                                  }
```

**Written by:** `SettingsDeployer.Sync()` in `deploy/settings.go`

**Where:** `<project>/.claude/settings.local.json`

**Effect on Claude Code:**
- **Hooks** fire shell commands before/after tool use (validation, linting, logging)
- **Permissions** control which tools Claude can invoke and with what arguments — this is the safety boundary

### 3. Skills → `.claude/skills/<name>/SKILL.md` (symlinks)

```
Registry (SkillDef)                   Filesystem
─────────────────────    →    ─────────────────────────────────
name: code-review                 .claude/skills/
source: ~/.hystak/skills/           code-review/
        code-review.md                SKILL.md → ~/.hystak/skills/code-review.md
                                    commit-helper/
name: commit-helper                   SKILL.md → ~/.hystak/skills/commit-helper.md
source: ~/.hystak/skills/
        commit-helper.md
```

**Written by:** `SkillsDeployer.SyncSkills()` in `deploy/skills.go`

**Where:** `<project>/.claude/skills/<name>/SKILL.md` (symlinks to source files)

**Effect on Claude Code:** Skills are custom slash commands / capabilities. Claude Code scans `.claude/skills/*/SKILL.md` and makes them available as invocable skills during conversation.

### 4. CLAUDE.md → Project Root

**Symlink mode** (template only, no prompts):
```
<project>/CLAUDE.md  →  ~/.hystak/templates/my-project-template.md
```

**Composed mode** (template + prompt fragments):
```
<!-- managed by hystak -->

[template content from ~/.hystak/templates/base.md]

[prompt fragment from ~/.hystak/prompts/safety-rules.md]

[prompt fragment from ~/.hystak/prompts/code-style.md]
```

**Written by:** `ClaudeMDDeployer` in `deploy/claude_md.go`

**Where:** `<project>/CLAUDE.md`

**Effect on Claude Code:** This is **the most direct influence on Claude's behavior**. CLAUDE.md content is loaded as system-level instructions at conversation start. It shapes how Claude reasons, what conventions it follows, what it refuses, and what patterns it prefers. Prompts are sorted by `Order` field, so you control instruction priority.

---

## Complete File Map

```
hystak owns (its own state):                Claude Code reads (runtime behavior):
──────────────────────────────              ─────────────────────────────────────

~/.hystak/                                  ~/.claude.json
  registry.yaml  ──── servers ────────────→   mcpServers: { ... }     ← global MCP connections
  projects.yaml
  profiles/*.yaml                           <project>/
  backups/**/*                                .mcp.json ──────────────→ mcpServers: { ... }  ← project MCP connections
                                              .claude/
                                                settings.local.json ──→ hooks: { ... }       ← pre/post tool commands
                                                                        permissions: { ... }  ← tool access control
                                                skills/
                                                  <name>/SKILL.md ────→ skill definitions    ← custom capabilities
                                              CLAUDE.md ──────────────→ system instructions  ← model context/behavior
```

### All File Paths

| Purpose | Path | Type | Written By | Read By |
|---------|------|------|-----------|---------|
| Registry | `~/.hystak/registry.yaml` | YAML | Service | Service |
| Projects | `~/.hystak/projects.yaml` | YAML | Service | Service |
| Profiles | `~/.hystak/profiles/*.yaml` | YAML | Service | Service |
| Backups | `~/.hystak/backups/**/*` | JSON | BackupManager | BackupManager |
| Global MCPs | `~/.claude.json` | JSON | ClaudeCodeDeployer | Claude Code |
| Project MCPs | `<proj>/.mcp.json` | JSON | ClaudeCodeDeployer | Claude Code |
| Project Settings | `<proj>/.claude/settings.local.json` | JSON | SettingsDeployer | Claude Code |
| Project Skills | `<proj>/.claude/skills/*/SKILL.md` | Symlinks | SkillsDeployer | Claude Code |
| Project CLAUDE.md | `<proj>/CLAUDE.md` | Symlink or File | ClaudeMDDeployer | Claude Code |
| Lock File | `<proj>/.hystak.lock` | Text (PID) | LockManager | LockManager |
| Worktree | `<proj>.hystak-wt-*` | Git Worktree | WorktreeManager | Git + Deployers |

---

## Profile-Driven Selection

Profiles select which subset of registry items get deployed:

```
Registry (all available):          Profile "dev-mode":        Deployed to project:
─────────────────────────          ─────────────────          ────────────────────
MCPs:                              mcps:                      .mcp.json:
  github ✓                           - github                   github, filesystem
  filesystem ✓                       - filesystem
  slack ✗                          skills:                    .claude/skills/:
  jira ✗                             - code-review              code-review/SKILL.md
Skills:                            hooks:                     settings.local.json:
  code-review ✓                      - lint-on-bash              hooks: {PreToolUse: [...]}
  commit-helper ✗                  permissions:                  permissions: {allow: [...]}
Hooks:                               - allow-bash
  lint-on-bash ✓                   claude_md: base            CLAUDE.md:
  notify-slack ✗                   prompts:                     [base template + safety rules]
Permissions:                         - safety-rules
  allow-bash ✓                     isolation: worktree
  allow-github ✗
Templates:
  base ✓
Prompts:
  safety-rules ✓
  code-style ✗
```

Switching profiles instantly reconfigures which MCP servers Claude connects to, what hooks fire, what permissions are granted, what skills are available, and what instructions Claude follows — all from one `hystak sync`.

---

## Key Types & Interfaces

### Model Types (`internal/model/`)

- **`ServerDef`** — Canonical MCP definition (name, transport, command, args, env, url, headers)
- **`ServerOverride`** — Per-project overrides (nil/empty fields ignored during merge)
- **`MCPAssignment`** — Server assignment with optional overrides (custom YAML dual format)
- **`Project`** — Project registration (name, path, clients, tags, MCPs, skills, hooks, permissions, profiles, active_profile, managed_MCPs)
- **`ProjectProfile`** — Inline profile in project (MCPs, skills, hooks, permissions, prompts, env, claude_md, isolation)
- **`Transport`** — Enum (stdio/sse/http)
- **`ClientType`** — Enum (claude-code/claude-desktop/cursor)
- **`DriftStatus`** — Enum (synced/drifted/missing/unmanaged)
- **`SkillDef`** — Skill definition (name, description, source path)
- **`HookDef`** — Hook definition (name, event, matcher, command, timeout)
- **`PermissionRule`** — Permission (name, rule, type: allow/deny)
- **`TemplateDef`** — CLAUDE.md template (name, source path)
- **`PromptDef`** — Prompt fragment (name, description, source, category, order, tags)

### Interface Types

- **`Resource`** — Constraint for all registry resource types (ResourceName, SetResourceName)
- **`Deployer`** — MCP server deployment (ClientType, ConfigPath, ReadServers, WriteServers, Bootstrap)
- **`ResourceDeployer`** — Non-MCP resource deployment (Kind, Sync, Preflight, ReadDeployed)

### Service Enums

- **`SyncAction`** — added/updated/unchanged/unmanaged
- **`ImportResolution`** — pending/keep/replace/rename/skip
- **`SyncConflictResolution`** — pending/keep/replace/skip

### Override Merge Rules

```
command: replace if non-nil
url:     replace if non-nil
args:    replace entirely
env:     map merge (override keys win)
headers: map merge (override keys win)
```
