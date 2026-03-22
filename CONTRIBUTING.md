# Contributing to hystak

## Prerequisites

- Go 1.25+ (see `go.mod` for exact version)
- [golangci-lint](https://golangci-lint.run/welcome/install/) for linting
- [VHS](https://github.com/charmbracelet/vhs) + ffmpeg for e2e tests (optional)

## Quick start

```bash
git clone https://github.com/lcrostarosa/hystak.git
cd hystak
make build
make test
make lint
```

## Development workflow

### Build

```bash
make build          # build binary with version info
go run .            # run without building
```

### Tests

```bash
make test           # unit tests
make test-race      # unit tests with race detector (CI runs this)
make lint           # golangci-lint (CI runs this)
make e2e            # e2e tests using VHS tape scripts
make test-cover     # tests with coverage summary
make cover-html     # generate HTML coverage report
```

All four checks (test, lint, build, e2e) must pass before a release is cut.
The release workflow will not run unless CI is fully green.

### Running a single test

```bash
go test ./internal/registry/ -run TestAdd
```

### Updating golden files

```bash
make test-update    # update TUI golden snapshots
make e2e-update     # regenerate e2e golden files
```

## Code style

- **errcheck**: All error return values must be handled. Use `_, _ = fmt.Fprintf(...)` for CLI output where errors are inconsequential. Use `_ = os.Remove(...)` for best-effort cleanup.
- **No unused code**: Remove unused variables, fields, and imports. The linter enforces this.
- **Business logic in `service/`**: CLI and TUI are thin layers. All logic goes through `internal/service/`.

## Project structure

```
internal/
  model/       Domain types (ServerDef, Project, etc.)
  registry/    Server catalog CRUD
  project/     Project store + resolution algorithm
  deploy/      Client-specific deployers (Claude Code, etc.)
  service/     Orchestration layer (sync, drift, import)
  cli/         Cobra command tree
  tui/         Bubble Tea UI
  config/      XDG config paths
```

## Submitting changes

1. Fork the repo and create a feature branch from `master`.
2. Make your changes. Add tests for new functionality.
3. Ensure `make test-race && make lint` pass locally.
4. Open a pull request against `master`.

## Release process

Releases are automated. Every push to `master` that passes CI (test + lint + build + e2e) triggers:

1. Auto-tagging (patch bump by default)
2. GoReleaser builds binaries for all platforms
3. GitHub Release is created

Add `[skip release]` to a commit message to push without releasing.
