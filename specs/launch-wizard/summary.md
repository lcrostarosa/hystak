# Launch Wizard — Summary

## Artifacts

| File | Description |
|------|-------------|
| `specs/launch-wizard/rough-idea.md` | Original idea as provided |
| `specs/launch-wizard/requirements.md` | 15 Q&A pairs covering all requirements |
| `specs/launch-wizard/research/tui-architecture.md` | Existing TUI component model, wizard/picker patterns |
| `specs/launch-wizard/research/deployer-service-layer.md` | Deployer interface, sync pipeline, service API |
| `specs/launch-wizard/research/config-formats.md` | Claude Code config file schemas and locations |
| `specs/launch-wizard/research/process-management.md` | SIGTSTP/SIGCONT feasibility analysis |
| `specs/launch-wizard/design.md` | Full design document (standalone) |
| `specs/launch-wizard/plan.md` | 14-step implementation plan with tests and demos |

## Overview

The Launch Wizard transforms hystak into a complete Claude Code session launcher. Key capabilities:

- **Guided first-launch wizard** with sequential category walk-through
- **On-demand reconfiguration** via hub-style category navigation
- **Profiles** — named loadouts (global + project-scoped, shareable as YAML)
- **Filesystem discovery** — scan `~/.claude/` and project dirs to curate available tools
- **Symlink deploys** — clean managed/unmanaged distinction without marker files
- **Configurable isolation** — none, worktree, or lock for concurrent session support
- **v1 reconfiguration** — exit → reconfigure → relaunch with `--continue`
- **v2 mid-session suspension** — SIGTSTP/SIGCONT job control (deferred, high-risk)

## Suggested Next Steps

1. Review the design and plan with any stakeholders
2. Begin implementation starting with Step 1 (config directory migration)
3. Core end-to-end flow is demoable after Step 9
4. Steps 10-12 add isolation, reconfiguration loop, and sharing
5. Step 13 (v2 process manager) should be attempted after the wizard UX is validated
