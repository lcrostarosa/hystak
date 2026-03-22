package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/project"
	"github.com/lcrostarosa/hystak/internal/registry"
	"github.com/lcrostarosa/hystak/internal/service"
)

func testProjectService() *service.Service {
	reg := &registry.Registry{
		Servers: map[string]model.ServerDef{
			"github": {
				Name:      "github",
				Transport: model.TransportStdio,
				Command:   "npx",
				Args:      []string{"-y", "@modelcontextprotocol/server-github"},
				Env:       map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
			},
			"qdrant": {
				Name:        "qdrant",
				Description: "Qdrant vector database",
				Transport:   model.TransportHTTP,
				URL:         "http://localhost:6333/mcp",
			},
			"slack": {
				Name:      "slack",
				Transport: model.TransportStdio,
				Command:   "npx",
				Args:      []string{"-y", "@modelcontextprotocol/server-slack"},
			},
		},
		Tags: map[string][]string{
			"core": {"github", "qdrant"},
		},
		Skills:      map[string]model.SkillDef{},
		Hooks:       map[string]model.HookDef{},
		Permissions: map[string]model.PermissionRule{},
		Templates:   map[string]model.TemplateDef{},
	}

	store := &project.Store{
		Projects: map[string]model.Project{
			"myproject": {
				Name:    "myproject",
				Path:    "/tmp/myproject",
				Tags:    []string{"core"},
				Clients: []model.ClientType{model.ClientClaudeCode},
				MCPs: []model.MCPAssignment{
					{Name: "slack"},
				},
			},
			"other": {
				Name: "other",
				Path: "/tmp/other",
			},
		},
	}

	return service.NewForTest(reg, store, nil, nil, "", nil)
}

func TestNewProfilesModelNilService(t *testing.T) {
	m := NewProfilesModel(nil, newDefaultKeyMap())
	if m.list.FilterState() != 0 {
		t.Error("expected initial filter state to be unfiltered")
	}
}

func TestNewProfilesModelPopulatesList(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())

	items := m.list.Items()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// Items should be sorted by name.
	first := items[0].(profileItem)
	second := items[1].(profileItem)
	if first.project.Name != "myproject" {
		t.Errorf("expected first project 'myproject', got %q", first.project.Name)
	}
	if second.project.Name != "other" {
		t.Errorf("expected second project 'other', got %q", second.project.Name)
	}
}

func TestProfileItemTitle(t *testing.T) {
	item := profileItem{
		project:     model.Project{Name: "myproject"},
		serverCount: 3,
	}
	if title := item.Title(); title != "myproject [3]" {
		t.Errorf("expected 'myproject [3]', got %q", title)
	}

	item.serverCount = 0
	if title := item.Title(); title != "myproject" {
		t.Errorf("expected 'myproject', got %q", title)
	}
}

func TestProfileItemDescription(t *testing.T) {
	// With path and no resources: should show just path.
	item := profileItem{
		project: model.Project{Name: "myproject", Path: "/tmp/myproject"},
	}
	if desc := item.Description(); desc != "/tmp/myproject" {
		t.Errorf("expected path description, got %q", desc)
	}

	// With path and MCPs: should show path with summary.
	item.serverCount = 3
	desc := item.Description()
	if !strings.Contains(desc, "/tmp/myproject") {
		t.Errorf("expected path in description, got %q", desc)
	}
	if !strings.Contains(desc, "MCPs") {
		t.Errorf("expected 'MCPs' in description, got %q", desc)
	}

	// No path, no resources: "no path".
	item2 := profileItem{project: model.Project{Name: "other"}}
	if desc := item2.Description(); desc != "no path" {
		t.Errorf("expected 'no path', got %q", desc)
	}
}

func TestProfileItemFilterValue(t *testing.T) {
	item := profileItem{project: model.Project{Name: "myproject"}}
	if fv := item.FilterValue(); fv != "myproject" {
		t.Errorf("expected 'myproject', got %q", fv)
	}
}

func TestProfileServerCount(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())

	items := m.list.Items()
	myproj := items[0].(profileItem)
	// myproject has tags=[core] (github, qdrant) + MCPs=[slack] = 3 servers
	if myproj.serverCount != 3 {
		t.Errorf("expected serverCount=3, got %d", myproj.serverCount)
	}

	other := items[1].(profileItem)
	if other.serverCount != 0 {
		t.Errorf("expected serverCount=0 for 'other', got %d", other.serverCount)
	}
}

