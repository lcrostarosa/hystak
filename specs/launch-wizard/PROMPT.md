# Launch Wizard for hystak

## Objective

Transform hystak from a configuration management tool into a complete Claude Code session launcher with a guided configuration wizard, profile system, filesystem discovery, and symlink-based deploys.

## Key Requirements

- Show wizard on first project launch (never launched before) and on demand via `--configure` flag or picker option
- Wizard covers all Claude Code config: MCPs, skills, permissions, hooks, CLAUDE.md, env vars, plugins
- First launch: sequential walk-through → hub → checklist confirm → launch
- On-demand: jump to hub → edit → checklist → launch
- Profiles: named loadouts (global in `~/.hystak/profiles/`, project-scoped), shareable as YAML, built-in "vanilla" empty profile
- Config ownership: `~/.hystak/` is source of truth, `~/.claude/` is read-only discovery, project dirs are deploy targets
- Symlink deploys for skills and CLAUDE.md; JSON merge for `.mcp.json` and `settings.local.json`
- Filesystem discovery scans `~/.claude/` (global) and project dirs for available items
- User-configurable isolation per project: none (default), worktree (concurrent sessions), lock
- Unmanaged configs preserved, shown in wizard with "adopt" option
- ASCII art boot logo on startup
- Merge wizard into existing management TUI (not a separate app)
- v1: exit Claude → reconfigure → relaunch with `--continue`
- v2 (deferred): SIGTSTP/SIGCONT mid-session suspension

## Acceptance Criteria

- Given a project with no active profile, when selected in picker, then wizard opens in sequential mode walking through all categories
- Given a project with active profile, when `--configure` used, then wizard opens in hub mode
- Given a profile with 3 MCPs and 2 skills enabled, when synced, then only those items deployed (symlinks for skills, JSON entries for MCPs)
- Given two sessions with worktree isolation, then each has independent configs in separate git worktrees
- Given vanilla profile selected, then all managed symlinks and entries removed, unmanaged preserved
- Given Claude exits via hystak, then user prompted to relaunch, reconfigure, or quit
- Given `hystak profile export frontend -o f.yaml` then valid YAML produced; `hystak profile import f.yaml` round-trips

## Reference

Full spec: `specs/launch-wizard/`
- `design.md` — architecture, components, interfaces, data models, error handling
- `plan.md` — 14-step incremental implementation plan with test requirements
- `research/` — TUI architecture, deployer layer, config formats, process management analysis
