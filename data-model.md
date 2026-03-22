# hystak — Data Model Specification

## File Layout

```
~/.hystak/
├── registry.yaml          # All managed resources (MCPs, skills, hooks, etc.)
├── projects.yaml          # Project definitions + profile assignments
├── profiles/              # One YAML file per profile
│   ├── default.yaml
│   ├── dev.yaml
│   ├── review.yaml
│   ├── minimal.yaml
│   └── empty.yaml         # Built-in, ships with hystak
├── user.yaml              # User preferences
├── keys.yaml              # Keybinding configuration
└── backups/               # Timestamped config snapshots
    ├── myproject_mcp_2026-03-22T10:30:00.json
    ├── myproject_settings_2026-03-22T10:30:00.json
    └── global_claude_2026-03-22T10:30:00.json
```

---

## registry.yaml

Single file containing all registered resources. Each resource type is a top-level key.

```yaml
# ~/.hystak/registry.yaml

mcps:
  github:
    transport: stdio                    # stdio | sse | http
    command: npx                        # Required for stdio
    args: ["-y", "@anthropic/mcp-github"]
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN}"   # Supports env var interpolation
    description: "GitHub MCP server"

  postgres:
    transport: stdio
    command: npx
    args: ["-y", "@anthropic/mcp-postgres"]
    env:
      DATABASE_URL: "${DATABASE_URL}"

  remote-api:
    transport: sse
    url: "https://mcp.example.com/sse"  # Required for sse/http
    headers:
      Authorization: "Bearer ${API_KEY}"

skills:
  code-review:
    description: "Structured code review skill"
    source: "/Users/me/skills/code-review/SKILL.md"  # Absolute path

  commit:
    description: "Conventional commit helper"
    source: "/Users/me/skills/commit/SKILL.md"

hooks:
  lint-on-edit:
    event: PostToolUse                  # PreToolUse | PostToolUse | Notification | Stop
    matcher: "Edit"                     # Tool name pattern
    command: "npm run lint --fix"
    timeout: 30                         # Seconds, default 30

  block-rm-rf:
    event: PreToolUse
    matcher: "Bash(rm -rf *)"
    command: "echo 'Blocked dangerous command' && exit 1"
    timeout: 5

permissions:
  allow-bash:
    rule: "Bash(*)"
    type: allow

  deny-rm:
    rule: "Bash(rm -rf /)"
    type: deny

  allow-read:
    rule: "Read(*)"
    type: allow

templates:
  standard:
    source: "/Users/me/templates/standard-claude.md"

  minimal:
    source: "/Users/me/templates/minimal-claude.md"

prompts:
  security-rules:
    description: "Security guidelines for code generation"
    source: "/Users/me/prompts/security.md"
    category: "safety"
    order: 10                           # Lower = earlier in composed output
    tags: ["security", "default"]

  style-guide:
    description: "Code style conventions"
    source: "/Users/me/prompts/style.md"
    category: "conventions"
    order: 20
    tags: ["style"]

tags:
  core-tools:
    members: [github, postgres]

  all-remote:
    members: [remote-api]
```

### Field Reference — MCPs

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `transport` | `stdio` \| `sse` \| `http` | Yes | Transport protocol |
| `command` | string | Yes (stdio) | Executable command |
| `args` | string[] | No | Command arguments |
| `env` | map[string]string | No | Environment variables (supports `${VAR}` interpolation) |
| `url` | string | Yes (sse/http) | Server URL |
| `headers` | map[string]string | No | HTTP headers (supports `${VAR}` interpolation) |
| `description` | string | No | Human-readable description |

### Field Reference — Tags

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `members` | string[] | Yes | List of MCP server names (must exist in `mcps`) |

---

## projects.yaml

Maps project directories to their configuration. Each project has one or more profiles.

```yaml
# ~/.hystak/projects.yaml

projects:
  interview-platform:
    path: "/Volumes/Secondary/workspace/interview-platform"
    active_profile: dev                 # Currently active profile name
    managed_mcps:                       # Servers deployed by hystak (for cleanup tracking)
      - github
      - postgres

  personal-site:
    path: "/Users/me/projects/personal-site"
    active_profile: default
    managed_mcps:
      - github

  # Global scope project (deploys to ~/.claude.json)
  global:
    path: "~"
    active_profile: default
    managed_mcps:
      - github
```

---

## profiles/ Directory

One YAML file per profile. Profiles can be **global** (usable by any project) or **project-scoped** (only for a specific project).

```yaml
# ~/.hystak/profiles/dev.yaml

name: dev
description: "Full development environment with all tools"
scope: global                           # global | project-scoped
project: null                           # Set if scope is project-scoped

mcps:
  # Bare string = use registry definition as-is
  - github
  - postgres
  # Map = registry definition + overrides
  - remote-api:
      overrides:
        env:
          API_KEY: "${DEV_API_KEY}"     # Override for this profile
        args: ["--verbose"]             # Replaces base args entirely

skills:
  - code-review
  - commit

hooks:
  - lint-on-edit

permissions:
  - allow-bash
  - allow-read
  - deny-rm

template: standard                      # References templates.standard in registry
prompts:
  - security-rules
  - style-guide

env:                                    # Profile-level env vars (available during sync)
  NODE_ENV: development
  DEBUG: "true"

isolation: none                         # none | worktree | lock
```

### Built-in Profile: empty

```yaml
# ~/.hystak/profiles/empty.yaml

name: empty
description: "Clean Claude Code launch — no MCPs, skills, hooks, or permissions"
scope: global
project: null

mcps: []
skills: []
hooks: []
permissions: []
template: null
prompts: []
env: {}
isolation: none
```

### Override Merge Rules

When sync encounters a map-style MCP assignment with `overrides`:

