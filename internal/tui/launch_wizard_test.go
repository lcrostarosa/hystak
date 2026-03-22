package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lcrostarosa/hystak/internal/discovery"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/profile"
)

func testDiscoveredItems() *discovery.Items {
	return &discovery.Items{
		MCPs: []discovery.DiscoveredMCP{
			{Name: "mcp-a", ServerDef: model.ServerDef{Transport: model.TransportStdio, Command: "a"}, Source: discovery.SourceRegistry},
			{Name: "mcp-b", ServerDef: model.ServerDef{Transport: model.TransportStdio, Command: "b"}, Source: discovery.SourceGlobal},
			{Name: "mcp-c", ServerDef: model.ServerDef{Transport: model.TransportSSE, URL: "http://c"}, Source: discovery.SourceProject},
		},
		Skills: []discovery.DiscoveredSkill{
			{Name: "skill-1", Description: "First skill", Source: discovery.SourceGlobal},
			{Name: "skill-2", Description: "Second skill", Source: discovery.SourceProject},
		},
		Permissions: []discovery.DiscoveredPermission{
			{Name: "perm-allow-read", Rule: "Read(*)", Type: "allow", Source: discovery.SourceGlobal},
		},
		Hooks: []discovery.DiscoveredHook{
			{Name: "hook-pre-tool", Event: "PreToolUse", Command: "echo hello", Source: discovery.SourceGlobal},
		},
		EnvVars: []discovery.DiscoveredEnvVar{
			{Key: "FOO", Value: "bar", Source: discovery.SourceGlobal},
		},
	}
}

func testProject() *model.Project {
	return &model.Project{Name: "test-proj", Path: "/tmp/proj"}
}

func newTestWizard() LaunchWizardModel {
	return NewLaunchWizardModel(testProject(), LWModeSequential, testDiscoveredItems(), nil)
}

func sendKey(m LaunchWizardModel, key string) LaunchWizardModel {
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return m
}

func sendSpecialKey(m LaunchWizardModel, kt tea.KeyType) LaunchWizardModel {
	m, _ = m.Update(tea.KeyMsg{Type: kt})
	return m
}

// advanceToChecklist walks through all sequential steps to reach the checklist phase.
func advanceToChecklist(m LaunchWizardModel) LaunchWizardModel {
	for i := 0; i < int(launchStepCount); i++ {
		m = sendSpecialKey(m, tea.KeyEnter)
	}
	return m
}

// --- Sequential mode tests ---

func TestLaunchWizardInitialStep(t *testing.T) {
	m := newTestWizard()
	if m.Step() != launchStepMCPs {
		t.Errorf("initial step = %d, want %d", m.Step(), launchStepMCPs)
	}
	if m.Phase() != phaseSteps {
		t.Errorf("initial phase = %d, want phaseSteps", m.Phase())
	}
}

func TestLaunchWizardStepForward(t *testing.T) {
	m := newTestWizard()
	m = sendSpecialKey(m, tea.KeyEnter)
	if m.Step() != launchStepSkills {
		t.Errorf("after enter step = %d, want %d", m.Step(), launchStepSkills)
	}
}

func TestLaunchWizardStepBackward(t *testing.T) {
	m := newTestWizard()
	m = sendSpecialKey(m, tea.KeyEnter) // → Skills
	m = sendSpecialKey(m, tea.KeyEsc)   // ← MCPs
	if m.Step() != launchStepMCPs {
		t.Errorf("after esc step = %d, want %d", m.Step(), launchStepMCPs)
	}
}

func TestLaunchWizardTabSkipsStep(t *testing.T) {
	m := newTestWizard()
	m = sendSpecialKey(m, tea.KeyTab) // skip MCPs → Skills
	if m.Step() != launchStepSkills {
		t.Errorf("after tab step = %d, want %d", m.Step(), launchStepSkills)
	}
}

