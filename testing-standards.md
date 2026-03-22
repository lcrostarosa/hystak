# Testing Standards

Anti-patterns found in hystak v1's test suite and the rules that replace them for the rebuild.

---

## 1. Fixture Factories

### Anti-pattern: Near-duplicate fixture helpers per file

v1 has at least five separate fixture factories that build overlapping but slightly different `*service.Service` instances:

- `testService()` in `mcps_test.go` — 2 servers, 1 project
- `testProjectService()` in `profiles_test.go` — 3 servers, 2 projects
- `testDiffService()` in `diff_test.go` — servers + deployed state
- `testImportService()` in `importer_test.go` — servers for import scenarios
- `setupService()` in `crud_test.go` — 3 servers, 1 project with tags

Each factory hardcodes its own registry, project store, and server definitions inline. When a field is added to `ServerDef` or the store shape changes, every factory must be updated independently. Tests using different factories make different assumptions about what "default state" looks like.

Worse, when a test needs a small variation (e.g., adding a tag), it abandons the shared factory entirely and rebuilds everything from scratch — 35 lines of boilerplate for one behavioral difference:

```go
// mcps_test.go:242 — full inline rebuild just to add one tag
func TestDeleteRefusedByTag(t *testing.T) {
    reg := &registry.Registry{
        Servers:     registry.NewStore[model.ServerDef, *model.ServerDef]("server"),
        Skills:      registry.NewStore[model.SkillDef, *model.SkillDef]("skill"),
        Hooks:       registry.NewStore[model.HookDef, *model.HookDef]("hook"),
        Permissions: registry.NewStore[model.PermissionRule, *model.PermissionRule]("permission"),
        Templates:   registry.NewStore[model.TemplateDef, *model.TemplateDef]("template"),
        Prompts:     registry.NewStore[model.PromptDef, *model.PromptDef]("prompt"),
        Tags: map[string][]string{
            "core": {"github"},
        },
    }
    _ = reg.Servers.Add(model.ServerDef{ ... })
    _ = reg.Servers.Add(model.ServerDef{ ... })
    store := &project.Store{ ... }
    svc := service.NewForTest(reg, store, nil, nil, "", nil)
    // ...
}
```

### Rule

- One canonical fixture builder per package, exported from a `testutil_test.go` or `internal/testutil/` package.
- The builder returns a fully constructed default scenario. Callers customize it through functional options or by mutating the returned struct before use — never by rebuilding from scratch.
- Every fixture factory must check errors from `Add` and similar calls. Never `_ =` in test setup.

```go
// internal/testutil/fixtures.go (shared across packages)
type TestFixture struct {
    Registry *registry.Registry
    Store    *project.Store
    Deployer *MockDeployer
    Service  *service.Service
}

type FixtureOption func(*TestFixture)

func WithTag(name string, servers []string) FixtureOption {
    return func(f *TestFixture) {
        f.Registry.Tags[name] = servers
    }
}

func WithServer(s model.ServerDef) FixtureOption {
    return func(f *TestFixture) {
        if err := f.Registry.Servers.Add(s); err != nil {
            panic("test fixture: " + err.Error())
        }
    }
}

func NewFixture(t *testing.T, opts ...FixtureOption) *TestFixture {
    t.Helper()
    f := &TestFixture{ /* default servers, projects */ }
    for _, opt := range opts {
        opt(f)
    }
    return f
}
```

Usage:

```go
func TestDeleteRefusedByTag(t *testing.T) {
    f := testutil.NewFixture(t, testutil.WithTag("core", []string{"github"}))
    // test only the interesting part
}
```

---

## 2. Error Handling in Test Setup

### Anti-pattern: Discarding errors from setup operations

Every fixture factory discards `Add` errors with `_ =`:

```go
// mcps_test.go:24
_ = reg.Servers.Add(model.ServerDef{
    Name: "github",
    // ...
})
```

