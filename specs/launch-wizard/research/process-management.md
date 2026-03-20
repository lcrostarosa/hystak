# Process Management Research: SIGTSTP/SIGCONT Feasibility

## Current Launch Mechanism

- **Unix:** `syscall.Exec()` — REPLACES hystak process with Claude. No parent survives.
- **Windows:** `os/exec.Command()` — spawns child, parent waits.

**This is the fundamental blocker.** syscall.Exec means there's no parent to manage suspension.

## What Needs to Change

1. Switch Unix from `syscall.Exec()` to `os/exec.Command()` with `Setpgid: true`
2. Parent stays alive, manages terminal ownership via `tcsetpgrp()`
3. Install SIGCHLD handler with WUNTRACED to detect child suspension
4. On suspension: save terminal state, reclaim terminal, show TUI
5. On resume: return terminal to child, send SIGCONT

## Risk Assessment

| Risk | Severity | Notes |
|------|----------|-------|
| Terminal state corruption | HIGH | Claude Code uses raw mode + alt screen. No save/restore API. |
| Race conditions | HIGH | tcsetpgrp + signal handlers + process groups |
| Platform differences | MEDIUM | macOS Darwin vs Linux ioctl behavior |
| Breaking Unix launch semantics | MEDIUM | Fork overhead, process lifecycle change |

## Alternatives

### Option 1: Shell-Style Job Control (recommended if pursuing SIGTSTP)
- os/exec.Command + process groups + tcsetpgrp + signal handlers
- 3-phase implementation: arch change → terminal handling → Bubble Tea integration

### Option 2: Transient Process Manager (simpler)
- Wrapper that spawns Claude as managed child
- Use SIGSTOP (not SIGTSTP) for deterministic suspension
- Avoids terminal ownership juggling

### Option 3: Sequential Workflow (safest)
- Keep syscall.Exec as-is
- No mid-session TUI — user exits Claude, reconfigures, relaunches
- Claude Code supports session resumption via `--continue`

## Recommendation

Start with Option 3 (sequential) for v1 — lowest risk, still useful.
Plan Option 1 (shell-style job control) for v2 once the wizard UX is proven.
The architectural change (syscall.Exec → os/exec.Command) can be done incrementally.

## References

- Claude Code issue #11898: iTerm2 suspension failures
- Claude Code issue #3355: terminal state corruption
- Go issue #41996: SIGTSTP handling limitations
- Bubble Tea: built-in SuspendMsg/ResumeMsg support