func TestSelectedProfile(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	proj, ok := m.selectedProfile()
	if !ok {
		t.Fatal("expected a selected profile")
	}
	if proj.Name != "myproject" {
		t.Errorf("expected selected profile 'myproject', got %q", proj.Name)
	}
}

func TestSelectedProfileEmpty(t *testing.T) {
	m := NewProfilesModel(nil, newDefaultKeyMap())
	_, ok := m.selectedProfile()
	if ok {
		t.Error("expected no selected profile with nil service")
	}
}

func TestProfilesSetSize(t *testing.T) {
	m := NewProfilesModel(nil, newDefaultKeyMap())
	m.SetSize(100, 30)
	if m.width != 100 {
		t.Errorf("expected width 100, got %d", m.width)
	}
	if m.height != 30 {
		t.Errorf("expected height 30, got %d", m.height)
	}
}

func TestProfilesIsConsuming(t *testing.T) {
	m := NewProfilesModel(nil, newDefaultKeyMap())
	if m.IsConsuming() {
		t.Error("expected IsConsuming to be false initially")
	}

	m.confirming = true
	if !m.IsConsuming() {
		t.Error("expected IsConsuming to be true when confirming")
	}
	m.confirming = false

	m.focus = focusRight
	if !m.IsConsuming() {
		t.Error("expected IsConsuming to be true when right pane focused")
	}
}

func TestProfilesStatusHelp(t *testing.T) {
	m := NewProfilesModel(nil, newDefaultKeyMap())
	help := m.StatusHelp()
	if !strings.Contains(help, "d: delete") {
		t.Errorf("expected 'd: delete' in help, got %q", help)
	}

	m.confirming = true
	help = m.StatusHelp()
	if !strings.Contains(help, "y: confirm") {
		t.Errorf("expected 'y: confirm' in confirming help, got %q", help)
	}

	m.confirming = false
	m.focus = focusRight
	help = m.StatusHelp()
	if !strings.Contains(help, "space: toggle") {
		t.Errorf("expected 'space: toggle' in right pane help, got %q", help)
	}
}

func TestProfileDeleteConfirmation(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	// Press 'd' to start delete confirmation.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'d'}}))
	if !m.confirming {
		t.Fatal("expected confirming to be true after 'd' press")
	}

	// Press 'n' to cancel.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'n'}}))
	if m.confirming {
		t.Error("expected confirming to be false after 'n' press")
	}
}

func TestProfileDeleteExecute(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'d'}}))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'y'}}))

	if m.confirming {
		t.Error("expected confirming to be false after confirmation")
	}

	if _, ok := svc.GetProject("myproject"); ok {
		t.Error("expected myproject to be deleted")
	}

	if len(m.list.Items()) != 1 {
		t.Errorf("expected 1 item after delete, got %d", len(m.list.Items()))
	}
}

func TestFocusSwitchToRight(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	if m.focus != focusLeft {
		t.Fatal("expected initial focus on left pane")
	}

	// Press 'enter' to focus right pane.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'c'}}))
	if m.focus != focusRight {
		t.Error("expected focus on right pane after enter")
	}
	// Active section should be MCPs and cursor should be 0.
	if m.activeSection != sectionMCPs {
		t.Errorf("expected activeSection=sectionMCPs, got %d", m.activeSection)
	}
	if m.sectionCursors[sectionMCPs] != 0 {
		t.Errorf("expected sectionCursors[sectionMCPs]=0, got %d", m.sectionCursors[sectionMCPs])
	}
}

func TestFocusSwitchBackToLeft(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	// Enter right pane, then escape back.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'c'}}))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEsc}))
	if m.focus != focusLeft {
		t.Error("expected focus on left pane after esc")
	}
}

func TestRightPaneNavigation(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	// Enter right pane.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'c'}}))

	// Should have 3 servers: github, qdrant, slack (sorted).
	if len(m.allMCPs) != 3 {
		t.Fatalf("expected 3 servers, got %d", len(m.allMCPs))
	}

	// Navigate down.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyDown}))
	if m.sectionCursors[sectionMCPs] != 1 {
		t.Errorf("expected sectionCursors[sectionMCPs]=1, got %d", m.sectionCursors[sectionMCPs])
	}

	// Navigate down again.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyDown}))
	if m.sectionCursors[sectionMCPs] != 2 {
		t.Errorf("expected sectionCursors[sectionMCPs]=2, got %d", m.sectionCursors[sectionMCPs])
	}

	// Try to navigate past end.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyDown}))
	if m.sectionCursors[sectionMCPs] != 2 {
		t.Errorf("expected sectionCursors[sectionMCPs] to stay at 2, got %d", m.sectionCursors[sectionMCPs])
	}

	// Navigate up.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyUp}))
	if m.sectionCursors[sectionMCPs] != 1 {
		t.Errorf("expected sectionCursors[sectionMCPs]=1, got %d", m.sectionCursors[sectionMCPs])
	}
}

