# Launch Wizard — Requirements

## Questions & Answers

### Q1: When should the wizard appear?

Currently the flow is: **picker → sync → launch Claude**. Should the wizard appear:

- (a) **Every time** — after picking a project, always show the wizard before launching?
- (b) **First launch only** — show the wizard the first time a project is launched, then skip it on subsequent launches (go straight to sync → launch)?
- (c) **On demand** — add a flag or menu option (e.g., `--configure` or a picker choice) to open the wizard, otherwise launch directly?

**A1:** Both (b) and (c). Show the wizard on first launch of a project (never been launched before), then skip on subsequent launches. Also available on demand for reconfiguration.

### Q2: What exactly should be configurable in the wizard?

You mentioned skills, tools, plugins, and MCPs. In the current hystak model, a project has **server assignments** (MCPs from the registry) with optional overrides. What additional configuration categories should the wizard cover? For example:

- MCP servers (already in hystak — pick from registry, set overrides)
- Skills (`.claude/skills/` — are these managed by hystak today, or new?)
- Permissions (`allowedTools`, `autoApprove` lists for Claude Code)
- Hooks (pre/post commands in Claude's settings)
- CLAUDE.md / project instructions
- Environment variables
- Something else?

Which of these should the wizard handle?

**A2:** All of them, plus plugins and anything else Claude Code supports. The wizard should be the single pane of glass for configuring a Claude Code project. Full list:
- MCP servers (from registry, with overrides)
- Skills (`.claude/skills/`)
- Permissions (`allowedTools`, `autoApprove`)
- Hooks (pre/post commands)
- CLAUDE.md / project instructions
- Environment variables
- Plugins
- Any other Claude Code configuration surface

### Q3: What does "mid-session reconfiguration" look like?

You said you should be able to go back and load MCPs/tools/skills mid-session. Once Claude is running, hystak has handed off control. How do you envision getting back to the wizard?

- (a) **Interrupt Claude** — a keybinding or command within Claude (e.g., `/hystak`) that suspends Claude, returns to the wizard, re-syncs, then resumes Claude?
- (b) **Separate terminal** — run `hystak configure <project>` in another terminal, edit the config, then Claude picks up changes on next MCP refresh?
- (c) **Something else?**

**A3:** Option (a) — interrupt Claude, return to the wizard, re-sync, then resume Claude. The experience should be seamless: suspend → configure → re-sync → resume.

### Q4: How should the wizard UI be structured?

The wizard needs to cover a lot of categories (MCPs, skills, permissions, hooks, CLAUDE.md, env vars, plugins). What layout feels right?

- (a) **Sequential steps** — walk through each category one at a time (like an install wizard), with the option to skip categories
- (b) **Tab/menu hub** — a central screen with tabs or a menu where you pick which category to configure, in any order (similar to the current management TUI)
- (c) **Checklist overview** — show everything at a glance with toggle/edit options, then confirm and launch

Which approach, or a mix?

**A4:** All three combined. First launch walks through categories sequentially (A), but each step is a tab/menu you can navigate freely (B). At the end, a checklist overview (C) shows everything configured — confirm and launch. On-demand reconfiguration would go straight to the hub (B) since the user already knows what they want to change.

### Q5: The boot logo — what should it look like?

You mentioned wanting a logo on boot like Claude Code's startup banner. What are you envisioning?

- Just the hystak name in ASCII art / stylized text?
- Name + version + tagline?
- Name + project name being launched?
- Any color/theme preferences, or match the existing TUI styles?

**A5:** Just the hystak name in ASCII art / stylized text. Keep it simple.

### Q6: Suspend/resume mechanics — how should this work technically?

When the user invokes the wizard mid-session (suspending Claude), there are a few approaches:

- (a) **SIGTSTP** — hystak sends SIGTSTP to suspend the Claude process, runs the wizard TUI, re-syncs config, then sends SIGCONT to resume Claude. Claude stays alive in the background.
- (b) **Exit and relaunch** — hystak terminates Claude, runs the wizard, re-syncs, then relaunches Claude with the same session/conversation context (if Claude supports resumption).
- (c) **Don't worry about the mechanism yet** — just design the wizard and we'll figure out the suspend/resume plumbing later.

Which direction?

**A6:** Option (a) — SIGTSTP/SIGCONT sounds reasonable. Suspend the Claude process in place, run the wizard overlay, re-sync configs, then resume. Claude stays alive in the background so no session is lost.

### Q7: How should the wizard discover what's available to configure?

For MCPs, hystak already has a registry. But for skills, plugins, hooks, permissions — where does the wizard pull the list of available options from?

- (a) **Registry-only** — everything must be registered in hystak's registry first (skills, hooks, etc. become new registry entity types alongside servers)
- (b) **Filesystem discovery** — scan known paths (e.g., `.claude/skills/`, `settings.json`) to discover what's available, and the wizard lets you toggle/configure them
- (c) **Mixed** — registry for MCPs and curated items, filesystem discovery for skills/hooks/permissions that already exist on disk
- (d) **User provides paths/values** — the wizard just has input fields and the user types in what they want

**A7:** Option (b) — filesystem discovery. hystak's goal is to reduce bloat on launch. The wizard discovers everything available on the system (skills, MCPs, plugins, hooks, etc.) and lets the user selectively enable/disable what they want for this project. The point is curation — not "add everything" but "here's everything available, pick what you actually need."

### Q8: What does "reduce bloat" mean concretely for the deploy step?

When hystak syncs/deploys to a project's config files, should it:

- (a) **Only write what's enabled** — if the wizard has 20 MCPs discovered but only 5 toggled on, only those 5 go into `.mcp.json`. Skills not selected don't get symlinked/copied into `.claude/skills/`. Essentially, the wizard acts as a filter.
- (b) **Write everything but mark disabled** — use Claude Code's `disabled: true` field for MCPs, and some equivalent mechanism for skills/hooks, so they exist in config but aren't active.
- (c) **Option (a) by default, (b) as a user preference**

**A8:** Option (a) — only write what's enabled. The wizard is a filter. Deploy only the selected subset.

### Q9: Concurrency, isolation, and config ownership

**Concurrent sessions:** User-configurable isolation strategy per project:
- **None (default):** Deploy to project root, one active session at a time.
- **Worktree:** Each launch gets a git worktree with its own config files. True isolation for concurrent sessions / agent teams.
- **Lock:** Deploy to project root but prevent concurrent launches.

Configurable via project setting, launch flag (`--worktree`), or wizard prompt on first setup.

**Config ownership model:**
- `~/.hystak/` — hystak's source of truth (registry, profiles, project configs). hystak owns this entirely.
- `~/.claude/` — read-only discovery source. hystak never modifies this.
- `project/.claude/`, `project/.mcp.json` — deploy targets. hystak uses **symlinks** for file-based configs (skills, CLAUDE.md) so managed vs. unmanaged is obvious. For `.mcp.json` (single JSON file), hystak tracks managed entries via metadata in `~/.hystak/` and preserves unmanaged entries.

**Vanilla mode:** A built-in empty profile that deploys nothing. Switching to it removes all hystak symlinks/managed entries, leaving unmanaged configs intact.

### Q10: Profiles — how deep should they go?

Profiles are named loadouts (sets of enabled MCPs, skills, hooks, etc.). A few questions:

- Should profiles be **project-scoped** (each project has its own profiles), **global** (shared across projects), or **both**?
- Should profiles be **shareable** — e.g., exportable as a YAML file that a teammate can import?

**A10:** Both. Global profiles are reusable templates (e.g., "frontend-dev", "backend-debug") that can be applied to any project. Project-scoped profiles can extend or override a global profile. Profiles should be shareable — exportable as YAML for teammates to import.

### Q11: What should the wizard look like for each configuration category?

For the sequential wizard steps, each category needs a UI. For MCPs and skills, I'm imagining a **multi-select list** — discovered items shown with checkboxes, toggle on/off. But some categories are more complex:

- **Permissions** — adding tool names to allowlists. Free-text input? Or discover available tools from MCPs?
- **Hooks** — shell commands with trigger points (pre-tool, post-tool, etc.). A form with fields?
- **CLAUDE.md** — a text file. Open in `$EDITOR`? Inline editor? Or just show current content and ask if they want to edit?
- **Environment variables** — key=value pairs. Table editor?

Should the wizard aim for inline editing of everything, or is it acceptable to shell out to `$EDITOR` for complex items like CLAUDE.md and hooks?

**A11:** Per-category UI:
- **MCPs** — multi-select list with discovery from registry + filesystem. Toggle on/off.
- **Skills** — multi-select list with filesystem discovery. Toggle on/off.
- **Permissions** — discovery-based. Scan enabled MCPs for available tools, present as multi-select. Also allow free-text additions.
- **Hooks** — form-based. List discovered hook trigger points (pre-tool-use, post-tool-use, notification, etc.), let user attach shell commands. Show existing hooks for toggle/edit.
- **CLAUDE.md** — shell out to `$EDITOR`. Show preview in wizard.
- **Environment variables** — inline key=value table editor (add/edit/remove rows).
- **Plugins** — multi-select list with discovery. Toggle on/off.

### Q12: What happens to the existing management TUI?

hystak currently has a full management TUI (Servers tab, Projects tab, forms, diff view, import). The wizard overlaps significantly — it also lets you pick MCPs, configure overrides, etc.

- (a) **Wizard replaces the management TUI** — the wizard becomes the primary interface, and the management TUI is removed or reduced to a read-only viewer.
- (b) **Wizard is launch-focused, management TUI stays** — wizard is for "what do I want for THIS session", management TUI is for "manage the global registry/catalog". Separate concerns.
- (c) **Merge them** — the wizard IS the management TUI, just with a launch-oriented entry point.

**A12:** Option (c) — merge them. The existing TUI gains a launch wizard mode. Registry management and session curation live in the same interface. First-run walks through sequentially, on-demand goes straight to the hub. Reuses existing TUI components (tabs, forms, overlays). A "confirm & launch" step is added at the end.

### Q13: Discovery scope — where should hystak scan for available items?

For filesystem discovery, what locations should hystak scan?

- **MCPs:** `~/.claude.json` (global), project `.mcp.json` (local), hystak registry
- **Skills:** `~/.claude/skills/`, project `.claude/skills/`
- **Hooks:** `~/.claude/settings.json`, project `.claude/settings.local.json`
- **Permissions:** same settings files as hooks
- **Plugins:** where does Claude Code store plugin configs?
- **CLAUDE.md:** project root, `~/.claude/CLAUDE.md`

Are there other locations I'm missing? And for plugins specifically — how are they configured in your setup?

**A13:** Discovery locations confirmed:
- **MCPs:** `~/.claude.json` (global), project `.mcp.json` (local), hystak registry
- **Skills:** `~/.claude/skills/` (global), project `.claude/skills/` (local)
- **Hooks:** `~/.claude/settings.json` (global), project `.claude/settings.local.json` (local)
- **Permissions:** same settings files as hooks
- **CLAUDE.md:** project root, `~/.claude/CLAUDE.md` (global)

Scan both global and project/repo scope. Plugins to be clarified later.

### Q14: Error handling and edge cases

A few scenarios to decide on:

- **Dirty project configs** — user manually edited `.mcp.json` outside of hystak. On next wizard launch, hystak discovers entries it didn't create. Should the wizard flag these and offer to adopt them into the profile, or just leave them as unmanaged?

**A14:** Option (c) — show unmanaged items tagged as such, and offer an "adopt" action to bring them under hystak management.

### Q15: Is there anything else about the launch experience you want to address?

We've covered:
- Wizard trigger (first launch + on demand)
- Config categories (MCPs, skills, permissions, hooks, CLAUDE.md, env vars, plugins)
- Mid-session reconfiguration (SIGTSTP suspend/resume)
- UI structure (sequential → hub → checklist confirm)
- Boot logo (ASCII art)
- Concurrency (user-configurable isolation: none/worktree/lock)
- Config ownership (~/.hystak source of truth, ~/.claude read-only, project = deploy target via symlinks)
- Profiles (global + project-scoped, shareable as YAML)
- Per-category UI (multi-select, forms, $EDITOR, key=value)
- Merged with existing management TUI
- Discovery from global + project scope
- Unmanaged config handling (show + offer adopt)

Is there anything else that's important to the vision that we haven't covered? Or are you ready to move on?
