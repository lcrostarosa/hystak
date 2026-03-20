# TUI Architecture Research

## Component Model

- Root `AppModel` manages entire state
- 6 tabs: Profiles, MCPs, Skills, Hooks, Permissions, Templates
- Mode-based overlay system (Form, Confirm, Diff, Import, Discovery, Conflict)
- Child-to-parent communication via custom `tea.Msg` types (Request/Completion pattern)

## Existing Wizard (wizard.go, 739 lines)

Multi-step state machine:
- wizardWelcome → wizardScanResult → wizardCatalog → wizardProjectForm → wizardSummary → wizardDone
- Async filesystem scanning via `tea.Cmd` returning `scanCompleteMsg`
- Catalog browsing with 4 sections (MCPs, Skills, Hooks, Permissions) and tab/toggle navigation
- Text input with auto-populate from CWD

## Existing Picker (picker.go, 176 lines)

- Uses `charmbracelet/bubbles/list` with filtering
- Three result types: Project launch, Bare launch, Manage (open full TUI)
- Returns `PickerResult` consumed by `cli/root.go`

## Reusable Components

- Multi-select lists (all tab models)
- Form overlays with conditional field visibility
- Confirmation dialogs
- Diff viewer with syntax coloring
- Import flow (3-phase: scan, select, apply)
- Discovery model (scan project, toggle, import)
- Conflict resolution (one-at-a-time with keep/replace/skip)

## Key Patterns for Launch Wizard

1. Define new Mode constant + model field in AppModel
2. Use Request/Completion message pattern for lifecycle
3. Overlay routing via `activeOverlay()` method
4. `SetSize()` on all components for responsive layout
5. `IsConsuming()` to block global shortcuts during focused input

## File References

- `internal/tui/app.go` — root model, mode/tab dispatch
- `internal/tui/wizard.go` — setup wizard (reusable pattern)
- `internal/tui/picker.go` — project picker
- `internal/tui/profiles.go` — profile management (launch trigger)
- `internal/tui/mcps.go` — MCP browsing (reusable for wizard steps)
