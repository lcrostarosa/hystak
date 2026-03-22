# hystak

Go CLI/TUI tool that manages MCP server configurations from a central registry and deploys them to Claude Code project configs.

## Build & Test

```bash
make build          # Build binary
make test           # go test -short ./...
make test-race      # go test -race ./...
make test-all       # go test ./... (includes integration)
make test-update    # UPDATE_GOLDEN=1 go test ./internal/tui/...
make test-cover     # Coverage report → coverage.out + HTML
make lint           # staticcheck + go vet
make e2e            # VHS tape E2E tests
```

## Project Structure

```
internal/
├── backup/      Config snapshot engine
├── catalog/     Built-in curated server/skill/hook catalog
├── cli/         Cobra command tree
├── config/      XDG paths, user config
├── deploy/      Deployer + ResourceDeployer implementations
├── discovery/   Auto-detection engine
├── errors/      Custom error types
├── isolation/   Worktree & lock-based concurrency
├── keyconfig/   Keybinding configuration
├── launch/      Client process execution (platform-specific)
├── model/       Domain types implementing Resource interface
├── profile/     Profile manager
├── project/     Project store with resolution algorithm
├── registry/    Generic Store[T] + Registry
├── service/     Orchestration layer (sync, diff, import, discover, backup)
└── tui/         Bubble Tea app
```

## Key Conventions

- **Go 1.25.6**, CGO disabled, 7 direct dependencies
- All writes to config files use `atomicWrite` (write-to-temp + fsync + rename)
- Never assign errors to `_` — use `t.Fatal` in tests, propagate in production
- Validate at write boundaries, trust internal invariants
- Fail fast — no defensive nil checks for impossible states
- `Store[T, PT]` generic serves all 6 resource types — don't duplicate CRUD
- `ResourceDeployer` interface for skills, settings, CLAUDE.md — extend by implementing, not modifying
- Bubble Tea: never do I/O in `Update` or constructors — use `tea.Cmd`
- Table-driven tests by default, `Test<Subject>_<Scenario>` naming
- Integration tests get `_Integration` suffix and `testing.Short()` guard

## Standards Docs

Read before writing code:

- `coding-standards.md` — Anti-patterns and rules for production code
- `testing-standards.md` — Anti-patterns and rules for test code
- `architecture.md` — Design patterns, package responsibilities, extensibility
- `data-model.md` — YAML/JSON schemas, file paths, validation rules
- `tui-wireframes.md` — TUI layout, overlays, color scheme
- `prd-revised.md` — Full feature specs (87 stories)
- `story-map.md` — Priority map (P0–P3) and milestones
