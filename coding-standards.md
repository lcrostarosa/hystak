# Coding Standards

Anti-patterns found in hystak v1 and the rules that replace them for the rebuild.

---

## 1. Error Handling

### Anti-pattern: Silent error swallowing

v1 discards errors with `_ =` or bare `continue` in at least 8 critical paths. Examples:

```go
// service.go — deployer init failure silently skipped
d, err := deploy.NewDeployer(ct)
if err != nil {
    continue
}

// wizard.go — every catalog install error discarded
_ = s.AddServer(def)

// user_config.go — corrupt config falls back to defaults
_ = yaml.Unmarshal(data, &cfg)
```

### Rule

- Never assign an error to `_` unless the function's godoc explicitly documents that the error is informational-only (e.g., `fmt.Fprintf` to a `bytes.Buffer`).
- Never `continue` past an error without distinguishing expected errors (like `AlreadyExists`) from unexpected ones.
- If a function can fail, its caller must handle the error or propagate it. No exceptions.
- Parse errors on user-facing config files must halt with a clear message. Silent fallback to defaults is a bug.

---

## 2. File I/O Safety

### Anti-pattern: Non-atomic writes

v1 uses `os.WriteFile` directly on live config files. A crash mid-write truncates the file. The registry, project store, deployer configs, and settings deployer all share this flaw.

```go
// registry.go — truncates then writes; crash = empty file
os.WriteFile(path, data, 0o644)
```

### Rule

- All writes to persistent config files must be atomic: write to a temp file in the same directory, `fsync`, then `os.Rename` over the target.
- Extract this into a single `atomicWrite(path string, data []byte, perm os.FileMode) error` helper. Every config writer calls it.
- Backup must be taken *before* any mutation (including `Bootstrap`), not after.

```go
func atomicWrite(path string, data []byte, perm os.FileMode) error {
    dir := filepath.Dir(path)
    tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp.*")
    if err != nil {
        return err
    }
    defer func() {
        if err != nil {
            _ = os.Remove(tmp.Name()) // clean up on failure
        }
    }()
    if _, err = tmp.Write(data); err != nil {
        return err
    }
    if err = tmp.Sync(); err != nil {
        return err
    }
    if err = tmp.Close(); err != nil {
        return err
    }
    return os.Rename(tmp.Name(), path)
}
```

---

## 3. Encapsulation

### Anti-pattern: Exposing internal state

`Store.Items()` returns the live backing map. `Store.Projects` is an exported map that callers mutate directly, bypassing any validation.

```go
// store.go — returns the live map
func (s *Store[T, PT]) Items() map[string]T {
    return s.items
}

// service.go — direct mutation of store internals
s.projects.Projects[projectName] = proj
```

### Rule

- Internal collections are unexported. Access is through methods only.
- `Items()` returns a shallow copy, never the backing map.
- All mutations go through store methods (`Add`, `Update`, `Delete`). If a method doesn't exist, add one — don't reach into the struct.
- If a type has invariants (e.g., names must be unique, references must be valid), enforce them at the write boundary, not at read time.

---

## 4. Validation at Write Time

### Anti-pattern: Deferred validation

v1 accepts invalid data at write time and only errors at use time. Tags can reference non-existent servers. Permissions accept arbitrary type strings. Transports are unvalidated strings. Hook timeouts can be negative.

```go
// registry.go — no server existence check
func (r *Registry) AddTag(name string, servers []string) error {
    r.Tags[name] = servers
    return nil
}
```

### Rule

- Validate at the boundary where data enters the system. `Add`, `Update`, `SetItems`, and YAML/JSON unmarshal paths must reject invalid data immediately.
- Define string enums as typed constants with a `Valid()` method. Never use bare `string` for a field with a known set of legal values.

