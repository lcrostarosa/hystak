# hystak — TUI Wireframe Specification

## Information Architecture

```
┌─────────────────────────────────────────────────────────┐
│  hystak                              [profile: dev]  ▊  │
├───────────┬───────────┬───────────┬─────────────────────┤
│ Registry  │ Projects  │  Tools    │  Help               │
└───────────┴───────────┴───────────┴─────────────────────┘
```

**4 tabs** (down from 7+):

| Tab | Purpose | Keyboard |
|-----|---------|----------|
| **Registry** | All managed resources with sub-navigation | `1` or Tab |
| **Projects** | Project list + profile management | `2` or Tab |
| **Tools** | Import, Discover, Diff, Launch, Doctor | `3` or Tab |
| **Help** | Keybindings reference, version | `4` or Tab |

---

## Tab 1: Registry

### Layout — Master/Detail with Sub-Nav

```
┌─ Registry ──────────────────────────────────────────────┐
│                                                         │
│  [MCPs]  Skills  Hooks  Permissions  Templates  Prompts │
│  ─────                                                  │
│                                                         │
│  NAME              TRANSPORT   COMMAND/URL              │
│  ────────────────  ─────────   ──────────────────────── │
│▸ github            stdio       npx -y @anthropic/mcp-…  │
│  postgres          stdio       npx -y @anthropic/mcp-…  │
│  remote-api        sse         https://mcp.example.co…  │
│  filesystem        stdio       npx -y @anthropic/mcp-…  │
│  puppeteer         stdio       npx -y @anthropic/mcp-…  │
│                                                         │
│                                                         │
│                                                         │
│                                                         │
│                                                         │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ A:Add  E:Edit  D:Delete  /:Filter  I:Import  T:Tag     │
└─────────────────────────────────────────────────────────┘
```

- Sub-nav row switches between resource types
- Arrow keys (or vim keys) navigate the sub-nav
- Enter or right-arrow selects a sub-nav item
- The table adapts columns per resource type

### Sub-Nav Column Layouts

| Sub-view | Columns |
|----------|---------|
| MCPs | NAME, TRANSPORT, COMMAND/URL |
| Skills | NAME, DESCRIPTION, SOURCE |
| Hooks | NAME, EVENT, MATCHER, COMMAND |
| Permissions | NAME, RULE, TYPE |
| Templates | NAME, SOURCE |
| Prompts | NAME, CATEGORY, ORDER, TAGS |

### Filter Mode

```
┌─ Registry ──────────────────────────────────────────────┐
│                                                         │
│  [MCPs]  Skills  Hooks  Permissions  Templates  Prompts │
│  ─────                                                  │
│                                                         │
│  Filter: post█                                          │
│                                                         │
│  NAME              TRANSPORT   COMMAND/URL              │
│  ────────────────  ─────────   ──────────────────────── │
│▸ postgres          stdio       npx -y @anthropic/mcp-…  │
│                                                         │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Esc:Clear filter   Enter:Select                         │
└─────────────────────────────────────────────────────────┘
```

### Add/Edit MCP Overlay

```
┌─ Add MCP Server ────────────────────────────────────────┐
│                                                         │
│  Name:        [________________________]                │
│                                                         │
│  Transport:   (●) stdio  ( ) sse  ( ) http              │
│                                                         │
│  Command:     [npx_______________________]              │
│                                                         │
│  Args:        [-y, @anthropic/mcp-github_]              │
│                                                         │
│  Env:         GITHUB_TOKEN = ${GITHUB_TOKEN}            │
│               [+ Add env var]                           │
│                                                         │
│  Headers:     (hidden — stdio transport)                │
│                                                         │
│  Description: [GitHub MCP server_________]              │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Tab:Next field  Shift+Tab:Prev  Enter:Save  Esc:Cancel  │
└─────────────────────────────────────────────────────────┘
```

- Transport selection shows/hides relevant fields (command+args for stdio, url+headers for sse/http)
- Env and Headers are expandable key-value editors

---

## Tab 2: Projects

### Layout — List + Detail Split

