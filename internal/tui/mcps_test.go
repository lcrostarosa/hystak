package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/project"
	"github.com/lcrostarosa/hystak/internal/registry"
	"github.com/lcrostarosa/hystak/internal/service"
)

func testService() *service.Service {
	reg := &registry.Registry{
		Servers:     registry.NewStore[model.ServerDef, *model.ServerDef]("server"),
		Skills:      registry.NewStore[model.SkillDef, *model.SkillDef]("skill"),
		Hooks:       registry.NewStore[model.HookDef, *model.HookDef]("hook"),
		Permissions: registry.NewStore[model.PermissionRule, *model.PermissionRule]("permission"),
		Templates:   registry.NewStore[model.TemplateDef, *model.TemplateDef]("template"),
		Prompts:     registry.NewStore[model.PromptDef, *model.PromptDef]("prompt"),
		Tags:        make(map[string][]string),
	}
	_ = reg.Servers.Add(model.ServerDef{
		Name:      "github",
		Transport: model.TransportStdio,
		Command:   "npx",
		Args:      []string{"-y", "@modelcontextprotocol/server-github"},
		Env:       map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
	})
	_ = reg.Servers.Add(model.ServerDef{
		Name:        "qdrant",
		Description: "Qdrant vector database",
		Transport:   model.TransportHTTP,
		URL:         "http://localhost:6333/mcp",
		Headers:     map[string]string{"Authorization": "Bearer ${QDRANT_API_KEY}"},
	})

	store := &project.Store{
		Projects: map[string]model.Project{
			"myproject": {
				Name: "myproject",
				Path: "/tmp/myproject",
				MCPs: []model.MCPAssignment{
					{Name: "github"},
				},
			},
		},
	}

	return service.NewForTest(reg, store, nil, nil, "", nil)
}

func TestNewMCPsModelNilService(t *testing.T) {
	m := NewMCPsModel(nil, newDefaultKeyMap())
	if m.list.FilterState() != 0 {
		t.Error("expected initial filter state to be unfiltered")
	}
}

func TestNewMCPsModelPopulatesList(t *testing.T) {
	svc := testService()
	m := NewMCPsModel(svc, newDefaultKeyMap())

	items := m.list.Items()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// Items should be sorted by name.
	first := items[0].(mcpItem)
	second := items[1].(mcpItem)
	if first.server.Name != "github" {
		t.Errorf("expected first server 'github', got %q", first.server.Name)
	}
	if second.server.Name != "qdrant" {
		t.Errorf("expected second server 'qdrant', got %q", second.server.Name)
	}
}

func TestMCPItemProfileCount(t *testing.T) {
	svc := testService()
	m := NewMCPsModel(svc, newDefaultKeyMap())

	items := m.list.Items()
	github := items[0].(mcpItem)
	qdrant := items[1].(mcpItem)

	// github is assigned to myproject.
	if github.profileCount != 1 {
		t.Errorf("expected github profileCount=1, got %d", github.profileCount)
	}
	// qdrant is not assigned.
	if qdrant.profileCount != 0 {
		t.Errorf("expected qdrant profileCount=0, got %d", qdrant.profileCount)
	}
}

func TestMCPItemTitle(t *testing.T) {
	item := mcpItem{
		server:       model.ServerDef{Name: "github"},
		profileCount: 3,
	}
	if title := item.Title(); title != "github ⌂3" {
		t.Errorf("expected 'github ⌂3', got %q", title)
	}

	item.profileCount = 0
	if title := item.Title(); title != "github" {
		t.Errorf("expected 'github', got %q", title)
	}
}

func TestMCPItemDescription(t *testing.T) {
	item := mcpItem{
		server: model.ServerDef{Name: "qdrant", Description: "Qdrant vector database", Transport: model.TransportHTTP},
	}
	if desc := item.Description(); desc != "Qdrant vector database" {
		t.Errorf("expected description, got %q", desc)
	}

	item.server.Description = ""
	if desc := item.Description(); desc != "http" {
		t.Errorf("expected transport as fallback description, got %q", desc)
	}
}

func TestMCPItemFilterValue(t *testing.T) {
	item := mcpItem{server: model.ServerDef{Name: "github"}}
	if fv := item.FilterValue(); fv != "github" {
		t.Errorf("expected 'github', got %q", fv)
	}
}