If `Add` ever starts returning errors for new validation rules (like the transport validation recommended in `coding-standards.md`), every fixture will silently produce an empty registry and every downstream assertion will fail with misleading error messages about "expected 2 items, got 0."

### Rule

- Test setup must never discard errors. Use `t.Fatal` or `t.Helper` + panic for setup operations that cannot fail in a correct test.
- If a setup function returns an error, use a `must` wrapper:

```go
func must(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatal(err)
    }
}

// usage
must(t, reg.Servers.Add(server))
```

---

## 3. Round-Trip Assertions

### Anti-pattern: Partial field comparison after marshal/unmarshal

YAML round-trip tests compare only some fields, missing others entirely:

```go
// server_test.go:53 — HTTP round-trip only checks URL and Transport
if s2.URL != s.URL || s2.Transport != s.Transport {
```

`Headers` is not compared. A bug that strips headers during YAML marshal/unmarshal would pass this test. The stdio round-trip similarly omits `Env` from the re-unmarshalled value.

`ServerOverride` round-trip tests exercise `Command`, `Args`, and `Env` but skip `URL` and `Headers`.

### Rule

- Round-trip tests must compare the full struct, not individual fields.
- Use `reflect.DeepEqual` or a dedicated comparison function (like `ServerDef.Equal`, once it handles nil/empty correctly).
- If `Equal` is defined on the type, the round-trip test is the place to exercise it:

```go
func TestServerDefYAMLRoundTrip(t *testing.T) {
    original := model.ServerDef{
        Name:      "test",
        Transport: model.TransportHTTP,
        URL:       "http://example.com",
        Headers:   map[string]string{"Auth": "Bearer tok"},
    }
    data, err := yaml.Marshal(original)
    must(t, err)

    var restored model.ServerDef
    must(t, yaml.Unmarshal(data, &restored))

    if !original.Equal(restored) {
        t.Errorf("round-trip mismatch:\n  got:  %+v\n  want: %+v", restored, original)
    }
}
```

- Add explicit round-trip tests for every serialized type, covering every field including optional/pointer fields.

---

## 4. Table-Driven Tests

### Anti-pattern: Copy-paste tests that differ by one variable

Multiple test functions with near-identical setup that differ by a single condition:

```go
// skills_test.go — two functions, same setup, one uses a regular file, one a symlink
func TestPreflightSkills_ConflictForRegularFile(t *testing.T) { /* 23 lines */ }
func TestPreflightSkills_NoConflictForSymlink(t *testing.T)   { /* 27 lines */ }

// crud_test.go — four structurally identical CRUD tests
func TestAddSkill_CRUD(t *testing.T)      { /* add, get, update, delete */ }
func TestAddHook_CRUD(t *testing.T)       { /* add, get, update, delete */ }
func TestAddPermission_CRUD(t *testing.T) { /* add, get, update, delete */ }
func TestAddTemplate_CRUD(t *testing.T)   { /* add, get, update, delete */ }
```

### Rule

- When two or more test functions share the same structure and differ only in inputs/expectations, use a table-driven test.
- Each table entry must have a `name` field used in `t.Run`.
- The table should make the varying dimension obvious at a glance:

```go
func TestPreflightSkills(t *testing.T) {
    tests := []struct {
        name       string
        setup      func(t *testing.T, dir string) // creates the file/symlink
        wantConflict bool
    }{
        {"regular file", createRegularFile, true},
        {"symlink", createSymlink, false},
        {"dangling symlink", createDanglingSymlink, false},
        {"no file", func(*testing.T, string) {}, false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            dir := t.TempDir()
            tt.setup(t, dir)
            conflicts := deployer.PreflightSkills(dir, skills)
            if (len(conflicts) > 0) != tt.wantConflict {
                t.Errorf("conflict = %v, want %v", len(conflicts) > 0, tt.wantConflict)
            }
        })
    }
}
```

