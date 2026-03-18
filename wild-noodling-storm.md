# hystak — Lightweight MCP Server Configuration Manager

## Context

MCP server configurations are scattered across client-specific config files (Claude Code, Claude Desktop, Cursor, etc.), making it difficult to maintain consistency, reuse server definitions, and track what's deployed where. ToolHive solves this at enterprise scale with Docker/K8s runtimes, gateways, and portals — but that's overkill for personal/small-team use.

**hystak** is a lightweight alternative: a central registry of MCP server definitions with a TUI for managing and deploying them across projects and MCP clients. One place to configure servers, then push them wherever needed.

## Tech Stack

- **Go** — single binary, fast startup, matches ToolHive's language
- **Bubble Tea** — Elm-architecture TUI framework (`github.com/charmbracelet/bubbletea`)
- **Bubbles** — reusable TUI components: list, table, textinput, viewport (`github.com/charmbracelet/bubbles`)
- **Lip Gloss** — terminal styling (`github.com/charmbracelet/lipgloss`)
- **gopkg.in/yaml.v3** — registry/project config persistence
- **encoding/json** — reading/writing client config files

## Architecture

### Config Storage

```
~/.config/hystak/
├── registry.yaml      # Central server catalog (source of truth)
└── projects.yaml      # Known projects + their server assignments + client targets
```

### Project Structure

```
hystak/
├── go.mod
├── go.sum
├── main.go                         # Entry point
├── internal/
│   ├── config/
│   │   └── paths.go                # XDG paths, constants
│   ├── model/
│   │   ├── server.go               # ServerDef struct
│   │   ├── project.go              # Project struct
│   │   └── client.go               # ClientType enum + config path logic
│   ├── registry/
│   │   └── registry.go             # Server CRUD, YAML load/save
│   ├── project/
│   │   └── project.go              # Project CRUD, server assignments
│   ├── deployer/
│   │   ├── deployer.go             # Deployer interface + factory
│   │   ├── claude_code.go          # .claude/settings.local.json writer
│   │   ├── claude_desktop.go       # claude_desktop_config.json writer
│   │   └── cursor.go               # .cursor/mcp.json writer
│   └── tui/
│       ├── app.go                  # Root model (tab switching, global keys)
│       ├── servers.go              # Servers tab model + view
│       ├── projects.go             # Projects tab model + view
│       ├── form.go                 # Server add/edit form model
│       ├── styles.go               # Lip Gloss style definitions
│       └── keys.go                 # Key map definitions
```

### Multi-Client Deployer

Each client type implements a `Deployer` interface:

```go
type Deployer interface {
    // ConfigPath returns the config file path for this client+project
    ConfigPath(projectPath string) string
    // ReadServers reads current MCP server configs from the client config
    ReadServers(projectPath string) (map[string]ServerConfig, error)
    // WriteServers writes MCP servers, preserving all other settings in the file
    WriteServers(projectPath string, servers map[string]ServerConfig) error
}
```

**Supported clients:**

| Client | Config Location | MCP Key |
|--------|----------------|---------|
| Claude Code (project) | `<project>/.claude/settings.local.json` | `mcpServers` |
| Claude Code (global) | `~/.claude/settings.json` | `mcpServers` |
| Claude Desktop | `~/Library/Application Support/Claude/claude_desktop_config.json` | `mcpServers` |
| Cursor | `<project>/.cursor/mcp.json` | `mcpServers` |

All clients use the same `mcpServers` key structure, so the server definition format is universal — only the file path differs.

## Data Models

### Server Definition (`registry.yaml`)

```yaml
servers:
  github:
    description: "GitHub API integration"
    transport: stdio
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN}"
    tags: [dev, vcs]

  filesystem:
    description: "Local filesystem access"
    transport: stdio
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/"]
    env: {}
    tags: [core]

  qdrant:
    description: "Qdrant vector database"
    transport: stdio
    command: uvx
    args: ["mcp-server-qdrant"]
    env:
      QDRANT_URL: "http://localhost:6333"
    tags: [data, vector]

  # SSE/HTTP transport example
  remote-api:
    description: "Remote API server"
    transport: sse
    url: "https://mcp.example.com/sse"
    headers:
      Authorization: "Bearer ${API_TOKEN}"
    tags: [remote]
```

### Project Registry (`projects.yaml`)

```yaml
projects:
  agents:
    path: /Volumes/Secondary/workspace/agents
    servers: [github, filesystem, qdrant]
    clients: [claude-code]              # Which clients to deploy to

  hystak:
    path: /Volumes/Secondary/workspace/hystak
    servers: [github, filesystem]
    clients: [claude-code, cursor]      # Deploy to both

  global:                               # Special: global/non-project configs
    path: "~"
    servers: [github]
    clients: [claude-desktop]           # Deploy to Claude Desktop global config
```

## TUI Layout

Bubble Tea Elm architecture with tab-based navigation:

```
┌─ hystak ─ MCP Server Manager ───────────────────────────┐
│  tab← [Servers]  [Projects] →tab              q quit    │
├──────────────────────────────────────────────────────────┤
│                                                          │
│  SERVERS TAB                                             │
│  ┌─ Registry ──────┐  ┌─ Details ─────────────────────┐ │
│  │ ▸ github     ⌂3 │  │ Name:      github             │ │
│  │   filesystem ⌂2 │  │ Transport: stdio              │ │
│  │   qdrant     ⌂1 │  │ Command:   npx                │ │
│  │   remote-api ⌂0 │  │ Args:      -y @mcp/server-gh  │ │
│  │                  │  │ Env:       GITHUB_TOKEN=${..}  │ │
│  │                  │  │ Tags:      dev, vcs            │ │
│  │                  │  │                                │ │
│  │                  │  │ Projects:  agents, hystak      │ │
│  └──────────────────┘  └────────────────────────────────┘ │
│  a add  e edit  d delete  i import  / filter             │
│                                                          │
│  PROJECTS TAB                                            │
│  ┌─ Projects ───────┐  ┌─ Servers (toggle) ───────────┐ │
│  │ ▸ agents      3↗ │  │ [x] github       ● synced    │ │
│  │   hystak      2↗ │  │ [x] filesystem   ● synced    │ │
│  │   global      1↗ │  │ [x] qdrant       ◐ drift     │ │
│  │                   │  │ [ ] remote-api                │ │
│  │                   │  ├─ Clients ────────────────────┤ │
│  │                   │  │ [x] claude-code              │ │
│  │                   │  │ [ ] cursor                   │ │
│  │                   │  │ [ ] claude-desktop            │ │
│  └───────────────────┘  └──────────────────────────────┘ │
│  a add project  s sync  S sync all  D diff               │
└──────────────────────────────────────────────────────────┘
```

## Implementation Steps

### Step 1: Go project scaffolding
- `go mod init` with Bubble Tea, Bubbles, Lip Gloss, yaml.v3 dependencies
- `main.go` entry point, `internal/config/paths.go` for XDG config paths
- Ensure `~/.config/hystak/` is created on first run with empty registry/projects

### Step 2: Data models + persistence
- `ServerDef` struct: Name, Description, Transport, Command, Args, Env, Tags, URL, Headers
- `Project` struct: Name, Path, Servers ([]string), Clients ([]ClientType)
- `ClientType` string enum: `claude-code`, `claude-desktop`, `cursor`
- YAML marshal/unmarshal with `yaml.v3`

### Step 3: Registry service
- `Load() (*Registry, error)` — read `registry.yaml`
- `Save() error` — write `registry.yaml`
- `Add(server ServerDef)`, `Update(name string, server ServerDef)`, `Delete(name string)`
- `Get(name string) (ServerDef, bool)`, `List() []ServerDef`
- `Import(configPath string) ([]ServerDef, error)` — parse a client config file and extract servers

### Step 4: Project service
- `Load() (*ProjectStore, error)` — read `projects.yaml`
- `Save() error` — write `projects.yaml`
- `Add/Remove/Get/List` for projects
- `Assign(project, server string)`, `Unassign(project, server string)`
- `SetClients(project string, clients []ClientType)`

### Step 5: Deployer
- `Deployer` interface with `ConfigPath`, `ReadServers`, `WriteServers`
- `ClaudeCodeDeployer` — reads/writes `.claude/settings.local.json`, merges only `mcpServers`
- `ClaudeDesktopDeployer` — reads/writes `claude_desktop_config.json`
- `CursorDeployer` — reads/writes `.cursor/mcp.json`
- `SyncProject(project, registry, deployers)` — deploy assigned servers to all target clients
- `DriftStatus(project, registry, deployers)` — compare deployed vs registry, return per-server status

### Step 6: TUI — Root app model
- Tab switching (Servers / Projects) with left/right arrows or tab key
- Global key bindings: `q` quit, `?` help, `tab`/`shift+tab` switch tabs
- Status bar at bottom with contextual help

### Step 7: TUI — Servers tab
- Left pane: `list.Model` from Bubbles showing server names + project count
- Right pane: rendered detail view of selected server (Lip Gloss styled)
- Key bindings: `a` add (enters form mode), `e` edit, `d` delete (with confirmation), `i` import, `/` filter

### Step 8: TUI — Projects tab
- Left pane: `list.Model` showing project names + server count
- Right pane: checkbox list of all registry servers (toggling assigns/unassigns)
- Below checkboxes: client target toggles
- Sync status indicator per server: `●` synced, `◐` drift, `○` not deployed
- Key bindings: `a` add project (path input), `s` sync selected, `S` sync all, `D` show diff

### Step 9: TUI — Server form
- Full-screen overlay for add/edit with `textinput.Model` fields
- Fields: name, description, transport (select), command, args, env (key=value pairs), tags
- `enter` to save, `esc` to cancel
- Validation: name required + unique, command required for stdio transport, url required for sse/http

### Step 10: Import flow
- `i` on Servers tab opens file picker (text input for path)
- Parses the target config file, extracts `mcpServers` entries
- Shows preview of servers to import with checkboxes
- Selected servers added to registry, source directory registered as project

## Key Design Decisions

1. **hystak owns `mcpServers`** — it's the source of truth. Manual edits to `mcpServers` in client configs are detected as drift.
2. **Other settings preserved** — permissions, hooks, and everything else in client config files are untouched. Only `mcpServers` is managed.
3. **Env vars stored as references** (`${VAR_NAME}`) — actual values come from the runtime environment. No secrets in hystak config.
4. **No server lifecycle management in v1** — config management only. Process start/stop can come in v2.
5. **Single `registry.yaml`** for simplicity — split to per-server files later if needed.
6. **Universal server format** — all supported clients use the same `mcpServers` schema, so one definition works everywhere.
7. **Project-level client targeting** — each project specifies which clients receive its config (e.g., deploy to both Claude Code and Cursor).

## Verification

1. `go run .` or `go build && ./hystak` launches TUI
2. Add a server via Servers tab → verify `~/.config/hystak/registry.yaml` updated
3. Add a project, assign servers, select clients → verify `projects.yaml` updated
4. Sync → verify target config files have correct `mcpServers` and other settings preserved
5. Manually edit a deployed config → drift detected on Projects tab (`◐` indicator)
6. Import from existing config → servers added to registry, project registered
7. Deploy same server to multiple clients (Claude Code + Cursor) → both config files updated correctly