func TestLaunchWizardToggleMCP(t *testing.T) {
	m := newTestWizard()
	// Toggle first MCP
	m = sendKey(m, " ")
	if !m.MCPSelections()["mcp-a"] {
		t.Error("expected mcp-a to be selected after space")
	}
	// Toggle again to deselect
	m = sendKey(m, " ")
	if m.MCPSelections()["mcp-a"] {
		t.Error("expected mcp-a to be deselected after second space")
	}
}

func TestLaunchWizardCursorNavigation(t *testing.T) {
	m := newTestWizard()
	m = sendKey(m, "j") // move down
	m = sendKey(m, " ") // toggle mcp-b
	if !m.MCPSelections()["mcp-b"] {
		t.Error("expected mcp-b to be selected")
	}
	if m.MCPSelections()["mcp-a"] {
		t.Error("expected mcp-a to NOT be selected")
	}
}

func TestLaunchWizardToggleAll(t *testing.T) {
	m := newTestWizard()
	m = sendKey(m, "a") // toggle all on
	for _, name := range []string{"mcp-a", "mcp-b", "mcp-c"} {
		if !m.MCPSelections()[name] {
			t.Errorf("expected %s to be selected after toggle all", name)
		}
	}
	m = sendKey(m, "a") // toggle all off
	for _, name := range []string{"mcp-a", "mcp-b", "mcp-c"} {
		if m.MCPSelections()[name] {
			t.Errorf("expected %s to be deselected after second toggle all", name)
		}
	}
}

func TestLaunchWizardSkillStep(t *testing.T) {
	m := newTestWizard()
	m = sendSpecialKey(m, tea.KeyEnter) // → Skills
	m = sendKey(m, " ")                 // toggle skill-1
	if !m.SkillSelections()["skill-1"] {
		t.Error("expected skill-1 to be selected")
	}
}

func TestLaunchWizardPreservesSelectionsOnBack(t *testing.T) {
	m := newTestWizard()
	m = sendKey(m, " ")                 // toggle mcp-a
	m = sendSpecialKey(m, tea.KeyEnter) // → Skills
	m = sendKey(m, " ")                 // toggle skill-1
	m = sendSpecialKey(m, tea.KeyEsc)   // ← MCPs
	if !m.MCPSelections()["mcp-a"] {
		t.Error("expected mcp-a selection to be preserved after going back")
	}
	m = sendSpecialKey(m, tea.KeyEnter) // → Skills again
	if !m.SkillSelections()["skill-1"] {
		t.Error("expected skill-1 selection to be preserved after going forward again")
	}
}

func TestLaunchWizardIsolationStep(t *testing.T) {
	m := newTestWizard()
	// Navigate to isolation step
	for i := 0; i < int(launchStepIsolation); i++ {
		m = sendSpecialKey(m, tea.KeyEnter)
	}
	if m.Step() != launchStepIsolation {
		t.Fatalf("expected isolation step, got %d", m.Step())
	}
	// Default is none
	if m.Isolation() != profile.IsolationNone {
		t.Errorf("expected default isolation 'none', got %s", m.Isolation())
	}
	// Move to worktree and select
	m = sendKey(m, "j") // move to worktree
	m = sendKey(m, " ") // select worktree
	if m.Isolation() != profile.IsolationWorktree {
		t.Errorf("expected isolation 'worktree', got %s", m.Isolation())
	}
}

func TestLaunchWizardSequentialToChecklist(t *testing.T) {
	m := newTestWizard()
	m = advanceToChecklist(m)
	if m.Phase() != phaseChecklist {
		t.Errorf("expected phaseChecklist after walking all steps, got %d", m.Phase())
	}
}

