# hystak

[![CI](https://github.com/lcrostarosa/hystak/actions/workflows/ci.yml/badge.svg)](https://github.com/lcrostarosa/hystak/actions/workflows/ci.yml)
[![Release](https://github.com/lcrostarosa/hystak/actions/workflows/release.yml/badge.svg)](https://github.com/lcrostarosa/hystak/actions/workflows/release.yml)
[![Latest Release](https://img.shields.io/github/v/release/lcrostarosa/hystak)](https://github.com/lcrostarosa/hystak/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/lcrostarosa/hystak)](https://goreportcard.com/report/github.com/lcrostarosa/hystak)
[![Go Version](https://img.shields.io/github/go-mod/go-version/lcrostarosa/hystak)](https://github.com/lcrostarosa/hystak/blob/master/go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A CLI/TUI tool that manages MCP server configurations, skills, hooks, and permissions from a central registry — then syncs and launches Claude Code with the right profile. Single binary, zero runtime dependencies.

## Problem

MCP server configurations are scattered across client-specific config files. Keeping them consistent, reusing definitions across projects, and tracking what's deployed where is tedious and error-prone. Switching between projects means manually editing configs every time.

## Solution

hystak maintains a **central registry** of MCP server definitions and a **project store** that tracks which servers, skills, hooks, and permissions go where. It deploys to client configs, translating formats as needed, detects drift, and launches Claude Code with the right setup — all from a single command.

```
~/.hystak/registry.yaml   ← source of truth for all server definitions
~/.hystak/projects.yaml   ← projects, profiles, and assignments
```

### Key Features

- **Registry** — single catalog of all your MCP server definitions
- **Projects & Profiles** — assign servers to projects, create named profiles with different configurations
- **Per-project overrides** — customize env vars, args, or commands per project without duplicating definitions
- **Tags** — named server groups for bulk assignment (e.g., `core: [github, filesystem]`)
- **Skills, Hooks & Permissions** — manage Claude Code skills, hooks, and permission rules per profile
- **Sync** — deploy from registry to client config files in one command
- **Drift detection** — semantic comparison on launch; see exactly what changed with unified diffs
- **Import** — pull existing servers from client configs into the registry with conflict resolution
- **Discovery** — auto-detect MCP servers, skills, and hooks from a project directory
- **Backup & Restore** — snapshot and restore client configs per project or globally
- **Launch** — sync configs and launch Claude Code in one step, with optional worktree isolation
- **Profile sharing** — export and import profiles for team collaboration
- **Setup wizard** — guided first-run experience that imports existing configs and creates your first profile
- **Multi-client** — extensible deployer interface (v1: Claude Code; Claude Desktop and Cursor planned)

## Install

```bash
go install github.com/lcrostarosa/hystak@latest
```

## Quick Start

```bash
# First run — the setup wizard walks you through importing existing configs
hystak

# Or import an existing Claude Code config manually
hystak import ~/.claude.json

# Sync a project and launch Claude Code
hystak run my-project

# Open the management TUI
hystak manage
```

## Usage

### Interactive Launcher

Run `hystak` with no arguments to launch the interactive profile picker:

```bash
hystak
```

On first run, the setup wizard guides you through importing existing MCP configs and creating your first profile. On subsequent runs, pick a profile and launch directly.

Arguments after `--` are forwarded to the Claude Code process:

```bash
hystak -- --model sonnet
```

### TUI

The management TUI provides full CRUD for all configuration types:

```bash
hystak manage
```

Navigate between Profiles, MCPs, Skills, Hooks, Permissions, and Templates tabs. Supports form editing, conflict resolution, diff viewing, and discovery.

### CLI

```bash
hystak list                          # list registry servers
hystak sync <project>                # deploy servers for a project
hystak sync --all                    # deploy servers for all projects
hystak import <path>                 # import servers from a client config file
hystak override <project> <server>   # set per-project overrides
hystak diff <project>                # show drift as unified diff
hystak run <project> [client]        # sync and launch a client in the project directory
hystak run <project> --profile dev   # launch with a specific profile
hystak backup <project>              # back up client configs for a project
hystak backup --all                  # back up all projects
hystak backup --list [project]       # list backups (all or per-project)
hystak restore <project>             # interactively restore a backup
hystak restore <project> --index 0   # restore most recent (non-interactive)
hystak restore --global              # restore a global-scope backup
hystak setup                         # run the setup wizard
hystak manage                        # open the management TUI
hystak profile list                  # list all profiles
hystak profile export <file>         # export profiles to a file
hystak profile import <file>         # import profiles from a file
hystak version                       # print version info
hystak --configure <project>         # open the launch wizard for a project
```

## Configuration

hystak stores its config in `~/.hystak/` (override with `HYSTAK_CONFIG_DIR` or `--config-dir`).

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
    profiles:
      default:
        description: "Standard dev profile"
        mcps: [github, qdrant]
        skills: [go-code-reviewer]
        hooks:
          pre-tool-use: ["echo running"]
        permissions:
          - tool: Bash
            allow: true
```

Servers are resolved by expanding tags, collecting individual assignments, deduplicating, and applying overrides via shallow merge.

## Architecture

```
main.go → cli/ (Cobra root dispatcher)
              ├── no args + terminal → picker TUI → launch
              ├── manage             → full management TUI
              └── subcommand         → cli/ handlers
                       ↓
                  service/ (all business logic)
                       ↓
            ┌──────────┼──────────┐
         registry/   project/   deploy/
         (YAML)      (YAML)     (Deployer interface)
```

- **`internal/model/`** — Domain types: `ServerDef`, `ServerOverride`, `Project`, `MCPAssignment`, `Transport`, `ClientType`, `Skill`, `Hook`, `Permission`, `Template`
- **`internal/registry/`** — Server catalog CRUD and tag group management
- **`internal/project/`** — Project store with server assignment and resolution (tag expansion, dedup, override merge)
- **`internal/deploy/`** — `Deployer` interface with client-specific implementations. Translates hystak's canonical types to each client's expected config schema
- **`internal/service/`** — Orchestration layer consumed by both CLI and TUI: sync, drift detection, import, diff, backup, profiles
- **`internal/cli/`** — Cobra command tree with profile picker and launch integration
- **`internal/tui/`** — Bubble Tea app with tab navigation, mode-based overlays, launch wizard, and discovery
- **`internal/config/`** — Config directory paths and migration from legacy XDG layout
- **`internal/profile/`** — Profile data structures for export/import
- **`internal/backup/`** — Backup engine for client config snapshots
- **`internal/discovery/`** — Auto-detection of MCP servers, skills, and hooks in project directories
- **`internal/isolation/`** — Worktree isolation and lock-based concurrency for safe parallel launches
- **`internal/launch/`** — Client process launcher with platform-specific exec
- **`internal/catalog/`** — Built-in server catalog for the setup wizard

## Development

### Build & Test

```bash
make build               # build binary
make test                # run all tests
make test-cover          # run tests with coverage report
make cover-html          # generate HTML coverage report
make lint                # run golangci-lint
make snapshot            # build snapshot with goreleaser
```

Or with plain Go:

```bash
go build -o hystak .
go test ./...
```

## Tech Stack

Go, [Cobra](https://github.com/spf13/cobra), [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Bubbles](https://github.com/charmbracelet/bubbles) + [Lip Gloss](https://github.com/charmbracelet/lipgloss), [yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3)

## License

[MIT](LICENSE)
