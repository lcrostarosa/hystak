# hystak — Lightweight MCP Server Configuration Manager

## Objective

Build a Go CLI/TUI tool that manages MCP server configurations from a central registry and deploys them to MCP client config files. Single binary, zero runtime dependencies.

## Key Requirements

- **Central registry** (`~/.config/hystak/registry.yaml`) — source of truth for all MCP server definitions
- **Project store** (`~/.config/hystak/projects.yaml`) — tracks which servers are assigned to which projects, with per-project overrides
- **Tags** — named server groups in `registry.yaml` for bulk assignment to projects
- **Per-project overrides** — shallow merge of env/args/command/url/headers on top of registry defaults
- **Claude Code deployer** (v1) — writes to `.mcp.json` (project) and `~/.claude.json` (global), preserving all non-mcpServers keys
- **Deployer interface** — extensible to Claude Desktop, Cursor, and future clients
- **Import** — parse existing client config, extract servers into registry, prompt on naming conflicts
- **Drift detection** — semantic comparison on app launch (command, args, env, url, headers); unified diff view with sync-to-resolve
- **Unmanaged servers** — preserve, flag, and prompt user to address
- **CLI + TUI** — `hystak` launches Bubble Tea TUI; subcommands (`list`, `sync`, `import`, `override`, `diff`, `version`) run non-interactively
- **Error handling** — fail fast: halt on malformed configs or dangling references; bootstrap missing config files automatically
- **Distribution** — goreleaser with cross-platform builds and GitHub Actions release workflow

## Tech Stack

- Go, Cobra (CLI), Bubble Tea + Bubbles + Lip Gloss (TUI), gopkg.in/yaml.v3, encoding/json, goreleaser

## Acceptance Criteria

- Given a server is added to the registry, then `registry.yaml` contains the entry
- Given a project has tags [core] and mcps with overrides, then resolved servers reflect tag expansion + override merge
- Given a sync is run, then the client config file contains the correct servers in the client's expected format
- Given a client config has an unmanaged server, then sync preserves it and flags it
- Given a deployed config differs from the registry, then drift is detected as "drifted" on launch
- Given a diff is viewed, then a unified diff is shown with an action to sync and resolve
- Given an import has naming conflicts, then the user is prompted per-conflict to keep/replace/rename
- Given a tag references a server, then deleting that server is blocked with an error

## Critical Design Notes

- Claude Code MCP configs live in `.mcp.json` and `~/.claude.json` — NOT `settings.local.json`
- Client schemas are NOT universal: env var syntax (`${VAR}` vs `${env:VAR}`), field presence (`type`), and extra fields (`disabled`, `autoApprove`) differ per client — deployers must translate
- `MCPAssignment` in `projects.yaml` supports dual YAML format: bare string (`- github`) or map with overrides (`- github: {overrides: {env: {KEY: val}}}`)

## Reference

All specs, research, data models, interfaces, and the 14-step implementation plan are in `specs/hystak/`:
- `design.md` — architecture, Go interfaces, data models, acceptance criteria, testing strategy
- `plan.md` — incremental implementation steps (scaffolding → types → registry → projects → deployer → service → CLI → TUI → CI)
- `research/` — Bubble Tea patterns, MCP config formats, competitive landscape, CLI+TUI coexistence, goreleaser