func TestLaunchWizardCompleteMsg(t *testing.T) {
	m := newTestWizard()
	m = sendKey(m, " ") // toggle mcp-a
	m = advanceToChecklist(m)

	// Now in checklist, press enter to launch
	var cmd tea.Cmd
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command from completing the wizard")
	}
	msg := cmd()
	complete, ok := msg.(LaunchWizardCompleteMsg)
	if !ok {
		t.Errorf("expected LaunchWizardCompleteMsg, got %T", msg)
	}
	if !complete.Launch {
		t.Error("expected Launch to be true")
	}
	if !containsString(complete.Profile.MCPs, "mcp-a") {
		t.Error("expected profile to contain mcp-a")
	}
}

func TestLaunchWizardCancelFromFirstStep(t *testing.T) {
	m := newTestWizard()
	var cmd tea.Cmd
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected a command from esc on first step")
	}
	msg := cmd()
	if _, ok := msg.(LaunchWizardCancelledMsg); !ok {
		t.Errorf("expected LaunchWizardCancelledMsg, got %T", msg)
	}
}

func TestLaunchWizardViewRenders(t *testing.T) {
	m := newTestWizard()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	view := m.View()
	if !strings.Contains(view, "MCPs") {
		t.Error("view should contain 'MCPs'")
	}
	if !strings.Contains(view, "mcp-a") {
		t.Error("view should contain 'mcp-a'")
	}
}

func TestLaunchWizardViewEachStep(t *testing.T) {
	m := newTestWizard()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})

	steps := []string{"MCPs", "Skills", "Permissions", "Hooks", "CLAUDE.md", "Env Vars", "Isolation"}
	for i, label := range steps {
		view := m.View()
		if !strings.Contains(view, label) {
			t.Errorf("step %d view should contain %q", i, label)
		}
		m = sendSpecialKey(m, tea.KeyEnter)
	}
}

func TestLaunchWizardEmptyDiscovery(t *testing.T) {
	m := NewLaunchWizardModel(testProject(), LWModeSequential, &discovery.Items{}, nil)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	view := m.View()
	if !strings.Contains(view, "No MCP servers discovered") {
		t.Error("empty discovery should show guidance message")
	}
}

func TestLaunchWizardWithExistingProfile(t *testing.T) {
	existing := &profile.Profile{
		MCPs:      []string{"mcp-a"},
		Skills:    []string{"skill-2"},
		EnvVars:   map[string]string{"MY_VAR": "val"},
		Isolation: profile.IsolationWorktree,
	}
	m := NewLaunchWizardModel(testProject(), LWModeSequential, testDiscoveredItems(), existing)
	if !m.MCPSelections()["mcp-a"] {
		t.Error("expected mcp-a pre-selected from existing profile")
	}
	if !m.SkillSelections()["skill-2"] {
		t.Error("expected skill-2 pre-selected from existing profile")
	}
	if m.Isolation() != profile.IsolationWorktree {
		t.Errorf("expected isolation worktree, got %s", m.Isolation())
	}
	if len(m.envKeys) != 1 || m.envKeys[0] != "MY_VAR" {
		t.Error("expected env vars pre-populated from existing profile")
	}
}

func TestLaunchWizardEnvVarAddDelete(t *testing.T) {
	m := newTestWizard()
	// Navigate to env vars step
	for i := 0; i < int(launchStepEnvVars); i++ {
		m = sendSpecialKey(m, tea.KeyEnter)
	}
	// Add an env var
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	if len(m.envKeys) != 1 {
		t.Fatalf("expected 1 env key after add, got %d", len(m.envKeys))
	}
	// Delete it
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	if len(m.envKeys) != 0 {
		t.Fatalf("expected 0 env keys after delete, got %d", len(m.envKeys))
	}
}

func TestLaunchWizardBuildProfile(t *testing.T) {
	m := newTestWizard()
	m.mcpSelections["mcp-a"] = true
	m.mcpSelections["mcp-c"] = true
	m.skillSelections["skill-1"] = true
	m.hookSelections["hook-pre-tool"] = true
	m.isolation = profile.IsolationLock

	p := m.buildProfile()
	if len(p.MCPs) != 2 {
		t.Errorf("expected 2 MCPs, got %d", len(p.MCPs))
	}
	if len(p.Skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(p.Skills))
	}
	if len(p.Hooks) != 1 {
		t.Errorf("expected 1 hook, got %d", len(p.Hooks))
	}
	if p.Isolation != profile.IsolationLock {
		t.Errorf("expected isolation lock, got %s", p.Isolation)
	}
}