---

## 5. Mega-Tests

### Anti-pattern: Sequential lifecycle in a single test function

```go
// profile_test.go — one test does Save, Get, Update, Delete in sequence
func TestCRUDRoundTrip(t *testing.T) {
    // Save
    // ... assert ...
    // Get
    // ... assert ...
    // Update
    // ... assert ...
    // Delete
    // ... assert ...
}
```

If `Save` fails, the remaining 3/4 of the test is dead. `go test -v` output says "TestCRUDRoundTrip FAIL" with no indication of which operation broke. CI bisect and failure triage are harder because one function covers four behaviors.

### Rule

- One behavior per test function. Each test should set up its own preconditions.
- If operations must run in sequence (e.g., save then load), that's fine within a single test — but the test name should describe the *behavior under test*, not the sequence of operations.
- If a test function has more than one conceptual assertion block separated by a setup step, it should be split:

```go
func TestProfileSave(t *testing.T)                { /* save + verify on disk */ }
func TestProfileGet(t *testing.T)                  { /* pre-create + get */ }
func TestProfileUpdate(t *testing.T)               { /* pre-create + update + verify */ }
func TestProfileDelete(t *testing.T)               { /* pre-create + delete + verify gone */ }
```

---

## 6. Test Naming

### Anti-pattern: Inconsistent naming across packages

v1 uses at least three naming styles:

```
TestVanilla                                    // profile_test.go — no context
TestCRUDRoundTrip                              // profile_test.go — describes method, not behavior
TestClaudeCodeDeployer_ReadServers_MissingFile  // claude_code_test.go — structured
TestAddServer                                  // crud_test.go — verb only
TestDeleteConfirmExecute                       // mcps_test.go — action sequence
```

### Rule

- Use the `Test<Subject>_<Scenario>` pattern consistently:

```
Test<Type or Function>_<Condition or Behavior>
```

Examples:

```
TestServerDef_YAMLRoundTrip_HTTP
TestServerDef_Equal_NilVsEmptySlice
TestStore_Add_Duplicate
TestStore_Delete_NotFound
TestSyncProject_PreservesUnmanagedServers
TestPreflightSkills_ConflictsOnRegularFile
```

- The name should describe what is being tested and under what condition, not the steps being performed.
- `TestValidateEmptyName` appearing in two packages is ambiguous in `go test -run` output. Prefix with the subject: `TestProfile_ValidateEmptyName`, `TestForm_ValidateEmptyName`.

---

## 7. Bubble Tea TUI Testing

### Anti-pattern: `time.Sleep` in teatest tests

```go
// launch_wizard_teatest_test.go:71-74
for i := 0; i < 8; i++ {
    tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
    time.Sleep(50 * time.Millisecond) // small delay for message processing
}
```

`time.Sleep` in tests is a flakiness source. On a loaded CI machine, 50ms may not be enough. On a fast machine, it's wasted time (400ms total here). The sleep is a workaround for not waiting for the model to process each message before sending the next.

### Rule

- Never use `time.Sleep` in teatest tests.
- After sending a message that changes visible state, use `teatest.WaitFor` with a content predicate before sending the next message.
- If you need to advance through N steps, wait for each step's expected output before sending the next key:

```go
steps := []struct {
    key      tea.KeyMsg
    waitFor  string
}{
    {enterKey, "Skills"},
    {enterKey, "Permissions"},
    {enterKey, "Hooks"},
    // ...
}
for _, step := range steps {
    tm.Send(step.key)
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return strings.Contains(string(bts), step.waitFor)
    }, teatest.WithDuration(2*time.Second))
}
```

### Anti-pattern: `wizardTestModel` / `formTestModel` wrappers

v1 needs adapter structs because `LaunchWizardModel.Update` and `FormModel.Update` return concrete types instead of `tea.Model`:

```go
// teatest_helpers_test.go:37 — adapter because Update returns concrete type
func (w wizardTestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    w.inner, cmd = w.inner.Update(msg)
    return w, cmd
}
```

### Rule

- All Bubble Tea models must return `tea.Model` from `Update`, not their concrete type. This eliminates the need for test adapters and is the standard Bubble Tea convention.
- If a model is a child component embedded in a parent, the parent does the concrete type assertion in its own `Update`. The child still returns `tea.Model`.

### Anti-pattern: Golden file tests with no documented update mechanism

Every teatest integration test ends with `teatest.RequireEqualOutput(t, out)`, which compares against golden files. There is no Makefile target, no documented flag, and no CI step for regenerating golden files when the UI changes legitimately.

### Rule

- Document the golden file update command in CLAUDE.md and at the top of the first teatest file:

```go
// To update golden files after intentional UI changes:
//   UPDATE_GOLDEN=1 go test ./internal/tui/...
```

- Add a Makefile target: `make test-update`.
- Golden file tests must be in files named `*_teatest_test.go` so they can be run or skipped as a group.
- Never create a golden file test for volatile output (timestamps, random IDs, system-dependent paths).

---

## 8. Idempotency Assertions

### Anti-pattern: Using `ModTime` to detect unnecessary recreation

```go
// skills_test.go:205 — compares modification times
if !info1.ModTime().Equal(info2.ModTime()) {
    t.Error("symlink was unnecessarily recreated")
}
```

On filesystems with 1-second mtime resolution (HFS+, some ext4 configs), two symlink operations within the same second produce identical `ModTime` regardless of whether the symlink was recreated. The assertion passes even when the code is wrong.

### Rule

- Use inode comparison (`os.SameFile`) to detect whether a file/symlink was recreated, not `ModTime`:

```go
info1, _ := os.Lstat(path)
// ... run sync again ...
info2, _ := os.Lstat(path)

if !os.SameFile(info1, info2) {
    t.Error("symlink was unnecessarily recreated")
}
```

- `os.SameFile` compares device + inode, which changes when a file is deleted and recreated, even within the same second.

---

## 9. Testing Internal State vs Behavior

### Anti-pattern: Directly accessing unexported fields

```go
// form_test.go:524
app.form.editName = "github"  // bypasses the form's initialization pathway

// mcps_test.go:176-177
m.confirming = true  // sets internal flag directly
```

These tests couple to the implementation's field layout. If `confirming` is renamed to `deleteMode` or `editName` is moved into a sub-struct, the tests break without any behavioral change. More importantly, setting internal state directly skips the code path that normally sets it, so the test may pass even when that code path is broken.

### Rule

- Drive state changes through the public API: send messages, call exported methods.
- Assert on observable outputs: `View()` content, returned `tea.Cmd`, or method return values.
- If a test needs to check internal state (e.g., "is the model in confirming mode?"), expose it through a getter or assert on its observable effect (like the status help text changing):

```go
// Instead of: m.confirming = true
m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

// Instead of: if m.confirming { ... }
help := m.StatusHelp()
if !strings.Contains(help, "y: confirm") {
    t.Error("expected confirming mode")
}
```

### Anti-pattern: Testing constant values

```go
// project_test.go:248 — asserts that DriftSynced == "synced"
{DriftSynced, "synced"},
```

This tests that a constant has its hardcoded value — it can never fail unless someone changes the constant, which the compiler would catch at all usage sites anyway. It adds no behavioral coverage.

### Anti-pattern: Change-detector tests

```go
// conflict_test.go:20 — fails whenever a new conflict type is added
if len(conflictDescriptions) != 4 {
    t.Errorf("expected 4 entries, got %d", len(conflictDescriptions))
}
```

This forces a test update every time a conflict type is added, but catches no actual bugs. If someone adds a type without adding a description, this test doesn't help — it just forces them to bump the count.

### Rule

