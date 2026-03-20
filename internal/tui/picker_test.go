package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lcrostarosa/hystak/internal/deploy"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/project"
	"github.com/lcrostarosa/hystak/internal/registry"
	"github.com/lcrostarosa/hystak/internal/service"
)

func newPickerTestService(projects map[string]model.Project) *service.Service {
	reg := &registry.Registry{
		Servers:     map[string]model.ServerDef{},
		Skills:      map[string]model.SkillDef{},
		Hooks:       map[string]model.HookDef{},
		Permissions: map[string]model.PermissionRule{},
		Templates:   map[string]model.TemplateDef{},
		Tags:        map[string][]string{},
	}
	store := &project.Store{Projects: projects}

	return service.NewForTest(reg, store, map[model.ClientType]deploy.Deployer{}, nil, "", nil)
}

func TestNewPickerModel_WithProjects(t *testing.T) {
	svc := newPickerTestService(map[string]model.Project{
		"proj1": {Name: "proj1", Path: "/tmp/proj1", Clients: []model.ClientType{model.ClientClaudeCode}},
		"proj2": {Name: "proj2", Path: "/tmp/proj2", Clients: []model.ClientType{model.ClientClaudeCode}},
	})

	picker := NewPickerModel(svc)

	// Should have 2 projects + 3 sentinel items = 5 items.
	items := picker.list.Items()
	if len(items) != 5 {
		t.Fatalf("expected 5 items, got %d", len(items))
	}

	// Last three should be sentinel items (bare, configure, manage).
	bare := items[len(items)-3].(pickerItem)
	if bare.kind != pickerBare {
		t.Errorf("expected third-to-last item to be bare launch, got kind %d", bare.kind)
	}

	configure := items[len(items)-2].(pickerItem)
	if configure.kind != pickerConfigure {
		t.Errorf("expected second-to-last item to be configure, got kind %d", configure.kind)
	}

	manage := items[len(items)-1].(pickerItem)
	if manage.kind != pickerManage {
		t.Errorf("expected last item to be manage, got kind %d", manage.kind)
	}
}

func TestNewPickerModel_Empty(t *testing.T) {
	svc := newPickerTestService(map[string]model.Project{})

	picker := NewPickerModel(svc)

	// Should have just 3 sentinel items.
	items := picker.list.Items()
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
}

func TestPickerResult_InitiallyNil(t *testing.T) {
	svc := newPickerTestService(map[string]model.Project{})
	picker := NewPickerModel(svc)

	if picker.Result() != nil {
		t.Error("result should be nil before selection")
	}
}

func TestPickerItem_WithCounts(t *testing.T) {
	svc := newPickerTestService(map[string]model.Project{
		"proj1": {
			Name:    "proj1",
			Path:    "/tmp/proj1",
			Clients: []model.ClientType{model.ClientClaudeCode},
			Skills:  []string{"s1", "s2"},
			Hooks:   []string{"h1"},
		},
	})

	picker := NewPickerModel(svc)
	items := picker.list.Items()

	pi := items[0].(pickerItem)
	if pi.kind != pickerProject {
		t.Errorf("expected pickerProject kind, got %d", pi.kind)
	}
}

func TestPickerItem_NewProjectIndicator(t *testing.T) {
	svc := newPickerTestService(map[string]model.Project{
		"proj1": {
			Name:     "proj1",
			Path:     "/tmp/proj1",
			Launched: false,
			Clients:  []model.ClientType{model.ClientClaudeCode},
		},
	})

	picker := NewPickerModel(svc)
	items := picker.list.Items()

	pi := items[0].(pickerItem)
	if !strings.Contains(pi.name, "(new)") {
		t.Errorf("expected unlaunched project title to contain '(new)', got %q", pi.name)
	}
}

func TestPickerItem_ActiveProfileDisplay(t *testing.T) {
	svc := newPickerTestService(map[string]model.Project{
		"proj1": {
			Name:          "proj1",
			Path:          "/tmp/proj1",
			Launched:      true,
			ActiveProfile: "frontend",
			Clients:       []model.ClientType{model.ClientClaudeCode},
		},
	})

	picker := NewPickerModel(svc)
	items := picker.list.Items()

	pi := items[0].(pickerItem)
	if !strings.Contains(pi.name, "[frontend]") {
		t.Errorf("expected launched project title to contain '[frontend]', got %q", pi.name)
	}
}

func TestPickerItem_LaunchedNoProfile(t *testing.T) {
	svc := newPickerTestService(map[string]model.Project{
		"proj1": {
			Name:     "proj1",
			Path:     "/tmp/proj1",
			Launched: true,
			Clients:  []model.ClientType{model.ClientClaudeCode},
		},
	})

	picker := NewPickerModel(svc)
	items := picker.list.Items()

	pi := items[0].(pickerItem)
	// Should not have (new) or [profile]
	if strings.Contains(pi.name, "(new)") {
		t.Errorf("launched project should not show '(new)', got %q", pi.name)
	}
	if strings.Contains(pi.name, "[") {
		t.Errorf("project with no active profile should not show brackets, got %q", pi.name)
	}
}

func TestPickerResult_ConfigureFlag(t *testing.T) {
	svc := newPickerTestService(map[string]model.Project{
		"proj1": {
			Name:    "proj1",
			Path:    "/tmp/proj1",
			Clients: []model.ClientType{model.ClientClaudeCode},
		},
	})

	picker := NewPickerModel(svc)

	// Simulate window size.
	m, _ := picker.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	picker = m.(PickerModel)

	// Press 'c' while a project is highlighted.
	m, cmd := picker.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'c'}}))
	picker = m.(PickerModel)

	if cmd == nil {
		t.Fatal("expected quit command from 'c' key")
	}
	result := picker.Result()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Configure {
		t.Error("expected Configure=true in result")
	}
}

func TestPickerView_ContainsLogo(t *testing.T) {
	svc := newPickerTestService(map[string]model.Project{})
	picker := NewPickerModel(svc)

	m, _ := picker.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	picker = m.(PickerModel)

	view := picker.View()
	if !strings.Contains(view, "hystak") {
		t.Errorf("expected picker view to contain logo 'hystak', got:\n%s", view)
	}
}

func TestPickerView_ContainsConfigureOption(t *testing.T) {
	svc := newPickerTestService(map[string]model.Project{})
	picker := NewPickerModel(svc)

	m, _ := picker.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	picker = m.(PickerModel)

	view := picker.View()
	if !strings.Contains(view, "Configure") {
		t.Errorf("expected picker view to contain 'Configure' option, got:\n%s", view)
	}
}

func TestPickerView_ContainsHintBar(t *testing.T) {
	svc := newPickerTestService(map[string]model.Project{})
	picker := NewPickerModel(svc)

	m, _ := picker.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	picker = m.(PickerModel)

	view := picker.View()
	if !strings.Contains(view, "configure") {
		t.Errorf("expected picker view to contain hint bar with 'configure', got:\n%s", view)
	}
}