func TestVisibleRange(t *testing.T) {
	tests := []struct {
		cursor, total, max int
		wantStart, wantEnd int
	}{
		{0, 5, 10, 0, 5},     // all visible
		{0, 20, 10, 0, 10},   // start at top
		{15, 20, 10, 10, 20}, // near bottom
		{10, 20, 10, 5, 15},  // in middle
		{19, 20, 10, 10, 20}, // at end
	}
	for _, tt := range tests {
		start, end := visibleRange(tt.cursor, tt.total, tt.max)
		if start != tt.wantStart || end != tt.wantEnd {
			t.Errorf("visibleRange(%d, %d, %d) = (%d, %d), want (%d, %d)",
				tt.cursor, tt.total, tt.max, start, end, tt.wantStart, tt.wantEnd)
		}
	}
}

func TestLaunchWizardCtrlCCancel(t *testing.T) {
	m := newTestWizard()
	// Navigate to middle step
	m = sendSpecialKey(m, tea.KeyEnter) // → Skills
	m = sendSpecialKey(m, tea.KeyEnter) // → Permissions
	var cmd tea.Cmd
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected command from ctrl+c")
	}
	msg := cmd()
	if _, ok := msg.(LaunchWizardCancelledMsg); !ok {
		t.Errorf("expected LaunchWizardCancelledMsg from ctrl+c, got %T", msg)
	}
}

// --- Hub mode tests ---

func TestLaunchWizardHubModeInitial(t *testing.T) {
	m := NewLaunchWizardModel(testProject(), LWModeHub, testDiscoveredItems(), nil)
	if m.Step() != launchStepMCPs {
		t.Errorf("hub initial step = %d, want MCPs", m.Step())
	}
	if m.WizardMode() != LWModeHub {
		t.Error("expected hub mode")
	}
	if m.Phase() != phaseSteps {
		t.Errorf("expected phaseSteps, got %d", m.Phase())
	}
}

func TestLaunchWizardHubTabCycles(t *testing.T) {
	m := NewLaunchWizardModel(testProject(), LWModeHub, testDiscoveredItems(), nil)
	m = sendSpecialKey(m, tea.KeyTab) // MCPs → Skills
	if m.Step() != launchStepSkills {
		t.Errorf("after tab step = %d, want Skills", m.Step())
	}
	// Tab all the way around
	for i := 0; i < int(launchStepCount)-1; i++ {
		m = sendSpecialKey(m, tea.KeyTab)
	}
	if m.Step() != launchStepMCPs {
		t.Errorf("tab should wrap around to MCPs, got %d", m.Step())
	}
}

func TestLaunchWizardHubShiftTab(t *testing.T) {
	m := NewLaunchWizardModel(testProject(), LWModeHub, testDiscoveredItems(), nil)
	// Shift+Tab from MCPs should wrap to Isolation
	m = sendSpecialKey(m, tea.KeyShiftTab)
	if m.Step() != launchStepIsolation {
		t.Errorf("shift+tab from MCPs should wrap to Isolation, got %d", m.Step())
	}
}

func TestLaunchWizardHubEdit(t *testing.T) {
	m := NewLaunchWizardModel(testProject(), LWModeHub, testDiscoveredItems(), nil)
	// Toggle an MCP
	m = sendKey(m, " ") // toggle mcp-a
	if !m.MCPSelections()["mcp-a"] {
		t.Error("expected mcp-a selected in hub mode")
	}
	// Switch to skills
	m = sendSpecialKey(m, tea.KeyTab)
	m = sendKey(m, " ") // toggle skill-1
	if !m.SkillSelections()["skill-1"] {
		t.Error("expected skill-1 selected in hub mode")
	}
	// MCP selection should be preserved
	if !m.MCPSelections()["mcp-a"] {
		t.Error("expected mcp-a selection preserved after switching category")
	}
}