```go
type Transport string

const (
    TransportStdio Transport = "stdio"
    TransportSSE   Transport = "sse"
    TransportHTTP  Transport = "http"
)

func (t Transport) Valid() bool {
    switch t {
    case TransportStdio, TransportSSE, TransportHTTP:
        return true
    }
    return false
}
```

- Dangling references (tags referencing missing servers, assignments referencing missing resources) are write-time errors, not read-time errors.

---

## 5. Equality and Comparison

### Anti-pattern: nil vs empty mismatch

`ServerDef.Equal` uses `slices.Equal` and `maps.Equal`, which treat `nil` and empty as different. YAML deserialization produces `nil` for absent fields; code construction produces empty. This creates false-positive drift on every sync.

### Rule

- Define a `normalizeNil` step or use comparison helpers that treat `nil` and empty as equivalent.
- Add a test for every `Equal` method that explicitly asserts `nil` and empty compare as equal.

```go
func slicesEqualNil[T comparable](a, b []T) bool {
    if len(a) == 0 && len(b) == 0 {
        return true
    }
    return slices.Equal(a, b)
}
```

---

## 6. Side Effects in Read Paths

### Anti-pattern: Mutations on read-only operations

`PersistentPreRunE` runs `DiscoverAndImport` before every subcommand, including `list`, `diff`, and `version`. Read-only commands silently mutate the registry.

### Rule

- Read-only commands must not have write side-effects.
- Structure CLI hooks so that mutation-triggering setup (auto-discover, auto-sync) only runs for commands that explicitly opt in, not via `PersistentPreRunE`.
- If auto-discovery is needed, gate it: `if cmd.Annotations["mutates"] == "true"`.

---

## 7. Bubble Tea Patterns

### Anti-pattern: Synchronous I/O in Update/Init

v1 performs file I/O synchronously inside model constructors and `Update` handlers, blocking the event loop and freezing the TUI.

```go
// diff.go — blocks event loop
func NewDiffModel(...) DiffModel {
    m.loadDiff() // synchronous file I/O
    return m
}

// diff.go — sync on keypress blocks
case "s":
    _, err := m.service.SyncProject(m.projectName)
```

### Rule

- Never perform I/O in `Update` or in a model constructor.
- All I/O goes in `tea.Cmd` functions. Deliver results via `tea.Msg`.
- Constructors return a model in a "loading" state. `Init()` returns a `tea.Cmd` that kicks off the async work.

```go
func NewDiffModel(svc *service.Service, project string) DiffModel {
    return DiffModel{service: svc, projectName: project, loading: true}
}

func (m DiffModel) Init() tea.Cmd {
    return func() tea.Msg {
        diff, err := m.service.Diff(m.projectName)
        return diffLoadedMsg{diff: diff, err: err}
    }
}
```

### Anti-pattern: Bare type assertion on `tea.Program.Run()`

`Run()` can return an unexpected model type if the program exits abnormally (ctrl+c, I/O error). A bare assertion panics.

```go
wr := result.(wizardWrapper) // panic on abnormal exit
```

### Rule

- Always use two-value type assertions on `Run()` results.

```go
wr, ok := result.(wizardWrapper)
if !ok {
    return nil
}
```

---

## 8. Bubble Tea Update Delegation

### Anti-pattern: Monolithic `Update()` switch