func TestSelectedMCP(t *testing.T) {
	svc := testService()
	m := NewMCPsModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	srv, ok := m.selectedMCP()
	if !ok {
		t.Fatal("expected a selected MCP")
	}
	// First item should be selected by default.
	if srv.Name != "github" {
		t.Errorf("expected selected MCP 'github', got %q", srv.Name)
	}
}

func TestSelectedMCPEmpty(t *testing.T) {
	m := NewMCPsModel(nil, newDefaultKeyMap())
	_, ok := m.selectedMCP()
	if ok {
		t.Error("expected no selected MCP with nil service")
	}
}

func TestSetSize(t *testing.T) {
	m := NewMCPsModel(nil, newDefaultKeyMap())
	m.SetSize(100, 30)
	if m.width != 100 {
		t.Errorf("expected width 100, got %d", m.width)
	}
	if m.height != 30 {
		t.Errorf("expected height 30, got %d", m.height)
	}
}

func TestIsConsuming(t *testing.T) {
	m := NewMCPsModel(nil, newDefaultKeyMap())
	if m.IsConsuming() {
		t.Error("expected IsConsuming to be false initially")
	}

	// Enter confirming state.
	m.confirming = true
	if !m.IsConsuming() {
		t.Error("expected IsConsuming to be true when confirming")
	}
}

func TestStatusHelp(t *testing.T) {
	m := NewMCPsModel(nil, newDefaultKeyMap())
	help := m.StatusHelp()
	if !strings.Contains(help, "d: delete") {
		t.Errorf("expected 'd: delete' in help, got %q", help)
	}

	m.confirming = true
	help = m.StatusHelp()
	if !strings.Contains(help, "y: confirm") {
		t.Errorf("expected 'y: confirm' in confirming help, got %q", help)
	}
}

func TestDeleteConfirmation(t *testing.T) {
	svc := testService()
	m := NewMCPsModel(svc, newDefaultKeyMap())
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

func TestDeleteConfirmExecute(t *testing.T) {
	svc := testService()
	m := NewMCPsModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	// Start delete.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'d'}}))
	if !m.confirming {
		t.Fatal("expected confirming state")
	}

	// Confirm with 'y'.
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'y'}}))
	if m.confirming {
		t.Error("expected confirming to be false after confirmation")
	}

	// github should be removed from the registry.
	if _, ok := svc.GetServer("github"); ok {
		t.Error("expected github to be deleted from registry")
	}

	// List should be refreshed — only 1 item.
	if len(m.list.Items()) != 1 {
		t.Errorf("expected 1 item after delete, got %d", len(m.list.Items()))
	}
}

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
	_ = reg.Servers.Add(model.ServerDef{
		Name:      "github",
		Transport: model.TransportStdio,
		Command:   "npx",
		Args:      []string{"-y", "@modelcontextprotocol/server-github"},
		Env:       map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
	})
	_ = reg.Servers.Add(model.ServerDef{
		Name:        "qdrant",
		Description: "Qdrant vector database",
		Transport:   model.TransportHTTP,
		URL:         "http://localhost:6333/mcp",
		Headers:     map[string]string{"Authorization": "Bearer ${QDRANT_API_KEY}"},
	})
	store := &project.Store{
		Projects: map[string]model.Project{
			"myproject": {
				Name: "myproject",
				Path: "/tmp/myproject",
				MCPs: []model.MCPAssignment{{Name: "github"}},
			},
		},
	}
	svc := service.NewForTest(reg, store, nil, nil, "", nil)

	m := NewMCPsModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	// Start and confirm delete — should fail because github is in tag "core".
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'d'}}))
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'y'}}))

	if m.err == nil {
		t.Error("expected error when deleting MCP referenced by tag")
	}
	if !strings.Contains(m.err.Error(), "referenced by tag") {
		t.Errorf("expected tag reference error, got: %s", m.err.Error())
	}

	// Server should still exist.
	if _, ok := svc.GetServer("github"); !ok {
		t.Error("expected github to still exist in registry")
	}
}

func TestViewRendersDetail(t *testing.T) {
	svc := testService()
	m := NewMCPsModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	view := m.View()
	// Should show selected MCP's details.
	if !strings.Contains(view, "github") {
		t.Errorf("expected 'github' in view, got:\n%s", view)
	}
	if !strings.Contains(view, "Transport:") {
		t.Errorf("expected 'Transport:' label in view, got:\n%s", view)
	}
}