func TestLaunchWizardHubEnterToChecklist(t *testing.T) {
	m := NewLaunchWizardModel(testProject(), LWModeHub, testDiscoveredItems(), nil)
	m = sendSpecialKey(m, tea.KeyEnter)
	if m.Phase() != phaseChecklist {
		t.Errorf("expected phaseChecklist after enter in hub, got %d", m.Phase())
	}
}

func TestLaunchWizardHubViewRenders(t *testing.T) {
	m := NewLaunchWizardModel(testProject(), LWModeHub, testDiscoveredItems(), nil)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	view := m.View()
	// Should show category menu items
	for _, label := range []string{"MCPs", "Skills", "Permissions", "Hooks", "Isolation"} {
		if !strings.Contains(view, label) {
			t.Errorf("hub view should contain %q", label)
		}
	}
	// Should show MCP items in content pane (MCPs is default step)
	if !strings.Contains(view, "mcp-a") {
		t.Error("hub view should show MCP items in content pane")
	}
}

func TestLaunchWizardHubSelectionCounts(t *testing.T) {
	m := NewLaunchWizardModel(testProject(), LWModeHub, testDiscoveredItems(), nil)
	m = sendKey(m, " ") // toggle mcp-a
	m = sendKey(m, "j")
	m = sendKey(m, " ") // toggle mcp-b
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	view := m.View()
	// The hub menu should show the count for MCPs
	if !strings.Contains(view, "(2)") {
		t.Error("hub should show (2) for MCPs with 2 selected")
	}
}

func TestLaunchWizardHubCancel(t *testing.T) {
	m := NewLaunchWizardModel(testProject(), LWModeHub, testDiscoveredItems(), nil)
	var cmd tea.Cmd
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected command from esc in hub mode")
	}
	msg := cmd()
	if _, ok := msg.(LaunchWizardCancelledMsg); !ok {
		t.Errorf("expected LaunchWizardCancelledMsg, got %T", msg)
	}
}

// --- Checklist tests ---

func TestLaunchWizardChecklistViewRenders(t *testing.T) {
	m := newTestWizard()
	m.mcpSelections["mcp-a"] = true
	m.skillSelections["skill-1"] = true
	m.phase = phaseChecklist
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	view := m.View()
	if !strings.Contains(view, "mcp-a") {
		t.Error("checklist should show selected MCP 'mcp-a'")
	}
	if !strings.Contains(view, "skill-1") {
		t.Error("checklist should show selected skill 'skill-1'")
	}
	if !strings.Contains(view, "launch") {
		t.Error("checklist should show 'launch' hint")
	}
}

func TestLaunchWizardChecklistAccuracy(t *testing.T) {
	m := newTestWizard()
	m.mcpSelections["mcp-a"] = true
	m.mcpSelections["mcp-c"] = true
	m.skillSelections["skill-2"] = true
	m.hookSelections["hook-pre-tool"] = true
	m.isolation = profile.IsolationWorktree
	m.phase = phaseChecklist
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	view := m.View()
	for _, name := range []string{"mcp-a", "mcp-c", "skill-2", "hook-pre-tool", "worktree"} {
		if !strings.Contains(view, name) {
			t.Errorf("checklist should contain %q", name)
		}
	}
}

func TestLaunchWizardChecklistEdit(t *testing.T) {
	m := newTestWizard()
	m = advanceToChecklist(m)
	m = sendKey(m, "e")
	if m.Phase() != phaseSteps {
		t.Errorf("expected phaseSteps after edit, got %d", m.Phase())
	}
	if m.WizardMode() != LWModeHub {
		t.Error("expected hub mode after edit from checklist")
	}
}

