# hystak — Prioritized Story Map

## Priority Definitions

| Priority | Meaning | Ship Gate |
|----------|---------|-----------|
| **P0** | Core loop — without these, the tool has no value | v0.1 (MVP) |
| **P1** | Daily-use polish — makes the tool usable for real workflows | v0.2 |
| **P2** | Power-user features — unlocks advanced use cases | v0.3 |
| **P3** | Edge cases & nice-to-have — completeness | v1.0 |

---

## P0 — Core Loop (MVP)

**Goal:** A user can register servers, assign them to a project, sync, and launch Claude Code.

| ID | Story | Effort |
|----|-------|--------|
| S-001 | First-run detection + config directory creation | S |
| S-002 | Keybinding prompt (single inline question) | S |
| S-003 | Scan existing configs for MCP servers | M |
| S-004 | Register first project (auto-fill from cwd) | S |
| S-012 | Add/Edit/Delete MCP servers in TUI | L |
| S-019 | `hystak list` (CLI server listing) | S |
| S-023 | Add project from unregistered directory | M |
| S-025 | Assign MCPs to project profile | M |
| S-028 | Set active profile | S |
| S-033 | `hystak sync <project>` (core sync engine) | L |
| S-042 | Deploy MCP servers to `.mcp.json` | M |
| S-038 | Preserve unmanaged servers during sync | M |
| S-053 | Launch Claude Code from TUI | M |
| S-054 | `hystak run <project>` with post-exit loop | M |
| S-082 | Malformed config error handling | S |
| S-083 | Auto-create `~/.hystak/` directory | S |
| S-085 | Bootstrap missing client config files | S |

**Total: 17 stories | Est. 2-3 weeks**

### P0 Acceptance Criteria
```
1. `hystak` in a new directory → prompted to register project
2. Add 3 MCP servers via TUI
3. Assign 2 of them to the project
4. `hystak sync` → .mcp.json written correctly
5. `hystak run` → Claude Code launches with those 2 servers
6. Manually-added servers in .mcp.json preserved
```

---

## P1 — Daily Use

**Goal:** Auto-discovery, profiles, drift detection, and filtering make the tool reliable for daily workflows.

| ID | Story | Effort |
|----|-------|--------|
| S-005 | Direct to Launch Wizard after first-run | M |
| S-007 | Silent auto-discovery on startup | M |
| S-008 | Non-blocking discovery errors | S |
| S-018 | Filter servers by name (`/`) | S |
| S-027 | Multiple profiles per project | M |
| S-029 | Built-in "empty" profile | S |
| S-034 | Set active profile in Projects tab | S |
| S-037 | Override merge during sync (env, args, command) | M |
| S-039 | Remove deactivated servers from config | M |
| S-040 | Automatic backup before sync | M |
| S-041 | Missing server error (fail sync cleanly) | S |
| S-049 | `hystak diff <project>` (drift detection) | M |
| S-050 | "No drift detected" output | S |
| S-052 | Semantic drift comparison | M |
| S-054 | Post-exit loop (R/C/Q) | M |
| S-055 | `hystak run --profile <name>` | S |
| S-060 | Launch Wizard — sequential mode (3 steps) | L |
| S-075 | Non-TTY detection (help instead of TUI) | S |
| S-080 | `hystak version` | S |

**Total: 19 stories | Est. 2-3 weeks**

### P1 Acceptance Criteria
```
1. Add a server to .mcp.json manually → hystak discovers it on next run
2. Create "dev" and "review" profiles with different server sets
3. Switch active profile → sync deploys correct set
4. `hystak diff` shows drifted servers after manual edit
5. Profile override changes env var on one server for one project
6. Backup created before each sync
```

---

## P2 — Power User

**Goal:** Full resource types, import/export, wizard hub mode, backup/restore, and the complete CLI surface.