A single `Update()` function that handles all key presses, window events, and custom messages grows to 200+ lines as features are added. Every new tab or mode adds more cases to the same switch block.

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyPressMsg:
        switch msg.String() {
        case "ctrl+c", "q":
            return m, tea.Quit
        case "tab":
            m.activeTab = (m.activeTab + 1) % len(m.tabs)
        }
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
    case myCustomMsg:
        m.data = msg.payload
    }
    return m, nil
}
```

### Rule

- **Delegate to sub-updaters**: Extract each message type into its own handler method (`handleKey`, `handleResize`) so handlers are testable in isolation.
- **Route keys through a keymap**: Use `key.Binding` to decouple key definitions from behavior, enabling user rebinding without touching handler logic.
- **Component delegation for tabs**: Each tab should implement `tea.Model`. The root `Update` handles global keys and resize, then forwards to the active tab. The root switch stays small regardless of feature count.

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Global handling first
    if msg, ok := msg.(tea.KeyPressMsg); ok {
        if key.Matches(msg, m.keys.Quit) {
            return m, tea.Quit
        }
    }

    // Resize propagates to all components
    if msg, ok := msg.(tea.WindowSizeMsg); ok {
        m.width = msg.Width
        m.height = msg.Height
    }

    // Delegate to active tab
    var cmd tea.Cmd
    m.tabs[m.activeTab], cmd = m.tabs[m.activeTab].Update(msg)
    return m, cmd
}
```

---

## 9. Key Bindings

### Anti-pattern: Context-dependent key collisions

In the importer's conflict view, `k` is bound to both "keep existing" (destructive action) and vim-up (navigation). A vim user pressing `k` to scroll silently applies "keep" with no confirmation.

### Rule

- Destructive or state-changing actions must not share keys with navigation in any mode.
- Document all key bindings per-mode in a single source of truth. When adding a binding, check for collisions against every active mode that shares the view.
- Destructive single-key actions (delete, apply, skip) should require either a confirmation step or a modifier key (`ctrl+`, `alt+`).

---

## 9. CLI Interactive I/O Separation

### Anti-pattern: Inline interactive prompts with business logic

The import command mixes `bufio.Reader` prompts, `fmt.Fprintf` output, and resolution assignment in one procedural block. This makes the conflict resolution flow untestable without stdin/stdout mocking, impossible to reuse from the TUI, and fragile to extend (adding a new resolution option means editing deep inside a loop).

```go
reader := bufio.NewReader(cmd.InOrStdin())
for i, c := range candidates {
    if !c.Conflict {
        continue
    }
    _, _ = fmt.Fprintf(out, "\nConflict: %q already exists in registry.\n", c.Name)
    _, _ = fmt.Fprintln(out, "  [k]eep existing  [r]eplace  [n]ame (rename)  [s]kip")
    _, _ = fmt.Fprint(out, "  Choice: ")

    input, err := reader.ReadString('\n')
    // ... switch on input, mutate candidates[i] ...
}
```

### Problems

1. **Untestable** — requires stdin/stdout faking to exercise conflict resolution paths.
2. **Not reusable** — the TUI has its own conflict resolution UI, duplicating the resolution logic.
3. **Violates SRP** — one function handles presentation, input parsing, validation, and state mutation.
4. **No input validation** — invalid input silently falls through to `ImportSkip` via the `default` case with no feedback to the user.

### Rule

- **Separate resolution gathering from resolution application.** The service layer accepts a slice of candidates with resolutions already set. The CLI and TUI each gather resolutions in their own way, then call the same `ApplyImport`.
- **Extract a `Resolver` interface** for conflict resolution:

```go
type ConflictResolver interface {
    Resolve(candidate ImportCandidate) (ImportResolution, string) // resolution, rename
}

// CLI implementation
type CLIResolver struct {
    reader *bufio.Reader
    out    io.Writer
}

// TUI implementation
type TUIResolver struct { /* uses overlay model */ }
```

- **Validate input in a loop** — re-prompt on invalid input rather than silently defaulting:

```go
func (r *CLIResolver) Resolve(c ImportCandidate) (ImportResolution, string) {
    for {
        fmt.Fprintf(r.out, "\nConflict: %q already exists.\n", c.Name)
        fmt.Fprintln(r.out, "  [k]eep  [r]eplace  [n]ame  [s]kip")
        fmt.Fprint(r.out, "  Choice: ")

        input, _ := r.reader.ReadString('\n')
        switch strings.TrimSpace(strings.ToLower(input)) {
        case "k", "keep":
            return ImportKeep, ""
        case "r", "replace":
            return ImportReplace, ""
        case "n", "name", "rename":
            fmt.Fprint(r.out, "  New name: ")
            name, _ := r.reader.ReadString('\n')
            return ImportRename, strings.TrimSpace(name)
        case "s", "skip":
            return ImportSkip, ""
        default:
            fmt.Fprintln(r.out, "  Invalid choice, try again.")
        }
    }
}
```

