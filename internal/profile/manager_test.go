package profile

import (
	"reflect"
	"slices"
	"testing"

	hysterr "github.com/hystak/hystak/internal/errors"
	"github.com/hystak/hystak/internal/model"
)

func TestManager_SaveLoad_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	m := NewManager(tmp)

	original := model.ProjectProfile{
		Name:        "dev",
		Description: "Full development environment",
		Scope:       "global",
		MCPs: []model.MCPAssignment{
			{Name: "github"},
			{Name: "postgres"},
		},
		Skills:      []string{"code-review", "commit"},
		Hooks:       []string{"lint-on-edit"},
		Permissions: []string{"allow-bash", "deny-rm"},
		Template:    "standard",
		Prompts:     []string{"security-rules", "style-guide"},
		Env:         map[string]string{"NODE_ENV": "development"},
		Isolation:   model.IsolationNone,
	}

	if err := m.Save(original); err != nil {
		t.Fatal(err)
	}

	loaded, err := m.Load("dev")
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(loaded, original) {
		t.Errorf("round-trip mismatch:\n  got:  %+v\n  want: %+v", loaded, original)
	}
}

func TestManager_Save_EmptyName(t *testing.T) {
	tmp := t.TempDir()
	m := NewManager(tmp)

	err := m.Save(model.ProjectProfile{})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, ok := err.(*hysterr.ValidationError); !ok {
		t.Errorf("expected *ValidationError, got %T", err)
	}
}

func TestManager_Save_PathTraversal(t *testing.T) {
	tmp := t.TempDir()
	m := NewManager(tmp)

	tests := []string{
		"../../etc/passwd",
		"../evil",
		"foo/../../bar",
	}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			err := m.Save(model.ProjectProfile{Name: name})
			if err == nil {
				t.Fatal("expected error for path traversal")
			}
			if _, ok := err.(*hysterr.ValidationError); !ok {
				t.Errorf("expected *ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestManager_Load_NotFound(t *testing.T) {
	tmp := t.TempDir()
	m := NewManager(tmp)

	_, err := m.Load("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing profile")
	}
	if _, ok := err.(*hysterr.ResourceNotFound); !ok {
		t.Errorf("expected *ResourceNotFound, got %T", err)
	}
}

func TestManager_Load_PathTraversal(t *testing.T) {
	tmp := t.TempDir()
	m := NewManager(tmp)

	_, err := m.Load("../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
	if _, ok := err.(*hysterr.ValidationError); !ok {
		t.Errorf("expected *ValidationError, got %T: %v", err, err)
	}
}

func TestManager_Load_BuiltInEmpty(t *testing.T) {
	tmp := t.TempDir()
	m := NewManager(tmp)

	p, err := m.Load("empty")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "empty" {
		t.Errorf("Name = %q, want empty", p.Name)
	}
	if p.Description == "" {
		t.Error("built-in empty profile should have a description")
	}
	if len(p.MCPs) != 0 {
		t.Errorf("MCPs should be empty, got %v", p.MCPs)
	}
	if p.Isolation != model.IsolationNone {
		t.Errorf("Isolation = %q, want none", p.Isolation)
	}
}

func TestManager_Delete(t *testing.T) {
	tmp := t.TempDir()
	m := NewManager(tmp)

	if err := m.Save(model.ProjectProfile{Name: "dev", Isolation: model.IsolationNone}); err != nil {
		t.Fatal(err)
	}
	if err := m.Delete("dev"); err != nil {
		t.Fatal(err)
	}
	_, err := m.Load("dev")
	if err == nil {
		t.Error("profile should not be loadable after delete")
	}
}

func TestManager_Delete_NotFound(t *testing.T) {
	tmp := t.TempDir()
	m := NewManager(tmp)

	err := m.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*hysterr.ResourceNotFound); !ok {
		t.Errorf("expected *ResourceNotFound, got %T", err)
	}
}

func TestManager_Delete_PathTraversal(t *testing.T) {
	tmp := t.TempDir()
	m := NewManager(tmp)

	err := m.Delete("../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
	if _, ok := err.(*hysterr.ValidationError); !ok {
		t.Errorf("expected *ValidationError, got %T: %v", err, err)
	}
}

func TestManager_List_IncludesEmpty(t *testing.T) {
	tmp := t.TempDir()
	m := NewManager(tmp)

	if err := m.Save(model.ProjectProfile{Name: "dev", Isolation: model.IsolationNone}); err != nil {
		t.Fatal(err)
	}

	names, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(names, "empty") {
		t.Errorf("List() = %v, should include 'empty'", names)
	}
	if !slices.Contains(names, "dev") {
		t.Errorf("List() = %v, should include 'dev'", names)
	}
}

func TestManager_List_EmptyDir(t *testing.T) {
	tmp := t.TempDir()
	m := NewManager(tmp)

	names, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"empty"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("List() = %v, want %v", names, want)
	}
}

func TestManager_List_NonexistentDir(t *testing.T) {
	m := NewManager("/nonexistent/profiles")

	names, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"empty"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("List() = %v, want %v", names, want)
	}
}

