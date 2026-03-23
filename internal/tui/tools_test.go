package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hystak/hystak/internal/model"
	"github.com/hystak/hystak/internal/service"
)

func TestToolsTab_GridNavigation(t *testing.T) {
	app := setupTestApp(t)
	tools := app.tabs[TabTools].(*toolsTab)
	tools = applyToolsWindowSize(t, tools, 80, 24)

	// Initially Sync should be highlighted (cursor 0)
	view := tools.View()
	if !strings.Contains(view, "Sync") {
		t.Fatal("view should contain Sync")
	}

	// Move right
	rightMsg := tea.KeyMsg{Type: tea.KeyRight}
	updated, _ := tools.Update(rightMsg)
	tools = updated.(*toolsTab)

	// View should still render, with Diff now reachable
	view = tools.View()
	if !strings.Contains(view, "Diff") {
		t.Error("view should contain Diff")
	}

	// Move down
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ = tools.Update(downMsg)
	tools = updated.(*toolsTab)

	view = tools.View()
	if !strings.Contains(view, "Launch") {
		t.Error("view should contain Launch")
	}
}

func TestToolsTab_EnterOpensProjectPicker(t *testing.T) {
	app := setupTestApp(t)
	tools := app.tabs[TabTools].(*toolsTab)
	tools = applyToolsWindowSize(t, tools, 80, 24)

	// Press Enter on Sync (cursor 0)
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := tools.Update(enterMsg)
	tools = updated.(*toolsTab)

	// View should show project picker
	view := tools.View()
	if !strings.Contains(view, "Select project") {
		t.Error("view should show project picker with 'Select project'")
	}
	if cmd == nil {
		t.Fatal("expected a command to load project list")
	}
}

func TestToolsTab_PickerEscReturns(t *testing.T) {
	app := setupTestApp(t)
	tools := app.tabs[TabTools].(*toolsTab)
	tools = applyToolsWindowSize(t, tools, 80, 24)

	// Open picker first
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := tools.Update(enterMsg)
	tools = updated.(*toolsTab)

	// Esc returns to grid
	escMsg := tea.KeyMsg{Type: tea.KeyEscape}
	updated, _ = tools.Update(escMsg)
	tools = updated.(*toolsTab)

	// View should show grid again (Sync, Diff, etc.)
	view := tools.View()
	if !strings.Contains(view, "Sync") {
		t.Error("after Esc, view should show Sync in grid")
	}
	if strings.Contains(view, "Select project") {
		t.Error("after Esc, project picker should be dismissed")
	}
}

func TestToolsTab_DiffDoneShowsOverlay(t *testing.T) {
	app := setupTestApp(t)
	tools := app.tabs[TabTools].(*toolsTab)
	tools = applyToolsWindowSize(t, tools, 80, 24)

	msg := toolsDiffDoneMsg{
		results: []service.DiffResult{
			{ServerName: "github", Status: model.DriftSynced},
			{ServerName: "postgres", Status: model.DriftDrifted},
		},
	}
	updated, _ := tools.Update(msg)
	tools = updated.(*toolsTab)

	// View should show diff results
	view := tools.View()
	if !strings.Contains(view, "github") {
		t.Error("diff view should contain 'github'")
	}
	if !strings.Contains(view, "postgres") {
		t.Error("diff view should contain 'postgres'")
	}
}

func TestToolsTab_DiffEscCloses(t *testing.T) {
	app := setupTestApp(t)
	tools := app.tabs[TabTools].(*toolsTab)
	tools = applyToolsWindowSize(t, tools, 80, 24)

	// Put into diff mode
	msg := toolsDiffDoneMsg{
		results: []service.DiffResult{
			{ServerName: "github", Status: model.DriftSynced},
		},
	}
	updated, _ := tools.Update(msg)
	tools = updated.(*toolsTab)

	// Esc closes diff view
	escMsg := tea.KeyMsg{Type: tea.KeyEscape}
	updated, _ = tools.Update(escMsg)
	tools = updated.(*toolsTab)

	// View should return to grid
	view := tools.View()
	if !strings.Contains(view, "Sync") {
		t.Error("after Esc from diff, view should show Sync in grid")
	}
}

func TestToolsTab_SyncDoneSetsMessage(t *testing.T) {
	app := setupTestApp(t)
	tools := app.tabs[TabTools].(*toolsTab)
	tools = applyToolsWindowSize(t, tools, 80, 24)

	msg := toolsSyncDoneMsg{
		results: []service.SyncResult{{Name: "github", Action: service.SyncAdded}},
	}
	updated, _ := tools.Update(msg)
	tools = updated.(*toolsTab)

	// View should show the sync complete message
	view := tools.View()
	if !strings.Contains(view, "Sync complete") {
		t.Errorf("view should contain 'Sync complete', got: %s", view)
	}
}

func TestToolsTab_SyncDone_WithError_ShowsInView(t *testing.T) {
	app := setupTestApp(t)
	tools := app.tabs[TabTools].(*toolsTab)
	tools = applyToolsWindowSize(t, tools, 80, 24)

	msg := toolsSyncDoneMsg{
		err: fmt.Errorf("sync failed: project not found"),
	}
	updated, _ := tools.Update(msg)
	tools = updated.(*toolsTab)

	view := tools.View()
	if !strings.Contains(view, "sync failed") {
		t.Error("sync error should appear in View()")
	}
}

func TestToolsTab_View_ShowsGrid(t *testing.T) {
	app := setupTestApp(t)
	tools := app.tabs[TabTools].(*toolsTab)
	tools = applyToolsWindowSize(t, tools, 80, 24)

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
		{ServerName: "aaa", Status: model.DriftSynced},
		{ServerName: "bbb", Status: model.DriftDrifted},
	}
	dv := newDiffViewModel(results, DefaultKeyMap())

	// Verify initial view contains first item
	view := dv.view(80, 24)
	if !strings.Contains(view, "aaa") {
		t.Error("initial diff view should contain aaa")
	}

	// Move down
	dv, _ = dv.update(tea.KeyMsg{Type: tea.KeyDown})
	view = dv.view(80, 24)
	if !strings.Contains(view, "bbb") {
		t.Error("after down, diff view should contain bbb")
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

// applyToolsWindowSize sends a WindowSizeMsg and returns the updated toolsTab.
func applyToolsWindowSize(t *testing.T, tools *toolsTab, w, h int) *toolsTab {
	t.Helper()
	updated, _ := tools.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return updated.(*toolsTab)
}