func TestViewEmptyWithZeroSize(t *testing.T) {
	m := NewMCPsModel(nil, newDefaultKeyMap())
	if view := m.View(); view != "" {
		t.Errorf("expected empty string for zero-size view, got %q", view)
	}
}

func TestRenderDetailNoSelection(t *testing.T) {
	m := NewMCPsModel(nil, newDefaultKeyMap())
	detail := m.renderDetail(40, 20)
	if !strings.Contains(detail, "No MCP selected") {
		t.Errorf("expected 'No MCP selected', got:\n%s", detail)
	}
}

func TestRenderDetailShowsFields(t *testing.T) {
	svc := testService()
	m := NewMCPsModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	detail := m.renderDetail(40, 20)

	checks := []string{"github", "Transport:", "Command:", "npx", "Args:", "GITHUB_TOKEN"}
	for _, check := range checks {
		if !strings.Contains(detail, check) {
			t.Errorf("expected %q in detail view, got:\n%s", check, detail)
		}
	}
}

func TestRenderDetailHTTPServer(t *testing.T) {
	svc := testService()
	m := NewMCPsModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	// Navigate to second item (qdrant).
	m, _ = m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyDown}))

	detail := m.renderDetail(40, 20)
	checks := []string{"qdrant", "URL:", "http://localhost:6333/mcp", "Headers:", "Authorization"}
	for _, check := range checks {
		if !strings.Contains(detail, check) {
			t.Errorf("expected %q in qdrant detail view, got:\n%s", check, detail)
		}
	}
}

func TestSortedKeys(t *testing.T) {
	m := map[string]string{"c": "3", "a": "1", "b": "2"}
	keys := sortedKeys(m)
	expected := []string{"a", "b", "c"}
	for i, k := range keys {
		if k != expected[i] {
			t.Errorf("expected key %d to be %q, got %q", i, expected[i], k)
		}
	}
}

func TestCountServerProfileRefs(t *testing.T) {
	reg := &registry.Registry{
		Servers:     registry.NewStore[model.ServerDef, *model.ServerDef]("server"),
		Skills:      registry.NewStore[model.SkillDef, *model.SkillDef]("skill"),
		Hooks:       registry.NewStore[model.HookDef, *model.HookDef]("hook"),
		Permissions: registry.NewStore[model.PermissionRule, *model.PermissionRule]("permission"),
		Templates:   registry.NewStore[model.TemplateDef, *model.TemplateDef]("template"),
		Prompts:     registry.NewStore[model.PromptDef, *model.PromptDef]("prompt"),
		Tags: map[string][]string{
			"core": {"github", "qdrant"},
		},
	}
	_ = reg.Servers.Add(model.ServerDef{
		Name:      "github",
		Transport: model.TransportStdio,
		Command:   "npx",
		Args:      []string{"-y", "@modelcontextprotocol/server-github"},
		Env:       map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
	})
	_ = reg.Servers.Add(model.ServerDef{
		Name:        "qdrant",
		Description: "Qdrant vector database",
		Transport:   model.TransportHTTP,
		URL:         "http://localhost:6333/mcp",
		Headers:     map[string]string{"Authorization": "Bearer ${QDRANT_API_KEY}"},
	})
	store := &project.Store{
		Projects: map[string]model.Project{
			"myproject": {
				Name: "myproject",
				Path: "/tmp/myproject",
				MCPs: []model.MCPAssignment{{Name: "github"}},
			},
			"other": {
				Name: "other",
				Path: "/tmp/other",
				Tags: []string{"core"},
			},
		},
	}
	svc := service.NewForTest(reg, store, nil, nil, "", nil)

	counts := svc.CountServerProfileRefs()
	// github: myproject (MCP) + other (tag)
	if counts["github"] != 2 {
		t.Errorf("expected github count=2, got %d", counts["github"])
	}
	// qdrant: other (tag)
	if counts["qdrant"] != 1 {
		t.Errorf("expected qdrant count=1, got %d", counts["qdrant"])
	}
}

func TestRefreshList(t *testing.T) {
	svc := testService()
	m := NewMCPsModel(svc, newDefaultKeyMap())
	m.SetSize(80, 24)

	// Add a new server to registry.
	_ = svc.AddServer(model.ServerDef{
		Name:      "new-server",
		Transport: model.TransportStdio,
		Command:   "new-cmd",
	})

	m.refreshList()
	if len(m.list.Items()) != 3 {
		t.Errorf("expected 3 items after refresh, got %d", len(m.list.Items()))
	}
}