func TestManager_List_Sorted(t *testing.T) {
	tmp := t.TempDir()
	m := NewManager(tmp)

	if err := m.Save(model.ProjectProfile{Name: "zebra", Isolation: model.IsolationNone}); err != nil {
		t.Fatal(err)
	}
	if err := m.Save(model.ProjectProfile{Name: "alpha", Isolation: model.IsolationNone}); err != nil {
		t.Fatal(err)
	}

	names, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alpha", "empty", "zebra"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("List() = %v, want %v", names, want)
	}
}

func TestManager_Exists(t *testing.T) {
	tmp := t.TempDir()
	m := NewManager(tmp)

	exists, err := m.Exists("empty")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("built-in empty should always exist")
	}

	exists, err = m.Exists("dev")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("dev should not exist before save")
	}

	if err := m.Save(model.ProjectProfile{Name: "dev", Isolation: model.IsolationNone}); err != nil {
		t.Fatal(err)
	}
	exists, err = m.Exists("dev")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("dev should exist after save")
	}
}

func TestManager_Exists_PathTraversal(t *testing.T) {
	tmp := t.TempDir()
	m := NewManager(tmp)

	_, err := m.Exists("../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
	if _, ok := err.(*hysterr.ValidationError); !ok {
		t.Errorf("expected *ValidationError, got %T: %v", err, err)
	}
}

func TestManager_LoadAll(t *testing.T) {
	tmp := t.TempDir()
	m := NewManager(tmp)

	if err := m.Save(model.ProjectProfile{Name: "dev", Description: "Dev env", Isolation: model.IsolationNone}); err != nil {
		t.Fatal(err)
	}

	profiles, err := m.LoadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 2 {
		t.Fatalf("LoadAll() = %d profiles, want 2", len(profiles))
	}
	names := []string{profiles[0].Name, profiles[1].Name}
	if !slices.Contains(names, "dev") {
		t.Error("LoadAll should include dev")
	}
	if !slices.Contains(names, "empty") {
		t.Error("LoadAll should include empty")
	}
}

func TestManager_SaveLoad_WithOverrides(t *testing.T) {
	tmp := t.TempDir()
	m := NewManager(tmp)

	cmd := "node"
	original := model.ProjectProfile{
		Name: "custom",
		MCPs: []model.MCPAssignment{
			{Name: "github"},
			{
				Name: "remote-api",
				Overrides: &model.ServerOverride{
					Command: &cmd,
					Env:     map[string]string{"KEY": "val"},
				},
			},
		},
		Isolation: model.IsolationWorktree,
	}

	if err := m.Save(original); err != nil {
		t.Fatal(err)
	}

	loaded, err := m.Load("custom")
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded.MCPs) != 2 {
		t.Fatalf("MCPs count = %d, want 2", len(loaded.MCPs))
	}
	if loaded.MCPs[0].Name != "github" {
		t.Errorf("MCPs[0].Name = %q, want github", loaded.MCPs[0].Name)
	}
	if loaded.MCPs[1].Overrides == nil {
		t.Fatal("MCPs[1].Overrides should not be nil")
	}
	if loaded.MCPs[1].Overrides.Command == nil || *loaded.MCPs[1].Overrides.Command != "node" {
		t.Errorf("MCPs[1].Overrides.Command mismatch")
	}
}

func TestManager_Save_Overwrite(t *testing.T) {
	tmp := t.TempDir()
	m := NewManager(tmp)

	if err := m.Save(model.ProjectProfile{Name: "dev", Description: "v1", Isolation: model.IsolationNone}); err != nil {
		t.Fatal(err)
	}
	if err := m.Save(model.ProjectProfile{Name: "dev", Description: "v2", Isolation: model.IsolationNone}); err != nil {
		t.Fatal(err)
	}

	loaded, err := m.Load("dev")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Description != "v2" {
		t.Errorf("Description = %q, want v2", loaded.Description)
	}
}

func TestEmptyProfile(t *testing.T) {
	p := EmptyProfile()
	if p.Name != "empty" {
		t.Errorf("Name = %q, want empty", p.Name)
	}
	if p.Scope != "global" {
		t.Errorf("Scope = %q, want global", p.Scope)
	}
	if len(p.MCPs) != 0 || len(p.Skills) != 0 || len(p.Hooks) != 0 || len(p.Permissions) != 0 {
		t.Error("empty profile should have zero resources")
	}
	if p.Isolation != model.IsolationNone {
		t.Errorf("Isolation = %q, want none", p.Isolation)
	}
}
