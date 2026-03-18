# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is hystak

hystak is a Go CLI/TUI tool that manages MCP (Model Context Protocol) server configurations from a central registry and deploys them to MCP client config files. Single binary, zero runtime dependencies.

## Build & Test Commands

```bash
go build -o hystak .                  # build binary
go test ./...                         # run all tests
go test ./internal/registry/          # run tests for a single package
go test ./internal/registry/ -run TestAdd  # run a single test
go run .                              # launch TUI (if terminal) or show help
go run . list                         # run a CLI subcommand
```

No Makefile, linter, or CI yet — standard `go build`/`go test` workflow.

## Architecture

Layered architecture following the Glow pattern (Charmbracelet):

```
main.go → cli/ (Cobra root dispatcher)
              ├── no args + terminal → tui/ (Bubble Tea)
              └── subcommand → cli/ handlers
                       ↓
                  service/ (all business logic)
                       ↓
            ┌──────────┼──────────┐
         registry/   project/   deploy/
         (YAML)      (YAML)     (Deployer interface)
```

- **`internal/model/`** — Domain types shared by all packages: `ServerDef`, `ServerOverride`, `Project`, `MCPAssignment`, `Transport`, `ClientType`, `DriftStatus`. `MCPAssignment` has custom YAML marshal/unmarshal for dual format (bare string or map with overrides).
- **`internal/registry/`** — Server catalog CRUD and tag group management. Reads/writes `~/.config/hystak/registry.yaml`.
- **`internal/project/`** — Project store with server assignment and the resolution algorithm (tag expansion → dedup → override merge). Reads/writes `~/.config/hystak/projects.yaml`.
- **`internal/deploy/`** — `Deployer` interface with client-specific implementations. v1 has Claude Code only (`claude_code.go`); `claude_desktop.go` and `cursor.go` are stubs. Each deployer translates hystak's canonical `ServerDef` to the client's expected JSON schema.
- **`internal/service/`** — Orchestration: sync, drift detection, import, diff. Both CLI and TUI consume this layer. All business logic lives here, not in CLI/TUI.
- **`internal/cli/`** — Cobra command tree. `PersistentPreRunE` initializes the shared `service.Service`. Root command dispatches to TUI when stdout is a terminal.
- **`internal/tui/`** — Bubble Tea app with tab navigation (Servers/Projects), mode-based overlays (Form, Confirm, Diff, Import). Child-to-parent communication via custom `tea.Msg` types.
- **`internal/config/`** — XDG-compliant config paths.

## Key Design Decisions

- **Client config schemas differ** — env var syntax (`${VAR}` vs `${env:VAR}`), field presence (`type`), and extra fields (`disabled`, `autoApprove`) vary per client. Deployers must translate; there is no universal schema.
- **Claude Code config locations** — `.mcp.json` (project scope) and `~/.claude.json` (global scope). NOT `settings.local.json`.
- **Override merge rules** — `env`/`headers`: map merge (override keys win). `args`: replace entirely. `command`/`url`: replace if non-nil.
- **Unmanaged servers** — servers in client configs but not in hystak are preserved during sync and flagged in the UI.
- **Fail fast** — malformed configs and dangling references halt with errors, never silently degrade.

## Implementation Status

Steps 1-8 of 14 are complete (scaffolding through TUI root). Steps 9+ (TUI tab content, form overlays, goreleaser) are pending. See `specs/hystak/plan.md` for the full plan.
