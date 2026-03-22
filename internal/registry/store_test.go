package registry

import (
	"testing"

	hysterr "github.com/hystak/hystak/internal/errors"
	"github.com/hystak/hystak/internal/model"
)

func TestStore_Add_Get(t *testing.T) {
	s := NewStore[model.ServerDef, *model.ServerDef]("server")

	srv := model.ServerDef{Name: "github", Transport: model.TransportStdio, Command: "npx"}
	if err := s.Add(srv); err != nil {
		t.Fatal(err)
	}

	got, ok := s.Get("github")
	if !ok {
		t.Fatal("Get returned false")
	}
	if got.Name != "github" {
		t.Errorf("Name = %q, want %q", got.Name, "github")
	}
	if got.Command != "npx" {
		t.Errorf("Command = %q, want %q", got.Command, "npx")
	}
}

func TestStore_Add_Duplicate(t *testing.T) {
	s := NewStore[model.ServerDef, *model.ServerDef]("server")

	srv := model.ServerDef{Name: "github", Transport: model.TransportStdio, Command: "npx"}
	if err := s.Add(srv); err != nil {
		t.Fatal(err)
	}

	err := s.Add(srv)
	if err == nil {
		t.Fatal("expected error for duplicate add, got nil")
	}
	if _, ok := err.(*hysterr.AlreadyExists); !ok {
		t.Errorf("expected *AlreadyExists, got %T: %v", err, err)
	}
}

func TestStore_Add_EmptyName(t *testing.T) {
	s := NewStore[model.ServerDef, *model.ServerDef]("server")

	err := s.Add(model.ServerDef{Transport: model.TransportStdio})
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if _, ok := err.(*hysterr.ValidationError); !ok {
		t.Errorf("expected *ValidationError, got %T: %v", err, err)
	}
}

func TestStore_Get_NotFound(t *testing.T) {
	s := NewStore[model.ServerDef, *model.ServerDef]("server")

	_, ok := s.Get("nonexistent")
	if ok {
		t.Error("Get returned true for nonexistent item")
	}
}

func TestStore_Update(t *testing.T) {
	s := NewStore[model.ServerDef, *model.ServerDef]("server")

	srv := model.ServerDef{Name: "github", Transport: model.TransportStdio, Command: "npx"}
	if err := s.Add(srv); err != nil {
		t.Fatal(err)
	}

	updated := model.ServerDef{Name: "github", Transport: model.TransportStdio, Command: "node"}
	if err := s.Update(updated); err != nil {
		t.Fatal(err)
	}

	got, ok := s.Get("github")
	if !ok {
		t.Fatal("Get returned false after update")
	}
	if got.Command != "node" {
		t.Errorf("Command = %q, want %q", got.Command, "node")
	}
}

func TestStore_Update_NotFound(t *testing.T) {
	s := NewStore[model.ServerDef, *model.ServerDef]("server")

	err := s.Update(model.ServerDef{Name: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for update of nonexistent item, got nil")
	}
	if _, ok := err.(*hysterr.ResourceNotFound); !ok {
		t.Errorf("expected *ResourceNotFound, got %T: %v", err, err)
	}
}

func TestStore_Delete(t *testing.T) {
	s := NewStore[model.ServerDef, *model.ServerDef]("server")

	if err := s.Add(model.ServerDef{Name: "github", Transport: model.TransportStdio}); err != nil {
		t.Fatal(err)
	}

	if err := s.Delete("github"); err != nil {
		t.Fatal(err)
	}

	_, ok := s.Get("github")
	if ok {
		t.Error("Get returned true after delete")
	}
	if s.Len() != 0 {
		t.Errorf("Len() = %d, want 0", s.Len())
	}
}

func TestStore_Delete_NotFound(t *testing.T) {
	s := NewStore[model.ServerDef, *model.ServerDef]("server")

	err := s.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected error for delete of nonexistent item, got nil")
	}
	if _, ok := err.(*hysterr.ResourceNotFound); !ok {
		t.Errorf("expected *ResourceNotFound, got %T: %v", err, err)
	}
}

func TestStore_List_SortedByName(t *testing.T) {
	s := NewStore[model.ServerDef, *model.ServerDef]("server")

	if err := s.Add(model.ServerDef{Name: "postgres", Transport: model.TransportStdio}); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(model.ServerDef{Name: "github", Transport: model.TransportStdio}); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(model.ServerDef{Name: "slack", Transport: model.TransportStdio}); err != nil {
		t.Fatal(err)
	}

	list := s.List()
	if len(list) != 3 {
		t.Fatalf("List() returned %d items, want 3", len(list))
	}
	want := []string{"github", "postgres", "slack"}
	for i, name := range want {
		if list[i].Name != name {
			t.Errorf("List()[%d].Name = %q, want %q", i, list[i].Name, name)
		}
	}
}

func TestStore_List_CustomSort(t *testing.T) {
	s := NewStore[model.PromptDef, *model.PromptDef]("prompt").WithSort(
		func(a, b model.PromptDef) int { return a.Order - b.Order },
	)

	if err := s.Add(model.PromptDef{Name: "style", Order: 20, Source: "/s"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(model.PromptDef{Name: "security", Order: 10, Source: "/s"}); err != nil {
		t.Fatal(err)
	}

	list := s.List()
	if len(list) != 2 {
		t.Fatalf("List() returned %d items, want 2", len(list))
	}
	if list[0].Name != "security" {
		t.Errorf("List()[0].Name = %q, want %q", list[0].Name, "security")
	}
	if list[1].Name != "style" {
		t.Errorf("List()[1].Name = %q, want %q", list[1].Name, "style")
	}
}

func TestStore_Items_ReturnsShallowCopy(t *testing.T) {
	s := NewStore[model.ServerDef, *model.ServerDef]("server")
	if err := s.Add(model.ServerDef{Name: "github", Transport: model.TransportStdio}); err != nil {
		t.Fatal(err)
	}

	items := s.Items()
	// Mutate the copy
	delete(items, "github")

	// Original should be unchanged
	if s.Len() != 1 {
		t.Errorf("Len() = %d after mutating Items() copy, want 1", s.Len())
	}
}

func TestStore_SetItems(t *testing.T) {
	s := NewStore[model.ServerDef, *model.ServerDef]("server")

	items := map[string]model.ServerDef{
		"github":   {Transport: model.TransportStdio, Command: "npx"},
		"postgres": {Transport: model.TransportStdio, Command: "npx"},
	}
	s.SetItems(items)

	if s.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", s.Len())
	}

	// SetItems should have set the Name from the map key
	got, ok := s.Get("github")
	if !ok {
		t.Fatal("Get(github) returned false")
	}
	if got.Name != "github" {
		t.Errorf("Name = %q, want %q (SetItems should set name from key)", got.Name, "github")
	}
}

func TestStore_Len(t *testing.T) {
	s := NewStore[model.ServerDef, *model.ServerDef]("server")

	if s.Len() != 0 {
		t.Errorf("empty store Len() = %d, want 0", s.Len())
	}

	if err := s.Add(model.ServerDef{Name: "a", Transport: model.TransportStdio}); err != nil {
		t.Fatal(err)
	}
	if s.Len() != 1 {
		t.Errorf("after add Len() = %d, want 1", s.Len())
	}
}

func TestStore_Kind(t *testing.T) {
	s := NewStore[model.SkillDef, *model.SkillDef]("skill")
	if got := s.Kind(); got != "skill" {
		t.Errorf("Kind() = %q, want %q", got, "skill")
	}
}
