# hystak

A lightweight MCP server configuration manager. Manage MCP server definitions from a central registry and deploy them to client config files — with per-project overrides, drift detection, and a TUI.

## Problem

MCP server configurations are scattered across client-specific config files (Claude Code, Claude Desktop, Cursor, etc.). Keeping them consistent, reusing definitions across projects, and tracking what's deployed where is tedious and error-prone.

## Solution

hystak maintains a **central registry** of MCP server definitions and a **project store** that tracks which servers go where. It deploys to client configs, translating formats as needed, and detects when deployed configs drift from the source of truth.

```
~/.config/hystak/registry.yaml   ← source of truth for all server definitions
~/.config/hystak/projects.yaml   ← which servers are assigned to which projects
```

### Key Features

- **Registry** — single catalog of all your MCP server definitions
- **Projects** — assign servers to projects individually or via tag groups
- **Per-project overrides** — customize env vars, args, or commands per project without duplicating definitions
- **Tags** — named server groups for bulk assignment (e.g., `core: [github, filesystem]`)
- **Sync** — deploy from registry to client config files in one command
- **Drift detection** — semantic comparison on launch; see exactly what changed with unified diffs
- **Import** — pull existing servers from client configs into the registry with conflict resolution
- **Multi-client** — extensible deployer interface (v1: Claude Code; Claude Desktop and Cursor planned)

## Install

```bash
go install github.com/lcrostarosa/hystak@latest
```

## Usage

### TUI

Run `hystak` with no arguments to launch the interactive TUI:

```bash
hystak
```

Navigate with Tab/arrow keys between Servers and Projects tabs. The TUI provides full CRUD, sync, import, and diff capabilities.

### CLI

```bash
hystak list                          # list registry servers
hystak sync <project>                # deploy servers for a project
hystak sync --all                    # deploy servers for all projects
hystak import <path>                 # import servers from a client config file
hystak override <project> <server>   # set per-project overrides
hystak diff <project>                # show drift as unified diff
hystak version                       # print version info
```

## Configuration

### Registry (`registry.yaml`)

```yaml
servers:
  github:
    description: "GitHub API integration"
    transport: stdio
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN}"

  qdrant:
    description: "Qdrant vector database"
    transport: stdio
    command: uvx
    args: ["mcp-server-qdrant"]
    env:
      QDRANT_URL: "${QDRANT_URL}"

tags:
  core: [github, filesystem]
```

### Projects (`projects.yaml`)

```yaml
projects:
  my-project:
    path: /path/to/my-project
    clients: [claude-code]
    tags: [core]
    mcps:
      - qdrant:
          overrides:
            env:
              COLLECTION_NAME: my-collection
```

Servers are resolved by expanding tags, collecting individual assignments, deduplicating, and applying overrides via shallow merge.

## Tech Stack

Go, [Cobra](https://github.com/spf13/cobra), [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Bubbles](https://github.com/charmbracelet/bubbles) + [Lip Gloss](https://github.com/charmbracelet/lipgloss), [yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3)

## License

TBD