- Test behavior, not implementation details.
- Instead of asserting map sizes, assert that every known enum value has a corresponding entry:

```go
func TestConflictDescriptions_AllTypesPresent(t *testing.T) {
    for _, rt := range allResourceTypes {
        if _, ok := conflictDescriptions[rt]; !ok {
            t.Errorf("missing description for resource type %q", rt)
        }
    }
}
```

- Instead of asserting constant string values, test how the constant is used: serialization round-trips, comparison behavior, display formatting.

---

## 10. Flakiness Prevention

### Anti-pattern: Assumptions about system state

```go
// isolation_test.go:263 — assumes PID 99999999 doesn't exist
_ = os.WriteFile(lockFile, []byte("99999999"), 0o644)
```

This test assumes the PID 99999999 is never running. On systems with different PID limits, `kill(pid, 0)` returns different errors for out-of-range PIDs than for valid-but-unused PIDs.

### Rule

- Never assume a specific PID, port, or system resource is available.
- For stale-PID testing, fork a subprocess, wait for it to exit, then use its (now-dead) PID:

```go
func deadPID(t *testing.T) int {
    t.Helper()
    cmd := exec.Command("true")
    must(t, cmd.Run())
    return cmd.Process.Pid // guaranteed dead
}
```

- For port-based tests, use `:0` to get a random available port.

### Anti-pattern: Hardcoded paths

```go
// mcps_test.go:43
Path: "/tmp/myproject",
```

On macOS `/tmp` is a symlink to `/private/tmp`. If any code resolves symlinks (e.g., the worktree manager does), path comparisons will fail. Tests that don't touch the filesystem should use a clearly fake path that can't be confused with a real one.

### Rule

- For tests that touch the filesystem: always use `t.TempDir()`.
- For tests that only store paths as data (never read/write): use obviously fake paths like `/test/project` and document that these are identity strings, not real paths.

---

## 11. Integration Test Boundaries

### Anti-pattern: Unmarked integration tests mixed with unit tests

```go
// cli_test.go:114 — TestSyncCommand writes real files, runs full CLI→service→deployer pipeline
func TestSyncCommand(t *testing.T) {
    dir := t.TempDir()
    // ... creates real files, exercises the full stack
}
```

There is no `testing.Short()` guard, no build tag, and no naming convention to distinguish this from unit tests. When this test is slow or flaky, there is no way to skip it independently.

### Rule

- Use `testing.Short()` to skip integration tests in fast feedback loops:

```go
func TestSyncCommand_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    // ...
}
```

- Name integration tests with an `_Integration` suffix.
- Add Makefile targets:

```makefile
test:       go test -short ./...
test-all:   go test ./...
test-e2e:   go test -run Integration ./...
```

---

## 12. Mock Contracts

### Anti-pattern: Mocks that don't match production contracts

```go
// service_test.go:58 — mockDeployer.Bootstrap initializes an in-memory map
func (m *mockDeployer) Bootstrap(projectPath string) error {
    if m.servers[projectPath] == nil {
        m.servers[projectPath] = make(map[string]model.ServerDef)
    }
    return nil
}
```

The real `Bootstrap` creates a file on disk. Any service code that calls `Bootstrap` then checks for the file (or reads its content) will behave differently in tests vs production. The mock satisfies the interface but not the behavioral contract.

### Rule

- Mocks must document which aspects of the contract they cover and which they don't.
- If the interface has filesystem side-effects, the mock should either use `t.TempDir()` to produce real files, or the test should use a fake filesystem abstraction.
- Compile-time interface checks belong in production code, not test files:

```go
// deployer.go (production code)
var _ Deployer = (*ClaudeCodeDeployer)(nil)
```

Not:

```go
// resource_deployer_test.go (test code)
var _ ResourceDeployer = (*SkillsDeployer)(nil)
```

Placing them in test files means they only catch violations when tests are compiled.

---

## 13. Assertion Quality