func TestSectionNavigation(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	// Enter right pane — starts at MCPs section.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'c'}}))
	if m.activeSection != sectionMCPs {
		t.Fatalf("expected activeSection=sectionMCPs initially, got %d", m.activeSection)
	}

	// Tab to next section.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyTab}))
	if m.activeSection != sectionSkills {
		t.Errorf("expected sectionSkills after tab, got %d", m.activeSection)
	}

	// Tab again.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyTab}))
	if m.activeSection != sectionHooks {
		t.Errorf("expected sectionHooks after second tab, got %d", m.activeSection)
	}

	// Shift+tab back.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyShiftTab}))
	if m.activeSection != sectionSkills {
		t.Errorf("expected sectionSkills after shift+tab, got %d", m.activeSection)
	}
}

func TestServerToggleAssign(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	// Enter right pane. Servers are: github, qdrant, slack.
	// github and qdrant are from tag "core", slack is direct MCP.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'c'}}))

	// Cursor on github (index 0). github is from tag — should error.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{' '}}))
	if m.err == nil {
		t.Fatal("expected error toggling tag-sourced server")
	}
	if !strings.Contains(m.err.Error(), "via tag") {
		t.Errorf("expected tag error, got: %s", m.err.Error())
	}
}

func TestServerToggleUnassign(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	// Enter right pane. Navigate to "slack" (index 2).
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'c'}}))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyDown})) // qdrant
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyDown})) // slack

	// slack is directly assigned. Toggle (unassign).
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{' '}}))
	if m.err != nil {
		t.Fatalf("unexpected error: %s", m.err.Error())
	}

	// Verify slack is no longer assigned.
	proj, _ := svc.GetProject("myproject")
	for _, mcp := range proj.MCPs {
		if mcp.Name == "slack" {
			t.Error("expected slack to be unassigned")
		}
	}
}

func TestServerToggleReassign(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	// Enter right pane. Navigate to "slack" (index 2).
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'c'}}))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyDown}))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyDown}))

	// Unassign slack.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{' '}}))
	// Reassign slack.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{' '}}))
	if m.err != nil {
		t.Fatalf("unexpected error on reassign: %s", m.err.Error())
	}

	proj, _ := svc.GetProject("myproject")
	found := false
	for _, mcp := range proj.MCPs {
		if mcp.Name == "slack" {
			found = true
		}
	}
	if !found {
		t.Error("expected slack to be reassigned")
	}
}

func TestIsServerAssigned(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())

	proj, _ := svc.GetProject("myproject")

	// github is assigned via tag "core".
	if !m.isServerAssigned(proj, "github") {
		t.Error("expected github to be assigned (via tag)")
	}
	// slack is assigned directly.
	if !m.isServerAssigned(proj, "slack") {
		t.Error("expected slack to be assigned (direct)")
	}
}

func TestIsServerFromTag(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())

	proj, _ := svc.GetProject("myproject")

	if !m.isServerFromTag(proj, "github") {
		t.Error("expected github to be from tag")
	}
	if m.isServerFromTag(proj, "slack") {
		t.Error("expected slack NOT to be from tag")
	}
}

func TestProfilesViewRendersDetail(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	view := m.View()
	if !strings.Contains(view, "myproject") {
		t.Errorf("expected 'myproject' in view, got:\n%s", view)
	}
	if !strings.Contains(view, "Path:") {
		t.Errorf("expected 'Path:' label in view, got:\n%s", view)
	}
	if !strings.Contains(view, "MCPs") {
		t.Errorf("expected 'MCPs' section header in view, got:\n%s", view)
	}
}

func TestProfilesViewEmptyWithZeroSize(t *testing.T) {
	m := NewProfilesModel(nil, newDefaultKeyMap())
	if view := m.View(); view != "" {
		t.Errorf("expected empty string for zero-size view, got %q", view)
	}
}

