# hystak — Revised PRD

## Overview

hystak is a TUI + CLI tool for managing MCP server configurations across multiple Claude Code projects. It maintains a central registry of servers, skills, hooks, permissions, templates, and prompts, then deploys selected subsets to individual projects via profiles.

**Core loop:** Registry → Profiles → Sync → Launch
---

## 1. First Run & Setup

### S-001: First-Run Detection
When `~/.hystak/` does not exist, hystak creates it with correct permissions and enters the first-run flow.

### S-002: Keybinding Prompt
On first run, a single inline prompt asks: `Navigation style? [A]rrows (recommended) / [V]im / [C]lassic`. Saved to `~/.hystak/keys.yaml`. No wizard step — just a prompt.

### S-003: Existing Config Scanning
After keybinding selection, hystak scans `~/.claude.json` and the current directory's `.mcp.json` for MCP servers. Discovered servers are shown as pre-checked candidates. User confirms import with Enter.

### S-004: First Project Registration
If the current directory is a project (contains `.git/`, `package.json`, `pyproject.toml`, or similar markers), hystak offers to register it. Profile name defaults to directory basename, path defaults to cwd.

### S-005: Direct to Launch Wizard
After import + project registration, hystak opens the Launch Wizard (S-060) in sequential mode so the user can select MCPs and launch immediately.

### S-006: Re-run Setup
`hystak setup` re-runs the first-run flow on demand, regardless of registry state. Useful for re-importing or adding catalog items.

---

## 2. Auto-Discovery & Import

### S-007: Silent Auto-Discovery
On any command, hystak scans `~/.claude.json` and the active project's `.mcp.json` for servers not in the registry. New servers are auto-imported silently. Controlled by `auto_sync` user setting.

### S-008: Non-Blocking Discovery
Discovery errors (unreadable files, malformed JSON) emit a warning line but never block execution.

### S-009: Manual Import Overlay
In the Registry tab (MCPs sub-view), pressing `I` opens an import overlay. User provides a file path. All discovered servers appear as a multi-select list.

### S-010: Import Conflict Resolution
When an imported server name collides with a registry entry, hystak shows per-server options:
- **Keep** existing
- **Replace** with imported
- **Rename** the import (prompts for new name)
- **Skip**

An "Apply to all" option resolves remaining conflicts with the same choice.

### S-011: Skill Discovery
In the Tools tab, "Discover" scans `<project>/.claude/skills/` for unregistered skill directories. Shows candidates with conflict detection for selective import.

---

## 3. Registry Management

The registry is the single source of truth for all managed resources. Each resource type lives in `~/.hystak/registry.yaml`.

### S-012: MCP Servers
**Add** (`A`): Form with Name, Transport (stdio/sse/http), Command/URL, Args, Env, Headers, Description.
**Edit** (`E`): Same form, pre-filled.
**Delete** (`D`): Confirmation prompt. Cascades: unassigns from all profiles. Blocked if referenced by a tag (error names the tag).
**Filter** (`/`): Real-time name filtering.
**Multi-select** (`Space` to toggle, then bulk action): Delete or assign to profile.

### S-013: Skills
Form fields: Name, Description, Source path (to SKILL.md on disk).

### S-014: Hooks
Form fields: Name, Event (PreToolUse / PostToolUse / Notification / Stop), Matcher pattern, Command, Timeout (seconds).

### S-015: Permission Rules
Form fields: Name, Rule pattern (e.g. `Bash(*)`), Type (allow / deny).

### S-016: Templates
Form fields: Name, Source path (to a CLAUDE.md template file).

### S-017: Prompt Fragments
Form fields: Name, Description, Source path, Category, Order (integer), Tags.
**Preview** (`P`): Shows composed CLAUDE.md with template + all selected fragments sorted by Order.

### S-018: Delete Cascading
Deleting any registry resource unassigns it from all profiles that reference it.

### S-019: List Servers (CLI)
`hystak list` prints a tab-separated table: NAME, TRANSPORT, COMMAND/URL.

---

## 4. Tag Management

### S-020: Create/Edit Tags
A tag is a named list of server names. Created/edited via the Registry tab or CLI (`hystak tag create <name> <server1> <server2> ...`).

### S-021: Tag Expansion During Sync
When a profile references a tag, sync expands it to member server names, deduplicates against individual assignments, and deploys all resolved servers.

### S-022: Dangling Tag References
If a tag references a server not in the registry, sync fails with an error naming the tag and missing server.

---

## 5. Project Management

### S-023: Add Project (TUI)
Running `hystak` from an unregistered directory triggers the first-project flow: register the directory, create a default profile, open the Launch Wizard.

### S-024: Delete Project (TUI)
`D` on a project in the Projects tab. Confirmation prompt. Removes the project and all its profiles/assignments.