```
┌─ Projects ──────────────────────────────────────────────┐
│                                                         │
│  PROJECTS              │  interview-platform            │
│  ──────────────────    │  ──────────────────────────    │
│▸ interview-platform    │  Path: /Volumes/Secondary/…    │
│  personal-site         │  Active: dev                   │
│  global                │                                │
│                        │  Profiles:                     │
│                        │    ● dev (active)              │
│                        │    ○ review                    │
│                        │    ○ minimal                   │
│                        │    ○ empty                     │
│                        │                                │
│                        │  MCPs (3):                     │
│                        │    ☑ github                    │
│                        │    ☑ postgres                  │
│                        │    ☑ remote-api                │
│                        │    ☐ filesystem                │
│                        │    ☐ puppeteer                 │
│                        │                                │
│                        │  Skills (2):                   │
│                        │    ☑ code-review               │
│                        │    ☑ commit                    │
│                        │                                │
│                        │  Hooks (1):                    │
│                        │    ☑ lint-on-edit              │
│                        │    ☐ block-rm-rf               │
│                        │                                │
│                        │  Permissions (3):              │
│                        │    ☑ allow-bash                │
│                        │    ☑ allow-read                │
│                        │    ☑ deny-rm                   │
│                        │                                │
│                        │  Template: standard            │
│                        │                                │
├─────────────────────────────────────────────────────────┤
│ A:Add  D:Delete  P:Profile  L:Launch  S:Sync  Space:Toggle │
└─────────────────────────────────────────────────────────┘
```

- Left pane: project list (scrollable)
- Right pane: detail view for selected project
- Right pane scrolls independently for long resource lists
- Space toggles checkboxes (assigns/unassigns from active profile)
- `P` opens a profile selector to switch active profile

### Profile Selector Overlay

```
┌─ Select Profile ────────────────────────────────────────┐
│                                                         │
│  interview-platform                                     │
│                                                         │
│  ● dev         Full development environment    (active) │
│  ○ review      Code review — read-only tools            │
│  ○ minimal     Just GitHub and filesystem               │
│  ○ empty       Clean launch — nothing enabled           │
│                                                         │
│  [+ New Profile]                                        │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Enter:Activate  N:New  D:Delete  Esc:Cancel             │
└─────────────────────────────────────────────────────────┘
```

---

## Tab 3: Tools

### Layout — Action Grid

```
┌─ Tools ─────────────────────────────────────────────────┐
│                                                         │
│   ┌───────────────┐  ┌───────────────┐                  │
│   │   Import      │  │   Discover    │                  │
│   │               │  │               │                  │
│   │  Import MCPs  │  │  Scan skills  │                  │
│   │  from file    │  │  in project   │                  │
│   └───────────────┘  └───────────────┘                  │
│                                                         │
│   ┌───────────────┐  ┌───────────────┐                  │
│   │   Diff        │  │   Doctor      │                  │
│   │               │  │               │                  │
│   │  Show config  │  │  Validate     │                  │
│   │  drift        │  │  registry     │                  │
│   └───────────────┘  └───────────────┘                  │
│                                                         │
│   ┌───────────────┐  ┌───────────────┐                  │
│   │   Launch      │  │   Backup      │                  │
│   │               │  │               │                  │
│   │  Sync + run   │  │  Backup or    │                  │
│   │  Claude Code  │  │  restore      │                  │
│   └───────────────┘  └───────────────┘                  │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Arrow keys to navigate  Enter:Select                    │
└─────────────────────────────────────────────────────────┘
```

### Import Overlay

```
┌─ Import ────────────────────────────────────────────────┐
│                                                         │
│  Source file: [~/.claude.json_______________] (Tab:Browse)│
│                                                         │
│  Discovered servers:                                    │
│                                                         │
│  ☑ github            stdio   npx -y @anthropic/mcp-…   │
│  ☑ postgres          stdio   npx -y @anthropic/mcp-…   │
│  ☐ my-custom-server  stdio   node server.js             │
│  ⚠ filesystem        stdio   npx -y @anthropic/mcp-…   │
│    └─ Conflict: "filesystem" already in registry        │
│       (K)eep existing  (R)eplace  Re(N)ame  (S)kip     │
│                                                         │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Space:Toggle  Enter:Import selected  Esc:Cancel         │
└─────────────────────────────────────────────────────────┘
```

### Diff Overlay

```
┌─ Drift: interview-platform ─────────────────────────────┐
│                                                         │
│  SERVER           STATUS                                │
│  ───────────────  ──────────                            │
│  github           ✓ synced                              │
│▸ postgres         ~ drifted                             │
│  remote-api       + missing (not deployed)              │
│  custom-server    ? unmanaged                           │
│                                                         │
│ ─── postgres: diff ──────────────────────────────────── │
│                                                         │
│  "postgres": {                                          │
│    "type": "stdio",                                     │
│    "command": "npx",                                    │
│    "args": ["-y", "@anthropic/mcp-postgres"],           │
│    "env": {                                             │
│-     "DATABASE_URL": "postgres://old-host/db"           │
│+     "DATABASE_URL": "postgres://new-host/db"           │
│    }                                                    │
│  }                                                      │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ S:Sync now  Esc:Close                                   │
└─────────────────────────────────────────────────────────┘
```

Status legend:
- `✓` synced (green)
- `~` drifted (yellow)
- `+` missing — in profile but not deployed (blue)
- `?` unmanaged — in config but not in profile (dim)