func TestLaunchWizardChecklistCancel(t *testing.T) {
	m := newTestWizard()
	m = advanceToChecklist(m)
	var cmd tea.Cmd
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected command from esc on checklist")
	}
	msg := cmd()
	if _, ok := msg.(LaunchWizardCancelledMsg); !ok {
		t.Errorf("expected LaunchWizardCancelledMsg, got %T", msg)
	}
}

func TestLaunchWizardChecklistLaunchCtrlL(t *testing.T) {
	m := newTestWizard()
	m.mcpSelections["mcp-b"] = true
	m = advanceToChecklist(m)
	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	if cmd == nil {
		t.Fatal("expected command from ctrl+l on checklist")
	}
	msg := cmd()
	complete, ok := msg.(LaunchWizardCompleteMsg)
	if !ok {
		t.Fatalf("expected LaunchWizardCompleteMsg, got %T", msg)
	}
	if !containsString(complete.Profile.MCPs, "mcp-b") {
		t.Error("expected profile to contain mcp-b")
	}
}

func TestLaunchWizardChecklistEditPreservesSelections(t *testing.T) {
	m := newTestWizard()
	m = sendKey(m, " ") // toggle mcp-a
	m = advanceToChecklist(m)
	m = sendKey(m, "e") // back to hub
	if !m.MCPSelections()["mcp-a"] {
		t.Error("expected mcp-a selection preserved after checklist → edit")
	}
}

func TestLaunchWizardChecklistEmptySelections(t *testing.T) {
	m := newTestWizard()
	m.phase = phaseChecklist
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	view := m.View()
	// All sections should show (none)
	if strings.Count(view, "(none)") < 4 {
		t.Error("checklist with no selections should show (none) for empty categories")
	}
}

func TestLaunchWizardChecklistShowsEnvVars(t *testing.T) {
	m := newTestWizard()
	m.envKeys = []string{"API_KEY"}
	m.envValues = []string{"secret123"}
	m.phase = phaseChecklist
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	view := m.View()
	if !strings.Contains(view, "API_KEY=secret123") {
		t.Error("checklist should show env vars")
	}
}

func TestLaunchWizardChecklistShowsClaudeMD(t *testing.T) {
	m := newTestWizard()
	m.claudeMDOptions = append(m.claudeMDOptions, "my-template.md")
	m.claudeMDChoice = 1
	m.phase = phaseChecklist
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	view := m.View()
	if !strings.Contains(view, "my-template.md") {
		t.Error("checklist should show selected CLAUDE.md template")
	}
}

// --- Helper tests ---

func TestCountSelected(t *testing.T) {
	sel := map[string]bool{"a": true, "b": false, "c": true}
	if got := countSelected(sel); got != 2 {
		t.Errorf("countSelected = %d, want 2", got)
	}
	if got := countSelected(nil); got != 0 {
		t.Errorf("countSelected(nil) = %d, want 0", got)
	}
}

func TestSelectedNames(t *testing.T) {
	m := newTestWizard()
	m.mcpSelections["mcp-c"] = true
	m.mcpSelections["mcp-a"] = true
	names := m.selectedNames(m.mcpSelections)
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	// Should be sorted
	if names[0] != "mcp-a" || names[1] != "mcp-c" {
		t.Errorf("expected sorted [mcp-a, mcp-c], got %v", names)
	}
}

func TestSelectionSummary(t *testing.T) {
	m := newTestWizard()
	m.mcpSelections["mcp-a"] = true
	m.mcpSelections["mcp-b"] = true
	if got := m.selectionSummary(launchStepMCPs); got != "(2)" {
		t.Errorf("MCP summary = %q, want %q", got, "(2)")
	}
	if got := m.selectionSummary(launchStepClaudeMD); got != "(none)" {
		t.Errorf("CLAUDE.md summary = %q, want %q", got, "(none)")
	}
	if got := m.selectionSummary(launchStepIsolation); got != "none" {
		t.Errorf("isolation summary = %q, want %q", got, "none")
	}
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
