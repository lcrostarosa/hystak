package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hystak/hystak/internal/model"
	"github.com/hystak/hystak/internal/service"
)

func TestToolsTab_GridNavigation(t *testing.T) {
	app := setupTestApp(t)
	tools := app.tabs[TabTools].(*toolsTab)
	tools.width = 80
	tools.height = 24

	if tools.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", tools.cursor)
	}

	// Move right
	rightMsg := tea.KeyMsg{Type: tea.KeyRight}
	updated, _ := tools.Update(rightMsg)
	tools = updated.(*toolsTab)
	if tools.cursor != 1 {
		t.Errorf("cursor after right = %d, want 1", tools.cursor)
	}

	// Move down
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ = tools.Update(downMsg)
	tools = updated.(*toolsTab)
	if tools.cursor != 3 {
		t.Errorf("cursor after down = %d, want 3", tools.cursor)
	}
}

func TestToolsTab_EnterOpensProjectPicker(t *testing.T) {
	app := setupTestApp(t)
	tools := app.tabs[TabTools].(*toolsTab)
	tools.width = 80
	tools.height = 24

	// Press Enter on Sync (cursor 0)
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := tools.Update(enterMsg)
	tools = updated.(*toolsTab)

	if tools.mode != toolsModePicker {
		t.Errorf("mode after Enter = %d, want picker", tools.mode)
	}
	if cmd == nil {
		t.Fatal("expected a command to load project list")
	}
}

func TestToolsTab_PickerEscReturns(t *testing.T) {
	app := setupTestApp(t)
	tools := app.tabs[TabTools].(*toolsTab)
	tools.mode = toolsModePicker

	escMsg := tea.KeyMsg{Type: tea.KeyEscape}
	updated, _ := tools.Update(escMsg)
	tools = updated.(*toolsTab)

	if tools.mode != toolsModeGrid {
		t.Errorf("mode after Esc = %d, want grid", tools.mode)
	}
}

func TestToolsTab_DiffDoneShowsOverlay(t *testing.T) {
	app := setupTestApp(t)
	tools := app.tabs[TabTools].(*toolsTab)

	msg := toolsDiffDoneMsg{
		results: []service.DiffResult{
			{ServerName: "github", Status: model.DriftSynced},
			{ServerName: "postgres", Status: model.DriftDrifted},
		},
	}
	updated, _ := tools.Update(msg)
	tools = updated.(*toolsTab)

	if tools.mode != toolsModeDiff {
		t.Errorf("mode after diff done = %d, want diff", tools.mode)
	}
	if len(tools.diffView.results) != 2 {
		t.Errorf("diff results = %d, want 2", len(tools.diffView.results))
	}
}

func TestToolsTab_DiffEscCloses(t *testing.T) {
	app := setupTestApp(t)
	tools := app.tabs[TabTools].(*toolsTab)
	tools.mode = toolsModeDiff
	tools.diffView = newDiffViewModel(nil, tools.keys)

	escMsg := tea.KeyMsg{Type: tea.KeyEscape}
	updated, _ := tools.Update(escMsg)
	tools = updated.(*toolsTab)

	if tools.mode != toolsModeGrid {
		t.Errorf("mode after Esc = %d, want grid", tools.mode)
	}
}

func TestToolsTab_SyncDoneSetsMessage(t *testing.T) {
	app := setupTestApp(t)
	tools := app.tabs[TabTools].(*toolsTab)

	msg := toolsSyncDoneMsg{
		results: []service.SyncResult{{Name: "github", Action: service.SyncAdded}},
	}
	updated, _ := tools.Update(msg)
	tools = updated.(*toolsTab)

	if tools.mode != toolsModeGrid {
		t.Errorf("mode = %d, want grid", tools.mode)
	}
	if !strings.Contains(tools.err, "Sync complete") {
		t.Errorf("err = %q, want Sync complete message", tools.err)
	}
}

func TestToolsTab_View_ShowsGrid(t *testing.T) {
	app := setupTestApp(t)
	tools := app.tabs[TabTools].(*toolsTab)
	tools.width = 80
	tools.height = 24

	view := tools.View()
	if !strings.Contains(view, "Sync") {
		t.Error("view should contain Sync")
	}
	if !strings.Contains(view, "Diff") {
		t.Error("view should contain Diff")
	}
	if !strings.Contains(view, "Launch") {
		t.Error("view should contain Launch")
	}
}

// --- Diff view tests ---

func TestDiffView_Navigation(t *testing.T) {
	results := []service.DiffResult{
		{ServerName: "a", Status: model.DriftSynced},
		{ServerName: "b", Status: model.DriftDrifted},
	}
	dv := newDiffViewModel(results, DefaultKeyMap())

	if dv.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", dv.cursor)
	}

	dv = dv.update(tea.KeyMsg{Type: tea.KeyDown})
	if dv.cursor != 1 {
		t.Errorf("cursor after down = %d, want 1", dv.cursor)
	}
}

func TestDiffView_StatusIcons(t *testing.T) {
	tests := []struct {
		status model.DriftStatus
	}{
		{model.DriftSynced},
		{model.DriftDrifted},
		{model.DriftMissing},
		{model.DriftUnmanaged},
	}
	for _, tt := range tests {
		icon := statusIcon(tt.status)
		if icon == " " {
			t.Errorf("statusIcon(%s) returned blank", tt.status)
		}
	}
}

// --- Wizard tests ---

func TestWizard_StepNavigation(t *testing.T) {
	app := setupTestApp(t)
	svc := app.svc
	w := newWizardModel(DefaultKeyMap(), svc, "proj", "dev")

	if w.step != wizardStepMCPs {
		t.Fatalf("initial step = %d, want MCPs", w.step)
	}

	// Enter advances to step 2
	w, _ = w.update(tea.KeyMsg{Type: tea.KeyEnter})
	if w.step != wizardStepOptions {
		t.Errorf("step after Enter = %d, want Options", w.step)
	}

	// Enter advances to step 3
	w, _ = w.update(tea.KeyMsg{Type: tea.KeyEnter})
	if w.step != wizardStepReview {
		t.Errorf("step after Enter = %d, want Review", w.step)
	}

	// Left goes back
	w, _ = w.update(tea.KeyMsg{Type: tea.KeyLeft})
	if w.step != wizardStepOptions {
		t.Errorf("step after Left = %d, want Options", w.step)
	}
}

func TestWizard_Toggle(t *testing.T) {
	app := setupTestApp(t)
	svc := app.svc
	w := newWizardModel(DefaultKeyMap(), svc, "proj", "dev")

	if len(w.allMCPs) == 0 {
		t.Skip("no MCPs in test service")
	}

	first := w.allMCPs[0]
	initialState := w.selectedMCP[first]

	// First toggle flips from initial state
	w, _ = w.update(tea.KeyMsg{Type: tea.KeySpace})
	if w.selectedMCP[first] == initialState {
		t.Errorf("MCP %q should have toggled from %v", first, initialState)
	}

	// Second toggle flips back
	w, _ = w.update(tea.KeyMsg{Type: tea.KeySpace})
	if w.selectedMCP[first] != initialState {
		t.Errorf("MCP %q should have toggled back to %v", first, initialState)
	}
}

func TestCountTrue(t *testing.T) {
	m := map[string]bool{"a": true, "b": false, "c": true}
	if got := countTrue(m); got != 2 {
		t.Errorf("countTrue = %d, want 2", got)
	}
}