### S-025: Assign Resources to Profile
Selecting a project in the Projects tab shows the right pane with 6 sections (MCPs, Skills, Hooks, Permissions, Templates, Prompts). Each section shows toggleable items from the registry.
---

## 6. Profiles

### S-027: Multiple Profiles Per Project
Each profile has its own: MCPs, skills, hooks, permissions, prompts, env vars, CLAUDE.md template, isolation strategy.

### S-028: Set Active Profile
One profile per project is active at a time. Sync and launch use the active profile.

### S-029: Empty Profile
The built-in "empty" profile deploys zero configuration. Useful for launching Claude Code in a clean state.

### S-030: Export Profile (CLI)
`hystak profile export <name>` serializes to YAML on stdout. `-o <file>` writes to a file.

### S-031: Import Profile (CLI)
`hystak profile import <file>` imports as a global profile. `--as <name>` renames on import.

### S-032: List Profiles (CLI)
`hystak profile list` shows NAME, SCOPE, DESCRIPTION. `--project <name>` filters to project-scoped profiles.

---

## 7. Sync & Deploy

### S-033: Sync Single Project
`hystak sync <project>` resolves the active profile, looks up all referenced resources, applies overrides, deploys to client configs, prints per-server results (added / updated / unchanged / unmanaged).

### S-034: Sync All
`hystak sync --all` syncs every project in order.

### S-035: Sync with Specific Profile
`hystak sync <project> --profile <name>` uses the named profile without changing which is active.

### S-036: Sync Dry-Run
`hystak sync <project> --dry-run` shows what would be written without touching disk.

### S-037: Override Merge
Per-project overrides on a server are merged during sync:
- `env` / `headers`: map-merge (override keys win)
- `args`: full replacement
- `command` / `url`: replaced if set

### S-038: Preserve Unmanaged Servers
Servers in `.mcp.json` not managed by hystak are preserved untouched during sync.

### S-039: Remove Deactivated Servers
Servers removed from the active profile are removed from client config if they were previously deployed by hystak (tracked in `managed_mcps`).

### S-040: Automatic Backup
Before writing, hystak backs up current config (unless `backup_policy: never`).

### S-041: Missing Server Error
Referencing a server not in the registry fails sync with a clear error. No partial writes.

### S-042: Deploy MCP Servers
Written to `<project>/.mcp.json` in Claude Code format. Non-`mcpServers` keys preserved. Global scope writes to `~/.claude.json`.

### S-043: Deploy Skills as Symlinks
Each skill deployed as symlink: `<project>/.claude/skills/<name>/SKILL.md` → source file. Stale symlinks removed on sync.

### S-044: Deploy Hooks & Permissions
Written to `<project>/.claude/settings.local.json`. Hooks grouped by event type, permissions split into allow/deny arrays.

### S-045: Deploy CLAUDE.md
- Template only → symlink to source
- Template + prompts → generated file with `<!-- managed by hystak -->` sentinel, template content + fragments sorted by Order
- User-owned CLAUDE.md (no sentinel) → never overwritten, reported as preflight conflict

---

## 8. Preflight & Conflict Resolution

### S-046: Conflict Detection
Before sync writes, hystak checks for user-owned files (not symlinks, no managed sentinel) at target paths. Detected conflicts are presented before writing.

### S-047: Conflict Resolution Overlay
Each conflict shows options: **Keep existing**, **Replace**, **Skip**. "Apply to all" available. Canceling mid-resolution aborts sync.

### S-048: Symlinks Are Not Conflicts
Existing symlinks (previously deployed by hystak) are updated silently.

---

## 9. Drift Detection

### S-049: Show Drift (CLI)
`hystak diff <project>` shows a unified diff between deployed and expected configs. `hystak diff --all` shows per-project status.

### S-050: No Drift
When configs match: `No drift detected.`

### S-051: Drift Overlay (TUI)
Tools tab → "Diff". Servers colored by status (synced / drifted / missing / unmanaged). Press `s` to sync from the diff view.

### S-052: Semantic Comparison
Drift compares only deployment-relevant fields (transport, command, args, env, url, headers). Ignores metadata and JSON formatting.

---

## 10. Launch & Run

### S-053: Launch from TUI
`L` in Projects tab: set active profile → sync → launch Claude Code.

### S-054: Run from CLI
`hystak run <project>` syncs and launches. Post-exit loop: **R**elaunch / **C**onfigure / **Q**uit.

### S-055: Run with Profile
`hystak run <project> --profile <name>` activates and syncs the named profile before launch.

### S-056: Run without Sync
`hystak run <project> --no-sync` launches without syncing.

### S-057: Run Dry-Run
`hystak run <project> --dry-run` shows sync plan and launch command without executing.