### Anti-pattern: Fragile string matching on rendered output

```go
// logo_test.go:12 — breaks if the logo art changes
if !strings.Contains(logo, "|___/") {

// diff_test.go:271 — passes even if colorizeDiff returns input unchanged
// only checks that raw strings "old", "new", "context" appear
```

### Rule

- Test the semantic content, not the rendering artifact. If you're testing a logo, assert it's non-empty and contains the product name. If you're testing coloring, assert that the output differs from the input, or test the ANSI codes directly.
- When using `strings.Contains` for view assertions, always verify a string that could *only* appear in the expected state, not a generic word:

```go
// Bad: "new" could appear in many contexts
if !strings.Contains(output, "new") {

// Good: only appears in the add-line coloring path
if !strings.Contains(output, "\x1b[32m+new") {
```

### Anti-pattern: Reimplemented standard library functions

```go
// claude_md_test.go:480 — hand-rolled strings.Contains
func contains(s string, substrs ...string) bool {
    for _, sub := range substrs {
        if !containsSubstr(s, sub) {
            return false
        }
    }
    return true
}
```

### Rule

- Never reimplement `strings.Contains`, `slices.Contains`, `maps.Keys`, etc. in test code. Use the standard library.

---

## 14. Cleanup Patterns

### Anti-pattern: Inconsistent `t.Cleanup` usage

```go
// isolation_test.go:130-131 — errors dropped in setup, cleanup registered regardless
_, _ = wm.Create(dir, "alpha")
_, _ = wm.Create(dir, "beta")

// TestWorktreeRemove performs cleanup inline, no t.Cleanup safety net
```

Some tests use `t.Cleanup`, others do inline cleanup. Some register cleanup for resources that might not have been created. Some drop setup errors entirely.

### Rule

- Always use `t.Cleanup` for resources that must be released, never rely on inline cleanup after assertions (which may `t.Fatal` before reaching the cleanup line).
- Register cleanup *after* confirming the resource was created:

```go
wtPath, err := wm.Create(dir, "alpha")
if err != nil {
    t.Fatal(err)
}
t.Cleanup(func() { _ = wm.Remove(dir, "alpha") })
```

- Prefer `t.TempDir()` (auto-cleaned) over manual directory creation + cleanup.

---

## 15. Subprocess Tests

### Anti-pattern: Subprocess output leaking to real stdout

```go
// launch_test.go:95
// Just verify it doesn't error — stdout goes to os.Stdout which is fine in tests.
```

This pollutes `go test -v` output with subprocess noise, making it hard to read test results.

### Rule

- Redirect subprocess output to `io.Discard` or a `bytes.Buffer` that can be asserted on:

```go
cmd.Stdout = io.Discard
cmd.Stderr = io.Discard
```

- If you need to verify subprocess output, capture it and assert:

```go
var buf bytes.Buffer
cmd.Stdout = &buf
must(t, cmd.Run())
if !strings.Contains(buf.String(), "expected output") {
    t.Errorf("unexpected output: %s", buf.String())
}
```

---

## 16. Test Coverage Gaps to Close

These specific gaps were found in v1 and must be covered in the rebuild:

| Area | Missing test |
|------|-------------|
| `ServerDef.Equal` | nil slice vs empty slice, nil map vs empty map |
| `WriteServers` | Called with empty/nil server map |
| `PermissionRule.EffectiveType` | Unknown type string, case sensitivity |
| `parseKVPairs` / `parseCSV` | Values containing the delimiter (commas in values) |
| `SyncSettings` | Remove all hooks/permissions → verify cleanup |
| `EnsureConfigDir` | `prompts` subdirectory (tested for 4/5 dirs, missed 1) |
| `LoadUserConfig` | Malformed YAML (parse error behavior) |
| `DriftReport` | Multi-client project (duplicate entry detection) |
| Symlink idempotency | Use `os.SameFile` not `ModTime` |