### Doctor Overlay

```
┌─ Doctor ────────────────────────────────────────────────┐
│                                                         │
│  Checking registry...                                   │
│                                                         │
│  ✓ 5 MCP servers                                        │
│  ✓ 2 skills                                             │
│  ✓ 2 hooks                                              │
│  ✓ 3 permissions                                        │
│  ✓ 1 template                                           │
│  ✓ 2 prompts                                            │
│  ✓ 1 tag                                                │
│                                                         │
│  Checking projects...                                   │
│                                                         │
│  interview-platform:                                    │
│    ✓ Profile "dev" valid                                │
│    ⚠ Profile "review" references missing skill "audit"  │
│                                                         │
│  personal-site:                                         │
│    ✓ Profile "default" valid                            │
│                                                         │
│  1 warning, 0 errors                                    │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Esc:Close                                               │
└─────────────────────────────────────────────────────────┘
```

---

## Tab 4: Help

```
┌─ Help ──────────────────────────────────────────────────┐
│                                                         │
│  hystak v0.1.0 (abc1234, 2026-03-22)                    │
│                                                         │
│  Navigation                                             │
│  ──────────                                             │
│  Tab / Shift+Tab    Switch tabs                         │
│  ↑ / ↓              Navigate lists                      │
│  Enter              Select / confirm                    │
│  Esc                Cancel / close overlay               │
│  Space              Toggle selection                    │
│  /                  Filter mode                         │
│  q                  Quit                                │
│                                                         │
│  Actions                                                │
│  ───────                                                │
│  A                  Add new item                        │
│  E                  Edit selected item                  │
│  D                  Delete selected item                │
│  I                  Import (Registry tab)               │
│  L                  Launch (Projects tab)               │
│  S                  Sync (Projects tab)                 │
│  P                  Preview / Profile                   │
│                                                         │
│  CLI Commands                                           │
│  ────────────                                           │
│  hystak                  Launch TUI                     │
│  hystak setup            Re-run first-time setup        │
│  hystak list             List registry servers          │
│  hystak sync <project>   Sync project configs           │
│  hystak diff <project>   Show config drift              │
│  hystak run <project>    Sync + launch Claude Code      │
│  hystak backup <proj>    Backup project configs         │
│  hystak restore <proj>   Restore from backup            │
│  hystak undo [<proj>]    Undo last sync                 │
│  hystak doctor           Validate registry              │
│  hystak profile list     List all profiles              │
│  hystak version          Show version info              │
│  hystak completion       Generate shell completions     │
│                                                         │
│  Config: ~/.hystak/                                     │
│  Keybindings: ~/.hystak/keys.yaml                       │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ q:Quit                                                  │
└─────────────────────────────────────────────────────────┘
```

---

## Launch Wizard — Sequential Mode

### Step 1: MCPs (Primary Value)

```
┌─ Launch Wizard ─── Step 1 of 3: MCPs ──────────────────┐
│                                                         │
│  Select MCP servers for this profile.                   │
│                                                         │
│  Registry:                                              │
│  ☑ github            stdio   (used by 2 profiles)       │
│  ☑ postgres          stdio   (used by 1 profile)        │
│  ☐ remote-api        sse     (used by 0 profiles)       │
│  ☐ filesystem        stdio   (used by 3 profiles)       │
│  ☐ puppeteer         stdio   (used by 0 profiles)       │
│                                                         │
│  Discovered (not yet in registry):                      │
│  ☐ slack             stdio   from .mcp.json             │
│                                                         │
│  Catalog:                                               │
│  ☐ brave-search      stdio   ★ popular                  │
│  ☐ sequential-think  stdio   ★ popular                  │
│  ☐ memory            stdio                              │
│                                                         │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Space:Toggle  Enter:Next step  Esc:Cancel               │
└─────────────────────────────────────────────────────────┘
```

### Step 2: Quick Options

```
┌─ Launch Wizard ─── Step 2 of 3: Options ───────────────┐
│                                                         │
│  ▾ Skills (2 selected)                                  │
│    ☑ code-review                                        │
│    ☑ commit                                             │
│                                                         │
│  ▾ Permissions (3 selected)                             │
│    ☑ allow-bash                                         │
│    ☑ allow-read                                         │
│    ☑ deny-rm                                            │
│                                                         │
│  ▸ Hooks (0 selected)                                   │
│                                                         │
│  ▸ CLAUDE.md (none)                                     │
│                                                         │
│                                                         │
│                                                         │
│                                                         │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Space:Toggle  Enter:Next  ◀:Back  ▸:Expand  ▾:Collapse  │
└─────────────────────────────────────────────────────────┘
```

### Step 3: Review & Launch