### S-058: Explicit Client
`hystak run <project> <client>` launches the named client via `exec`. No post-exit loop (process replaced).

### S-059: Forward Args
`hystak run <project> -- --verbose` forwards extra args to the client.

---

## 11. Launch Wizard

### S-060: Sequential Mode (First Launch)
3 steps with progress indicator:

**Step 1 — MCPs** (primary value)
Multi-select list showing all registry servers + discovered servers. Popular/catalog items marked. Profile reference counts shown.

**Step 2 — Quick Options**
Single screen with 4 collapsible sections:
- Skills (toggle list)
- Permissions (toggle list)
- Hooks (toggle list, collapsed by default)
- CLAUDE.md template (dropdown, collapsed by default)

**Step 3 — Review & Launch**
Checklist with counts per category. Options: **Launch** (Enter), **Edit** (back to hub), **Cancel** (Esc).

### S-061: Hub Mode (Reconfiguration)
Sidebar menu with categories + selection counts. Direct jump to any category. Categories: MCPs, Skills, Permissions, Hooks, CLAUDE.md, Prompts, Env Vars, Isolation.

### S-062: Env Var Editor
Key-value editor. Ctrl+A to add, Ctrl+D to delete, Enter to edit.

### S-063: Isolation Strategy
Choose: **None** (shared config, single session), **Worktree** (git worktree per profile), **Lock** (shared config with mutex). Descriptions shown inline.

---

## 12. Isolation

### S-064: Worktree Isolation
Creates git worktree at `<project>.hystak-wt-<profile>/`. Reuses existing worktree on relaunch. Errors if not a git repo.

### S-065: Lock Isolation
Creates `.hystak.lock` with PID. Errors if lock held by running process. Auto-cleans stale locks (PID not running).

---

## 13. Backup & Restore

### S-066: Backup
`hystak backup <project>` backs up client configs to `~/.hystak/backups/` with timestamps. `--all` backs up every project.

### S-067: List Backups
`hystak backup --list <project>` shows TIMESTAMP, CLIENT, SCOPE, PATH table.

### S-068: Restore
`hystak restore <project>` shows interactive backup selection. `--index 0` restores most recent non-interactively. `--global` restores `~/.claude.json`. Confirmation prompt before restore.

### S-069: Undo Last Sync
`hystak undo [<project>]` restores from the most recent automatic backup. Shortcut for the common "I just synced and something broke" case.

### S-070: Backup Retention
Oldest backups pruned when count exceeds `max_backups` (default 10) per scope.

---

## 14. Per-Project Overrides

### S-071: Override Semantics
Overrides are per-server, per-project. Applied during sync without modifying the registry.
- `env` / `headers`: map-merge (override keys win)
- `args`: full replacement
- `command` / `url`: replaced if non-nil

### S-072: Dual YAML Format
`projects.yaml` MCPs support bare strings (`- github`) and maps (`- github: {overrides: {env: {KEY: val}}}`).

---

## 15. User Configuration

### S-073: Settings (`~/.hystak/user.yaml`)
| Key | Default | Description |
|-----|---------|-------------|
| `auto_sync` | `true` | Enable auto-discovery on startup |
| `backup_policy` | `always` | `always` / `never` |
| `max_backups` | `10` | Per-scope retention limit |

---

## 16. Validation & Health

### S-074: Doctor Command
`hystak doctor` validates the registry and all projects:
- Circular tag references
- Tags referencing missing servers
- Profiles referencing missing resources
- Skills with missing source files
- Orphaned managed_mcps entries

Prints per-issue severity (error / warning) with fix suggestions.

---

## 17. Terminal & UX

### S-075: Non-TTY Detection
Piped output or redirected stdout prints help text instead of launching TUI.

### S-076: JSON Output
`--json` flag on list/diff/doctor commands outputs machine-readable JSON.

### S-077: Quiet Mode
`--quiet` suppresses informational output, only printing errors.

### S-078: Custom Keybindings
`~/.hystak/keys.yaml` respected for all TUI interactions.

### S-079: Config Directory Override
`--config-dir <path>` redirects all config I/O to the specified directory.

### S-080: Version
`hystak version` prints version, git commit, build date.

### S-081: Shell Completions
`hystak completion <shell>` generates completions for bash, zsh, fish.

---

## 18. Error Handling & Edge Cases

### S-082: Malformed Config
Invalid YAML fails with parse error identifying file and line.

### S-083: Missing Config Directory
`~/.hystak/` auto-created with `0755` permissions.

### S-085: Bootstrap Client Config
Missing `.mcp.json` or `~/.claude.json` created with correct structure before first write.

### S-086: Missing Skill Source
Sync fails with error identifying the missing source path.