| Field | Merge Strategy | Example |
|-------|---------------|---------|
| `env` | Map merge — override keys win, base keys preserved | Base `{A: 1, B: 2}` + Override `{B: 3, C: 4}` = `{A: 1, B: 3, C: 4}` |
| `headers` | Map merge — override keys win | Same as env |
| `args` | Full replacement — override replaces entire array | Base `[--flag]` + Override `[--other]` = `[--other]` |
| `command` | Replace if set | Override `python3` replaces base `python` |
| `url` | Replace if set | Override URL replaces base URL |

---

## user.yaml

```yaml
# ~/.hystak/user.yaml

auto_sync: true                         # Auto-discover and import on startup
backup_policy: always                   # always | never
max_backups: 10                         # Per-scope retention limit
```

---

## keys.yaml

```yaml
# ~/.hystak/keys.yaml

profile: arrows                         # arrows | vim | classic

# Full override (optional — profile provides defaults)
bindings:
  next_tab: ["Tab", "l"]               # Multiple keys per action
  prev_tab: ["Shift+Tab", "h"]
  list_up: ["Up", "k"]
  list_down: ["Down", "j"]
  select: ["Space"]
  confirm: ["Enter"]
  cancel: ["Esc", "q"]
  add: ["a"]
  edit: ["e"]
  delete: ["d"]
  filter: ["/"]
  launch: ["l"]                         # Context: Projects tab
  import: ["i"]                         # Context: Registry tab
  preview: ["p"]                        # Context: Prompts sub-view
  sync_from_diff: ["s"]                 # Context: Drift overlay
```

### Default Keybinding Profiles

| Action | Arrows | Vim | Classic |
|--------|--------|-----|---------|
| Next tab | Tab | Tab, l | Tab |
| Prev tab | Shift+Tab | Shift+Tab, h | Shift+Tab |
| List up | Up | k | Up |
| List down | Down | j | Down |
| Page up | PgUp | Ctrl+u | PgUp |
| Page down | PgDn | Ctrl+d | PgDn |
| Top | Home | g, g | Home |
| Bottom | End | G | End |

---

## Deployed File Formats

### .mcp.json (per-project)

Written by hystak during sync. Preserves non-`mcpServers` keys.

```json
{
  "mcpServers": {
    "github": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@anthropic/mcp-github"],
      "env": {
        "GITHUB_TOKEN": "ghp_abc123"
      }
    },
    "manually-added": {
      "type": "stdio",
      "command": "node",
      "args": ["my-server.js"]
    }
  }
}
```

- `github`: managed by hystak (tracked in `managed_mcps`)
- `manually-added`: unmanaged (preserved during sync)

### settings.local.json (per-project)

Written to `<project>/.claude/settings.local.json`.

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Edit",
        "command": "npm run lint --fix",
        "timeout": 30
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Bash(rm -rf *)",
        "command": "echo 'Blocked dangerous command' && exit 1",
        "timeout": 5
      }
    ]
  },
  "permissions": {
    "allow": [
      "Bash(*)",
      "Read(*)"
    ],
    "deny": [
      "Bash(rm -rf /)"
    ]
  }
}
```

### CLAUDE.md (per-project)

**Symlink mode** (template only, no prompts):
```
CLAUDE.md -> /Users/me/templates/standard-claude.md
```

**Composed mode** (template + prompts):
```markdown
<!-- managed by hystak -->

# Project Instructions

(template content here...)

---

## Security Rules

(prompt fragment: security-rules, order 10)

---

## Style Guide

(prompt fragment: style-guide, order 20)
```

### Skills (per-project)

Deployed as symlinks under `<project>/.claude/skills/`:

```
.claude/skills/
├── code-review/
│   └── SKILL.md -> /Users/me/skills/code-review/SKILL.md
└── commit/
    └── SKILL.md -> /Users/me/skills/commit/SKILL.md
```

---

## Entity Relationship Diagram

```
┌──────────────┐     ┌──────────────┐
│   registry   │     │   projects   │
│──────────────│     │──────────────│
│ mcps{}       │◄────│ managed_mcps │
│ skills{}     │     │ active_prof  │──┐
│ hooks{}      │     │ path         │  │
│ permissions{}│     └──────────────┘  │
│ templates{}  │                       │
│ prompts{}    │     ┌──────────────┐  │
│ tags{}       │◄────│  profiles/   │◄─┘
└──────────────┘     │──────────────│
       ▲             │ mcps[]       │  (bare string or map with overrides)
       │             │ skills[]     │
       └─────────────│ hooks[]      │
    all names must   │ permissions[]│
    exist in registry│ template     │
                     │ prompts[]    │
                     │ env{}        │
                     │ isolation    │
                     └──────────────┘

Tags expand to MCP names during sync:
  tag.members[] ──resolve──► mcp names ──deduplicate──► final server list
```

---

## Validation Rules

| Rule | Scope | Severity |
|------|-------|----------|
| MCP name in profile must exist in `registry.mcps` | Sync | Error |
| Skill name in profile must exist in `registry.skills` | Sync | Error |
| Skill source path must exist on disk | Sync | Error |
| Hook name in profile must exist in `registry.hooks` | Sync | Error |
| Permission name in profile must exist in `registry.permissions` | Sync | Error |
| Template name in profile must exist in `registry.templates` | Sync | Error |
| Prompt name in profile must exist in `registry.prompts` | Sync | Error |
| Tag member must exist in `registry.mcps` | Sync | Error |
| No circular tag references | Doctor | Error |
| Profile `active_profile` must reference existing profile file | Sync | Error |
| `managed_mcps` entries should exist in active profile or be scheduled for removal | Doctor | Warning |
| Skill source path should be absolute | Doctor | Warning |
| Env var interpolation `${VAR}` should reference set variables | Doctor | Warning |