```
┌─ Launch Wizard ─── Step 3 of 3: Review ────────────────┐
│                                                         │
│  Profile: dev                                           │
│  Project: interview-platform                            │
│                                                         │
│  ┌─────────────────────────────────────────────────┐    │
│  │  MCPs           2   github, postgres             │    │
│  │  Skills         2   code-review, commit          │    │
│  │  Permissions    3   allow-bash, allow-read, …    │    │
│  │  Hooks          0                                │    │
│  │  Template       1   standard                     │    │
│  │  Prompts        2   security-rules, style-guide  │    │
│  │  Env Vars       0                                │    │
│  │  Isolation      none                             │    │
│  └─────────────────────────────────────────────────┘    │
│                                                         │
│                                                         │
│  Ready to sync and launch Claude Code.                  │
│                                                         │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Enter:Launch  E:Edit (hub mode)  Esc:Cancel             │
└─────────────────────────────────────────────────────────┘
```

---

## Launch Wizard — Hub Mode

```
┌─ Configure: interview-platform ─────────────────────────┐
│                                                         │
│  ┌────────────────┐  ┌──────────────────────────────┐   │
│  │                │  │                              │   │
│  │  MCPs      (2) │  │  ☑ github                    │   │
│  │  Skills    (2) │  │  ☑ postgres                  │   │
│  │  Permissions(3)│  │  ☐ remote-api                │   │
│  │  Hooks     (0) │  │  ☐ filesystem                │   │
│  │  CLAUDE.md (1) │  │  ☐ puppeteer                 │   │
│  │▸ Prompts   (2) │  │                              │   │
│  │  Env Vars  (0) │  │                              │   │
│  │  Isolation     │  │                              │   │
│  │                │  │                              │   │
│  │  ─────────     │  │                              │   │
│  │  Review        │  │                              │   │
│  │                │  │                              │   │
│  └────────────────┘  └──────────────────────────────┘   │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ ↑↓:Category  Space:Toggle  Enter:Review  Esc:Cancel     │
└─────────────────────────────────────────────────────────┘
```

- Left sidebar: categories with selection counts
- Right pane: toggle list for selected category
- "Review" at bottom of sidebar jumps to review screen

---

## Post-Exit Loop

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│  Claude Code exited (0).                                │
│                                                         │
│  [R] Relaunch    Sync and restart Claude Code           │
│  [C] Configure   Edit profile, then relaunch            │
│  [Q] Quit        Exit hystak                            │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

---

## Conflict Resolution Overlay

```
┌─ Sync Conflicts ────────────────────────────────────────┐
│                                                         │
│  The following files already exist and are not managed   │
│  by hystak. Choose how to resolve each conflict:        │
│                                                         │
│  1. .claude/settings.local.json                         │
│     ├─ Status: User-owned file (not a symlink)          │
│     └─ Action: (K)eep  (R)eplace  (S)kip               │
│                                                         │
│  2. CLAUDE.md                                           │
│     ├─ Status: User-owned file (no managed sentinel)    │
│     └─ Action: (K)eep  (R)eplace  (S)kip               │
│                                                         │
│                                                         │
│  [A] Apply same action to all remaining                 │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ K/R/S per item  A:Apply to all  Esc:Abort sync          │
└─────────────────────────────────────────────────────────┘
```

---

## Responsive Behavior

### Minimum Terminal Size
- **Width**: 80 columns (degrades gracefully below — truncates columns)
- **Height**: 24 rows

### Width Adaptations

| Width | Behavior |
|-------|----------|
| 80–99 | Truncate long paths with `…` |
| 100–119 | Full paths shown |
| 120+ | Projects tab shows wider detail pane |

### Detail Pane Collapse
Below 100 columns, the Projects tab switches from side-by-side to stacked layout:
- Press Enter on a project to show its detail view
- Esc returns to the project list

---

## Color Scheme

Monochrome-compatible with color enhancement:

| Element | Color | Fallback (no color) |
|---------|-------|---------------------|
| Active tab | Bold + underline | Bold + underline |
| Selected row | Reverse video | Reverse video |
| Checked ☑ | Green | `[x]` |
| Unchecked ☐ | Dim | `[ ]` |
| Error/conflict | Red | `!` prefix |
| Warning | Yellow | `?` prefix |
| Synced status | Green | `✓` |
| Drifted status | Yellow | `~` |
| Missing status | Blue | `+` |
| Unmanaged status | Dim | `?` |
| Diff added | Green | `+` prefix |
| Diff removed | Red | `-` prefix |
| Overlay border | Cyan | Standard box-drawing |
| Footer shortcuts | Bold keys + dim descriptions | Same |

### TERM Handling
- 256-color and truecolor: use the palette above
- 16-color: map to nearest ANSI colors
- `NO_COLOR` / `TERM=dumb`: monochrome fallback with text indicators
