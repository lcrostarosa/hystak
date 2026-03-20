package tui

import (
	"testing"

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

	// Should have 2 projects + 2 sentinel items = 4 items.
	items := picker.list.Items()
	if len(items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(items))
	}

	// Last two should be sentinel items.
	bare := items[len(items)-2].(pickerItem)
	if bare.kind != pickerBare {
		t.Errorf("expected second-to-last item to be bare launch, got kind %d", bare.kind)
	}

	manage := items[len(items)-1].(pickerItem)
	if manage.kind != pickerManage {
		t.Errorf("expected last item to be manage, got kind %d", manage.kind)
	}
}

func TestNewPickerModel_Empty(t *testing.T) {
	svc := newPickerTestService(map[string]model.Project{})

	picker := NewPickerModel(svc)

	// Should have just 2 sentinel items.
	items := picker.list.Items()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
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

	// First item should be proj1 with description showing skill/hook counts.
	pi := items[0].(pickerItem)
	if pi.name != "proj1" {
		t.Errorf("expected item name 'proj1', got %q", pi.name)
	}
	if pi.kind != pickerProject {
		t.Errorf("expected pickerProject kind, got %d", pi.kind)
	}
}