### Registry Resources
| ID | Story | Effort |
|----|-------|--------|
| S-009 | Manual import overlay | M |
| S-010 | Import conflict resolution (Keep/Replace/Rename/Skip) | M |
| S-011 | Skill discovery from `.claude/skills/` | M |
| S-013 | Add/Edit/Delete skills | M |
| S-014 | Add/Edit/Delete hooks | M |
| S-015 | Add/Edit/Delete permission rules | M |
| S-016 | Add/Edit/Delete templates | S |
| S-017 | Add/Edit/Delete prompt fragments + preview | M |
| S-018 | Delete cascading to profiles | M |
| S-020 | Tag create/edit | M |
| S-021 | Tag expansion during sync | M |

### Sync & Deploy
| ID | Story | Effort |
|----|-------|--------|
| S-034 | Sync all projects (`--all`) | S |
| S-035 | Sync with specific profile (`--profile`) | S |
| S-036 | Sync dry-run (`--dry-run`) | M |
| S-042 | Deploy to global config (`~/.claude.json`) | S |
| S-043 | Deploy skills as symlinks | M |
| S-044 | Deploy hooks & permissions to `settings.local.json` | M |
| S-045 | Deploy CLAUDE.md (symlink or composed) | M |

### Profiles
| ID | Story | Effort |
|----|-------|--------|
| S-030 | Export profile to YAML | M |
| S-031 | Import profile from YAML | M |
| S-032 | List profiles (CLI) | S |

### Backup & Restore
| ID | Story | Effort |
|----|-------|--------|
| S-066 | `hystak backup <project>` | M |
| S-067 | List backups | S |
| S-068 | Interactive restore | M |
| S-069 | `hystak undo` (quick restore) | M |
| S-070 | Backup retention pruning | S |

### Launch
| ID | Story | Effort |
|----|-------|--------|
| S-056 | `hystak run --no-sync` | S |
| S-057 | `hystak run --dry-run` | S |
| S-058 | Explicit client (`hystak run <project> <client>`) | S |
| S-059 | Forward extra args to client | S |
| S-061 | Launch Wizard — hub mode | L |
| S-062 | Env var editor in wizard | M |

**Total: 31 stories | Est. 4-5 weeks**

---

## P3 — Completeness

**Goal:** Edge cases, isolation strategies, advanced UX, full error recovery.

### Conflict Resolution
| ID | Story | Effort |
|----|-------|--------|
| S-046 | Preflight conflict detection | M |
| S-047 | Conflict resolution overlay | M |
| S-048 | Symlinks are not conflicts | S |

### Drift
| ID | Story | Effort |
|----|-------|--------|
| S-051 | Drift overlay in TUI | L |

### Isolation
| ID | Story | Effort |
|----|-------|--------|
| S-063 | Isolation strategy selection in wizard | M |
| S-064 | Worktree isolation (create/reuse/error) | L |
| S-065 | Lock isolation (create/detect/stale cleanup) | M |

### Validation & Migration
| ID | Story | Effort |
|----|-------|--------|
| S-074 | `hystak doctor` (registry validation) | L |
| S-086 | Missing skill source error | S |

### CLI Polish
| ID | Story | Effort |
|----|-------|--------|
| S-006 | `hystak setup` (re-run wizard) | S |
| S-022 | Dangling tag reference error | S |
| S-072 | Dual YAML format (bare string + map) | S |
| S-076 | `--json` output flag | M |
| S-077 | `--quiet` flag | S |
| S-079 | `--config-dir` override | S |
| S-081 | Shell completions (bash/zsh/fish) | M |

**Total: 20 stories | Est. 3-4 weeks**

---

## Dependency Graph

```
P0 (core sync)
 │
 ├── P1 (profiles + drift)
 │    │
 │    ├── P2 (full resources + import/export + backup)
 │    │    │
 │    │    └── P3 (isolation + conflicts + doctor)
 │    │
 │    └── P2 (launch wizard hub mode)
 │
 └── P1 (auto-discovery)
      │
      └── P2 (import overlay + skill discovery)
```

---

## Milestone Timeline

| Milestone | Stories | Calendar Est. |
|-----------|---------|---------------|
| **v0.1 — MVP** | P0 (17 stories) | Week 3 |
| **v0.2 — Daily Driver** | P0 + P1 (36 stories) | Week 6 |
| **v0.3 — Power User** | P0–P2 (67 stories) | Week 11 |
| **v1.0 — Complete** | All (87 stories) | Week 14 |