func TestProfilesRenderDetailNoSelection(t *testing.T) {
	m := NewProfilesModel(nil, newDefaultKeyMap())
	detail := m.renderDetail(40, 20)
	if !strings.Contains(detail, "No profile selected") {
		t.Errorf("expected 'No profile selected', got:\n%s", detail)
	}
}

func TestProfilesRenderDetailShowsFields(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	detail := m.renderDetail(40, 20)
	checks := []string{"myproject", "Path:", "/tmp/myproject", "Tags:", "core", "Clients:", "claude-code", "MCPs", "[x]", "[t]"}
	for _, check := range checks {
		if !strings.Contains(detail, check) {
			t.Errorf("expected %q in detail view, got:\n%s", check, detail)
		}
	}
}

func TestRenderDetailShowsCheckboxes(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	detail := m.renderDetail(40, 20)
	// github and qdrant are from tag "core" — shown as [t].
	// slack is directly assigned — shown as [x].
	if !strings.Contains(detail, "[t] github") {
		t.Errorf("expected '[t] github' for tag-sourced server, got:\n%s", detail)
	}
	if !strings.Contains(detail, "[t] qdrant") {
		t.Errorf("expected '[t] qdrant' for tag-sourced server, got:\n%s", detail)
	}
	if !strings.Contains(detail, "[x] slack") {
		t.Errorf("expected '[x] slack' for directly assigned server, got:\n%s", detail)
	}
}

func TestRenderDetailCursorInRightPane(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	// Enter right pane.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'c'}}))

	detail := m.renderDetail(40, 20)
	// Cursor should be on first server (github).
	if !strings.Contains(detail, "\u25b8") {
		t.Errorf("expected cursor indicator in right pane, got:\n%s", detail)
	}
}

func TestBuildAllServerNames(t *testing.T) {
	svc := testProjectService()
	names := buildAllServerNames(svc)
	expected := []string{"github", "qdrant", "slack"}
	if len(names) != len(expected) {
		t.Fatalf("expected %d servers, got %d", len(expected), len(names))
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("expected server %d to be %q, got %q", i, expected[i], name)
		}
	}
}

func TestCountAssignedServers(t *testing.T) {
	svc := testProjectService()
	proj, _ := svc.GetProject("myproject")
	count := svc.CountAssignedServers(proj)
	// tags=[core] gives github+qdrant, MCPs=[slack] = 3
	if count != 3 {
		t.Errorf("expected 3 assigned servers, got %d", count)
	}
}

func TestProfilesRefreshList(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	// Add a new project.
	_ = svc.AddProject(model.Project{
		Name: "newproject",
		Path: "/tmp/newproject",
	})

	m.refreshList()
	if len(m.list.Items()) != 3 {
		t.Errorf("expected 3 items after refresh, got %d", len(m.list.Items()))
	}
}

func TestBuildAllSkillNames(t *testing.T) {
	svc := testProjectService()
	// No skills in registry — should return empty slice.
	names := buildAllSkillNames(svc)
	if names == nil {
		// nil is acceptable for empty result from registry with no skills
		return
	}
	if len(names) != 0 {
		t.Errorf("expected 0 skill names, got %d", len(names))
	}
}

func TestAutoSyncResultMsg(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	// Send a successful AutoSyncResultMsg.
	msg := AutoSyncResultMsg{
		ProjectName: "myproject",
		Results:     []service.SyncResult{},
		Err:         nil,
	}
	m, _ = m.Update(msg)
	// syncMsg should be set.
	if m.syncMsg == "" {
		t.Error("expected syncMsg to be set after successful auto-sync")
	}
}

func TestAutoSyncResultMsgError(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	// Send a failed AutoSyncResultMsg.
	msg := AutoSyncResultMsg{
		ProjectName: "myproject",
		Err:         fmt.Errorf("sync failed"),
	}
	m, _ = m.Update(msg)
	if m.err == nil {
		t.Error("expected err to be set after failed auto-sync")
	}
}

func TestLaunchKey(t *testing.T) {
	svc := testProjectService()
	m := NewProfilesModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	// Press 'enter' to launch.
	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	if cmd == nil {
		t.Fatal("expected a command from enter press")
	}
	// Execute the cmd and check for RequestLaunchMsg.
	result := cmd()
	launchMsg, ok := result.(RequestLaunchMsg)
	if !ok {
		t.Fatalf("expected RequestLaunchMsg, got %T", result)
	}
	if launchMsg.ProfileName != "myproject" {
		t.Errorf("expected ProfileName='myproject', got %q", launchMsg.ProfileName)
	}
}