- **Test the resolver independently** — pass a `strings.Reader` as stdin, assert resolution outputs without involving the full import pipeline.

---

## 10. Deterministic Output

### Anti-pattern: Map iteration for ordered output

`buildProfile` iterates Go maps to build slices, producing non-deterministic ordering that causes spurious diffs when comparing saved profiles.

```go
for name, sel := range m.mcpSelections {
    if sel {
        p.MCPs = append(p.MCPs, name)
    }
}
```

### Rule

- Never iterate a map to build a user-visible or persisted slice.
- Collect keys, sort, then iterate in sorted order.
- If a helper like `selectedNames` already exists for this, use it everywhere.

---

## 10. Cleanup Symmetry

### Anti-pattern: Add path works, remove path doesn't

`SyncSettings` writes hooks and permissions when they exist but does nothing when they're removed. The old values persist in `settings.local.json` forever.

### Rule

- Every "write" operation must have a corresponding "remove" operation that cleans up when the resource set becomes empty.
- Test the full lifecycle: add resources, verify they exist, remove all resources, verify the config file is clean.

---

## 11. `os.Stat` Error Handling

### Anti-pattern: Only checking `IsNotExist`

```go
if _, err := os.Stat(p); os.IsNotExist(err) {
    // create file
}
// permission errors, I/O errors silently ignored
```

### Rule

- Always handle the three-way stat result: exists, does not exist, or error.

```go
_, err := os.Stat(p)
switch {
case err == nil:
    // exists
case errors.Is(err, fs.ErrNotExist):
    // does not exist — create
default:
    return err // propagate unexpected errors
}
```

---

## 12. Dead Code

### Anti-pattern: Unused constructors, redundant helpers, duplicate logic

v1 has: unused error constructors in `errors.go`, a `newTeaProgram` helper called from one place, a hand-rolled integer parser instead of `strconv.Atoi`, `contains` helpers that reimplement `strings.Contains`, and a local `max` function that shadows the Go 1.21 builtin.

### Rule

- Use stdlib functions (`strconv.Atoi`, `strings.Contains`, `max`, `min`) instead of hand-rolling.
- Delete dead code. Don't comment it out, don't keep it "for later." If it's needed later, git has it.
- Run `staticcheck` and `deadcode` in CI. Unused exports are a warning; unused unexported symbols are an error.

---

## 13. Test Coverage Gaps

### Anti-pattern: Tests that don't cover the full surface

- `paths_test.go` asserts 4 of 5 subdirectories, missing `prompts`.
- No tests for nil-vs-empty equality edge cases.
- No tests for the remove/cleanup path of settings sync.
- Wizard tests don't verify error propagation from catalog installs.

### Rule

- When adding a new item to a set (e.g., a new subdirectory), update the corresponding test in the same commit.
- Every `Equal` method gets a test case for nil-vs-empty.
- Every sync operation gets a test for the full lifecycle: create, update, remove, verify cleanup.
- Error paths are tested, not just happy paths. If a function can return an error, at least one test must trigger it.

---

## 14. String Parsing

### Anti-pattern: Fragile delimiter-based parsing

`parseKVPairs` splits on `,` then `=`. A value containing a comma (`DATABASE_URL=host?a=1,b=2`) is silently truncated.

### Rule

- For user-facing input formats, use an unambiguous delimiter or a proper parser.
- If comma-separated key=value pairs are needed, either escape commas or switch to newline-delimited input.
- Add test cases for values containing the delimiter character.
