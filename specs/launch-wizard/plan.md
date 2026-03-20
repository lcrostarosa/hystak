# Launch Wizard — Implementation Plan

## Checklist

- [ ] Step 1: Config directory migration (~/.config/hystak → ~/.hystak)
- [ ] Step 2: Profile model and manager
- [ ] Step 3: Discovery engine
- [ ] Step 4: Symlink deployers
- [ ] Step 5: Profile-aware sync pipeline
- [ ] Step 6: Boot logo and picker updates
- [ ] Step 7: Launch wizard TUI — sequential mode
- [ ] Step 8: Launch wizard TUI — hub mode and checklist
- [ ] Step 9: CLI integration and launch flow
- [ ] Step 10: Worktree isolation
- [ ] Step 11: v1 reconfiguration loop (exit → reconfigure → relaunch)
- [ ] Step 12: Profile sharing (export/import)
- [ ] Step 13: v2 process manager (SIGTSTP/SIGCONT)
- [ ] Step 14: Polish and edge cases

---

## Step 1: Config Directory Migration

**Objective:** Move hystak's source of truth from `~/.config/hystak/` to `~/.hystak/` and establish the new directory structure.

**Implementation guidance:**
- Add migration logic to `internal/config/` that runs on startup.
- If `~/.config/hystak/` exists and `~/.hystak/` does not, copy `registry.yaml` and `projects.yaml` to the new location.
- If both exist, prefer `~/.hystak/` and warn the user about the old directory.
- Update `config.ConfigDir()` to return `~/.hystak/`.
- Create subdirectories: `profiles/`, `skills/`, `templates/`, `backups/`.
- Update all references to config paths throughout the codebase.

**Test requirements:**
- Migration from old path to new path succeeds.
- Fresh install creates `~/.hystak/` with correct subdirectories.
- Both-exist scenario warns but doesn't fail.
- All existing tests pass with updated config path.

**Integration notes:** This is a foundational change — every subsequent step depends on the new directory structure.

**Demo:** Run `hystak list` on a machine with existing `~/.config/hystak/` configs. Configs appear from `~/.hystak/` after automatic migration. Old directory is left intact with a deprecation notice printed.

---

## Step 2: Profile Model and Manager

**Objective:** Implement the `Profile` type and `Manager` for CRUD, persistence, and the vanilla built-in profile.

**Implementation guidance:**
- Create `internal/profile/` package.
- `Profile` struct: Name, Description, MCPs ([]string), Skills ([]string), Hooks ([]string), Permissions ([]string), EnvVars (map[string]string), ClaudeMD (string), Isolation (IsolationStrategy).
- `Manager` handles loading/saving profiles from `~/.hystak/profiles/` (global) and within project configs (project-scoped).
- Extend `model.Project` with: `Profiles map[string]Profile`, `ActiveProfile string`, `Launched bool`.
- Built-in `vanilla` profile: empty, cannot be deleted or modified.
- YAML serialization matching the format in the design doc.

**Test requirements:**
- CRUD round-trip: create, read, update, delete profiles.
- Vanilla profile always exists and returns empty selections.
- Project-scoped profiles stored correctly in project config.
- Profile validation: reject empty name, duplicate names.
- YAML marshal/unmarshal round-trip.

**Integration notes:** Profiles are consumed by the sync pipeline (Step 5) and the wizard TUI (Steps 7-8). This step only builds the data layer.

**Demo:** Programmatically create a "frontend" profile with 3 MCPs and 2 skills, save it, reload it, verify contents match.

---

## Step 3: Discovery Engine

**Objective:** Build a filesystem scanner that finds all available Claude Code configuration items from global and project scopes.

**Implementation guidance:**
- Create `internal/discovery/` package.
- `Engine` struct with `Scan(projectPath string) (*DiscoveredItems, error)`.
- Scan sources:
  - MCPs: parse `~/.claude.json` → `mcpServers`, parse `project/.mcp.json` → `mcpServers`, list from hystak registry.
  - Skills: glob `~/.claude/skills/*/SKILL.md`, glob `project/.claude/skills/*/SKILL.md`.
  - Hooks: parse `~/.claude/settings.json` → `hooks`, parse `project/.claude/settings.local.json` → `hooks`.
  - Permissions: parse same settings files → `permissions.allow` and `permissions.deny`.
  - Env vars: parse same settings files → `env`.
- Each discovered item tagged with `DiscoverySource` (Global, Project, Registry) and `IsManaged` (is it a symlink or tracked by hystak).
- For skills: `IsManaged = os.Readlink succeeds` (it's a symlink).
- For MCPs/settings: `IsManaged = name exists in project's active profile`.
- Graceful degradation: skip unreadable files, log warnings, never fail the whole scan.

**Test requirements:**
- Mock filesystem with various config layouts → verify correct discovery.
- Missing `~/.claude/` directory → empty results, no error.
- Malformed JSON in `.mcp.json` → warning logged, other sources still scanned.
- Symlink skill detected as managed.
- Regular file skill detected as unmanaged.
- Deduplication: same MCP in global + project → both shown with correct source.

**Integration notes:** The discovery engine is consumed by the wizard TUI (Steps 7-8) and the existing first-time wizard can be updated to use it.

**Demo:** Seed `~/.claude/` with 3 MCPs and 2 skills, a project with 1 MCP and 1 skill. Run discovery. Output shows 4 MCPs (3 global, 1 project) and 3 skills (2 global, 1 project) with correct sources.

---

## Step 4: Symlink Deployers

**Objective:** Replace file-copy and sentinel-based deployers with symlink-based deploys for skills and CLAUDE.md.

**Implementation guidance:**
- Modify `internal/deploy/skills.go`:
  - `SyncSkills` creates symlinks: `project/.claude/skills/<name>/SKILL.md` → `~/.hystak/skills/<name>/SKILL.md`.
  - Before creating symlink, ensure source exists in `~/.hystak/skills/`.
  - Remove old symlinks not in current profile (check with `os.Readlink`).
  - Leave non-symlink skill directories untouched (unmanaged).
  - Remove `.hystak-managed` marker file support (migration: if marker exists, read it, create equivalent symlinks, delete marker).
- Modify `internal/deploy/claude_md.go`:
  - `SyncClaudeMD` creates symlink: `project/CLAUDE.md` → `~/.hystak/templates/<name>.md`.
  - Only create symlink if no `CLAUDE.md` exists, or existing `CLAUDE.md` is already a symlink (managed).
  - Regular file = user-owned, never overwrite.
  - Remove `<!-- managed by hystak -->` sentinel support (migration: if sentinel file exists and template matches, replace with symlink).
- Add `IsManaged(path string) bool` helper: returns true if path is a symlink.

**Test requirements:**
- Skill deploy creates symlinks, not file copies.
- Skill removal only removes symlinks, not regular files.
- CLAUDE.md deploy creates symlink when no file exists.
- CLAUDE.md deploy replaces existing symlink with new target.
- CLAUDE.md deploy skips regular files (user-owned).
- Migration: `.hystak-managed` marker converted to symlinks.
- Migration: sentinel CLAUDE.md converted to symlink.
- Broken symlinks (dangling target) detected and reported.

**Integration notes:** Must coordinate with Step 1 (skills stored in `~/.hystak/skills/`) and Step 2 (profile determines which skills to deploy).

**Demo:** Add skill "code-review" to registry. Assign to project profile. Sync. Verify `project/.claude/skills/code-review/SKILL.md` is a symlink pointing to `~/.hystak/skills/code-review/SKILL.md`. Remove from profile, re-sync, verify symlink gone.

---

## Step 5: Profile-Aware Sync Pipeline

**Objective:** Modify the sync pipeline to resolve items from a profile rather than directly from project assignments.

**Implementation guidance:**
- Add `Service.SyncProfile(projectName string, profileName string) ([]SyncResult, error)`.
- Resolution flow: load project → load profile → resolve MCPs/skills/hooks/permissions from profile selections → deploy via deployers.
- Profile's MCPs list references registry names → resolved the same way `ResolveServers` works today (but driven by profile, not project.MCPs).
- Profile's skills/hooks/permissions similarly resolved from registry.
- `SyncProject` becomes a wrapper: load active profile → call `SyncProfile`.
- Add `Service.HasLaunched(projectName string) bool` — checks project's `Launched` field.
- Add `Service.SetActiveProfile(projectName, profileName string) error`.
- Existing `SyncProject` callers (CLI sync command, TUI sync button) continue to work via the active profile.
- **Migration**: For existing projects without profiles, auto-generate a "default" profile from their current MCPs, skills, hooks, permissions, and claudeMD fields.

**Test requirements:**
- Sync with profile deploys only profile's items.
- Switching profile and re-syncing removes old items, deploys new ones.
- Unmanaged items preserved across profile switches.
- Empty/vanilla profile removes all managed items.
- Migration: existing project auto-gets "default" profile matching its current config.
- Backward compatibility: `hystak sync myproject` still works.

**Integration notes:** This is the core behavior change. Steps 1-4 are prerequisites. Steps 6+ consume this.

**Demo:** Create project with 5 MCPs in registry. Create profile "light" with 2 MCPs and "full" with all 5. Sync with "light" → verify 2 deployed. Switch to "full" → verify 5 deployed. Switch to "vanilla" → verify 0 managed, unmanaged preserved.

---

## Step 6: Boot Logo and Picker Updates

**Objective:** Add the ASCII art boot logo and update the picker to support wizard entry points.

**Implementation guidance:**
- Create `internal/tui/logo.go` with `RenderLogo() string`.
- ASCII art: stylized "hystak" text using lipgloss styling (match existing TUI color scheme).
- Display logo before picker. Options: brief display with auto-advance, or show as header in picker.
- Update `PickerModel` to add new options:
  - "Configure" option for existing projects (opens hub mode wizard).
  - First-launch detection: if selected project has `Launched: false`, auto-redirect to sequential wizard.
- Update `PickerResult` to include `Configure bool`.
- Update picker item rendering to show active profile name next to project name.

**Test requirements:**
- Logo renders without panics at various terminal widths.
- Picker shows "Configure" option.
- Picker result correctly reports Configure intent.
- Project without active profile shows "(new)" or similar indicator.
- Project with active profile shows profile name.

**Integration notes:** Picker changes feed into CLI integration (Step 9). Logo is standalone.

**Demo:** Launch hystak → see ASCII art logo → picker shows projects with profile names → "Configure" option visible → selecting it returns correct result.

---

## Step 7: Launch Wizard TUI — Sequential Mode

**Objective:** Build the first-launch wizard that walks through each configuration category step by step.

**Implementation guidance:**
- Create `internal/tui/launch_wizard.go` with `LaunchWizardModel`.
- Steps: MCPs → Skills → Permissions → Hooks → CLAUDE.md → Env Vars → Isolation.
- Each step shows discovered items (from Discovery Engine) as a multi-select list.
- Items pre-selected based on: extending a global profile (if chosen) or previous profile state.
- Navigation: Enter to advance, Esc to go back, Tab to skip step.
- For each category:
  - **MCPs**: Multi-select list. Items grouped by source (Registry, Global, Project). Toggle with Space.
  - **Skills**: Multi-select list. Show skill name + description. Toggle with Space.
  - **Permissions**: Multi-select from discovered permissions + free-text input (Ctrl+A to add custom).
  - **Hooks**: List existing hooks with toggle. "Add new" option opens hook form (reuse existing `HookFormModel`).
  - **CLAUDE.md**: Show available templates. Select one or "none". "Edit" opens `$EDITOR`.
  - **Env Vars**: Key=value table. Arrow keys to navigate, Enter to edit, Ctrl+A to add row, Ctrl+D to delete row.
  - **Isolation**: Radio select (none / worktree / lock).
- Register `ModeLaunchWizard` in AppModel. Handle `RequestLaunchWizardMsg` and `LaunchWizardCompleteMsg`.

**Test requirements:**
- All steps render without panics.
- Toggling items updates internal selection state.
- Skipping a step preserves default selections.
- Going back preserves previous step's selections.
- Completing all steps produces correct profile with selected items.
- Empty discovery → steps show empty state with guidance.

**Integration notes:** Reuse existing TUI components where possible (list.Model from bubbles, form models). Discovery Engine (Step 3) provides the data.

**Demo:** Launch wizard for a project. Walk through all 7 steps, toggling various items. Complete wizard. Print resulting profile selections — verify they match what was toggled.

---

## Step 8: Launch Wizard TUI — Hub Mode and Checklist

**Objective:** Add hub mode (direct category navigation) and the confirmation checklist for both modes.

**Implementation guidance:**
- **Hub mode**: Show a category menu on the left (MCPs, Skills, Permissions, Hooks, CLAUDE.md, Env Vars, Isolation). Selecting a category shows its configuration panel on the right. Same editing UI as sequential steps, but navigable in any order.
- When wizard mode is `modeHub`, skip the sequential walk-through and show the hub directly.
- **Checklist**: Final step in both modes. Shows a summary of all selections across all categories:
  - MCPs: list of enabled server names
  - Skills: list of enabled skill names
  - Hooks: count + list
  - Permissions: count + list
  - Env vars: count + list
  - CLAUDE.md: template name or "none"
  - Isolation: strategy name
- Checklist has "Launch" (Ctrl+L), "Edit" (jump back to hub), and "Cancel" (Esc) actions.
- Profile naming: prompt for profile name before saving (default: "default" for first-time, or auto-increment for new profiles).
- On "Launch": emit `LaunchWizardCompleteMsg{Profile: built_profile, Launch: true}`.

**Test requirements:**
- Hub mode renders category menu and content panel.
- Switching categories preserves selections in other categories.
- Checklist accurately reflects all selections.
- "Launch" emits correct message with complete profile.
- "Edit" returns to hub with all selections intact.
- Profile name prompt validates (no empty, no duplicates).

**Integration notes:** Combines with Step 7 to form the complete wizard. Consumed by Step 9 for the actual launch.

**Demo:** Open wizard in hub mode. Navigate to Skills, toggle some. Navigate to MCPs, toggle some. Navigate to Checklist — see both selections summarized. Press Launch.

---

## Step 9: CLI Integration and Launch Flow

**Objective:** Wire the boot logo, picker, wizard, sync, and launch into the complete CLI flow.

**Implementation guidance:**
- Modify `cli/root.go` `RunE`:
  1. Show boot logo.
  2. If `svc.IsEmpty()`, run first-time setup wizard (existing).
  3. Run picker.
  4. If picker result is a project with `Launched: false` → run launch wizard (sequential).
  5. If picker result is "Configure" → run launch wizard (hub).
  6. If picker result is a project with active profile → sync active profile and launch.
  7. If picker result is "Manage" → open management TUI.
  8. If picker result is "Vanilla" → launch bare.
- After wizard completes with `Launch: true`:
  - Save profile to project config.
  - Set as active profile.
  - Mark project as `Launched: true`.
  - Call `SyncProfile`.
  - Launch Claude Code.
- Add `--configure` flag to root command (opens wizard for the specified project).
- Add `hystak run --profile <name>` flag to use a specific profile.
- Ensure `hystak sync` uses active profile.

**Test requirements:**
- First-time project → sequential wizard → sync → launch.
- Returning project → skip wizard → sync → launch.
- `--configure` flag → hub mode wizard.
- `--profile` flag → use specified profile for sync.
- `hystak sync` uses active profile.
- All existing CLI commands still work.

**Integration notes:** This is the integration step. Everything from Steps 1-8 comes together here.

**Demo:** Full end-to-end: `hystak` → logo → pick project (first time) → wizard → configure 3 MCPs + 1 skill → checklist → launch → Claude starts with only those items in `.mcp.json` and `.claude/skills/`.

---

## Step 10: Worktree Isolation

**Objective:** Implement worktree-based isolation for concurrent sessions.

**Implementation guidance:**
- Create `internal/isolation/` package.
- `WorktreeManager`:
  - `Create(projectPath, profileName string) (worktreePath string, error)` — creates git worktree at a predictable path (e.g., `.git/hystak-worktrees/<profileName>/`).
  - `Exists(projectPath, profileName string) bool` — check if worktree already exists.
  - `Remove(projectPath, profileName string) error` — cleanup.
  - `List(projectPath string) ([]WorktreeInfo, error)` — list active worktrees.
- `LockManager`:
  - `Acquire(projectPath string) error` — create lock file with PID.
  - `Release(projectPath string) error` — remove lock file.
  - `IsLocked(projectPath string) (bool, int, error)` — check lock + return PID.
- Integration into launch flow:
  - If isolation is `worktree`: create worktree → deploy profile to worktree path → launch Claude in worktree.
  - If isolation is `lock`: acquire lock → deploy → launch → release on exit.
  - If isolation is `none`: deploy to project root as usual.

**Test requirements:**
- Worktree creation at expected path.
- Worktree has isolated `.mcp.json` and `.claude/` after deploy.
- Two worktrees for same project don't interfere.
- Worktree cleanup removes directory.
- Lock prevents second launch, shows PID.
- Lock released on normal exit.
- Lock stale detection (PID no longer running).
- Non-git directory with worktree isolation → graceful error.

**Integration notes:** Requires launch flow changes (Step 9) to route through isolation manager before deploying.

**Demo:** Set project isolation to `worktree`. Launch with "frontend" profile → worktree created, Claude running. In another terminal, launch with "backend" profile → second worktree, second Claude. Both running independently. Exit both → worktrees cleaned up.

---

## Step 11: v1 Reconfiguration Loop

**Objective:** When Claude Code exits, offer the user a chance to reconfigure before relaunching.

**Implementation guidance:**
- In the launch flow (after `os/exec.Command` or detecting Claude exit):
  - Show a prompt: "Claude exited. [R]elaunch / [C]onfigure / [Q]uit"
  - Relaunch: sync active profile → launch with `claude --continue`.
  - Configure: open wizard in hub mode → modify → sync → launch with `--continue`.
  - Quit: exit hystak.
- This requires the v1 launch to use `os/exec.Command` (not `syscall.Exec`) to keep hystak alive.
- **Migration**: Change `internal/launch/exec_unix.go` from `syscall.Exec` to `os/exec.Command` with proper stdio forwarding and signal passthrough.
- Ensure Ctrl+C in Claude exits Claude, not hystak (process group management with `Setpgid`).

**Test requirements:**
- Claude exit detected by parent hystak process.
- Relaunch passes `--continue` flag.
- Configure opens hub wizard with current profile pre-loaded.
- Quit exits cleanly.
- Ctrl+C during Claude session terminates Claude, not hystak.
- Signals (SIGINT, SIGTERM) forwarded correctly to child.

**Integration notes:** This is the first step of the v2 migration — switching from `syscall.Exec` to `os/exec.Command`. It doesn't implement SIGTSTP suspension yet, just the post-exit loop.

**Demo:** Launch project → Claude starts → user quits Claude → prompt appears → select "Configure" → modify profile → relaunch → Claude resumes previous session with new tools available.

---

## Step 12: Profile Sharing

**Objective:** Enable profile export/import for team sharing.

**Implementation guidance:**
- `profile.Manager.Export(name string) ([]byte, error)` — serialize profile to YAML.
- `profile.Manager.Import(data []byte) (*Profile, error)` — deserialize, validate, save.
- CLI commands:
  - `hystak profile export <name> [-o file.yaml]` — export to stdout or file.
  - `hystak profile import <file.yaml>` — import profile, prompt if name conflicts.
  - `hystak profile list` — list all profiles (global + project-scoped).
- Import validation: check that referenced MCPs, skills, hooks exist in registry. Warn about missing items, offer to import anyway (items will be skipped during sync).

**Test requirements:**
- Export produces valid YAML matching profile schema.
- Import round-trip: export → import → compare → identical.
- Import with missing registry items warns but succeeds.
- Name conflict on import → prompt for rename.
- CLI commands work end-to-end.

**Integration notes:** Standalone feature. No dependencies on other steps beyond Step 2 (profile model).

**Demo:** Create "frontend" profile with 5 MCPs and 3 skills. Export to `frontend.yaml`. Share with teammate. Teammate imports. Verify profile contents match.

---

## Step 13: v2 Process Manager (SIGTSTP/SIGCONT)

**Objective:** Implement mid-session reconfiguration by suspending Claude Code and showing the wizard.

**Implementation guidance:**
- Create `internal/launch/manager.go` with `ProcessManager`.
- Launch Claude as child process in its own process group (`Setpgid: true`).
- Install signal handlers:
  - Custom trigger (e.g., SIGUSR1 or a keybinding) → suspend Claude.
  - SIGCHLD with WUNTRACED → detect child suspension.
- On suspend:
  - Send SIGTSTP to Claude's process group.
  - Reclaim terminal via `tcsetpgrp(ttyFd, parentPgid)`.
  - Save terminal state (`tcgetattr`).
  - Run wizard TUI (hub mode).
  - On wizard complete: sync profile.
  - Restore terminal state (`tcsetattr`).
  - Return terminal to Claude via `tcsetpgrp(ttyFd, childPgid)`.
  - Send SIGCONT to Claude's process group.
- Extensive terminal state testing across macOS and Linux.
- Fallback: if suspension fails, fall back to v1 behavior (prompt on exit).

**Test requirements:**
- Claude suspends and resumes without terminal corruption.
- Wizard TUI appears cleanly after suspension.
- Config changes deployed before resume.
- Claude picks up new MCPs after resume.
- Fallback triggers on suspension failure.
- Works on macOS Terminal, iTerm2, and common Linux terminals.

**Integration notes:** High-risk step. Should be behind a feature flag initially. Builds on Step 11's `os/exec.Command` foundation.

**Demo:** Launch project → Claude running → trigger suspend → wizard appears → add an MCP → confirm → Claude resumes → new MCP available in Claude's session.

---

## Step 14: Polish and Edge Cases

**Objective:** Handle remaining edge cases, improve error messages, and polish the UX.

**Implementation guidance:**
- **Adopt unmanaged items**: In wizard, "unmanaged" items show an "Adopt" action. Adopting adds the item to the registry and includes it in the current profile.
- **Broken symlinks**: During sync, detect dangling symlinks (target deleted). Warn user, offer to remove or re-create.
- **Profile extends**: Implement `extends: global/profile-name` in project profiles. Resolved at load time (shallow merge: project overrides global).
- **Permission discovery from MCPs**: If MCP servers expose tool manifests, parse them to auto-suggest permission rules. Fallback: show common patterns.
- **Keyboard shortcuts**: Add `?` help overlay in wizard showing all shortcuts.
- **Progress indicators**: Show spinner during discovery scan and sync operations.
- **Error recovery**: If sync partially fails (e.g., one skill symlink fails), continue with others, show summary of failures.
- **Config validation**: On startup, validate `~/.hystak/` state. Warn about orphaned profiles, missing registry entries.

**Test requirements:**
- Adopt flow: unmanaged item → adopt → appears in registry + profile.
- Broken symlink detection and cleanup.
- Profile extends resolution with override semantics.
- Partial sync failure recovery.
- Help overlay renders correctly.

**Integration notes:** This step is a catch-all for quality improvements. Can be parallelized across the codebase.

**Demo:** Full polished experience: logo → picker → wizard (smooth animations, help text, progress indicators) → checklist → launch. No rough edges.
