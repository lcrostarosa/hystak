package registry

import (
	"testing"

	"github.com/lcrostarosa/hystak/internal/model"
)

func TestStore_Add_Get(t *testing.T) {
	s := NewStore[model.SkillDef, *model.SkillDef]("skill")

	skill := model.SkillDef{Name: "code-review", Description: "reviews code", Source: "/tmp/cr.md"}
	if err := s.Add(skill); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, ok := s.Get("code-review")
	if !ok {
		t.Fatal("Get returned false")
	}
	if got.Description != "reviews code" {
		t.Errorf("Description = %q", got.Description)
	}
}

func TestStore_Add_Duplicate(t *testing.T) {
	s := NewStore[model.SkillDef, *model.SkillDef]("skill")

	skill := model.SkillDef{Name: "x", Source: "/tmp/x.md"}
	_ = s.Add(skill)

	if err := s.Add(skill); err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestStore_Update(t *testing.T) {
	s := NewStore[model.SkillDef, *model.SkillDef]("skill")

	_ = s.Add(model.SkillDef{Name: "x", Source: "/old"})

	err := s.Update("x", model.SkillDef{Source: "/new"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := s.Get("x")
	if got.Source != "/new" {
		t.Errorf("Source = %q", got.Source)
	}
	if got.Name != "x" {
		t.Errorf("Name should be preserved after update, got %q", got.Name)
	}
}

func TestStore_Update_NotFound(t *testing.T) {
	s := NewStore[model.SkillDef, *model.SkillDef]("skill")
	if err := s.Update("nope", model.SkillDef{}); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestStore_Delete(t *testing.T) {
	s := NewStore[model.SkillDef, *model.SkillDef]("skill")
	_ = s.Add(model.SkillDef{Name: "x", Source: "/tmp"})

	if err := s.Delete("x"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if s.Len() != 0 {
		t.Errorf("expected empty store, got %d", s.Len())
	}
}

func TestStore_Delete_NotFound(t *testing.T) {
	s := NewStore[model.SkillDef, *model.SkillDef]("skill")
	if err := s.Delete("nope"); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestStore_List_SortedByName(t *testing.T) {
	s := NewStore[model.SkillDef, *model.SkillDef]("skill")
	_ = s.Add(model.SkillDef{Name: "charlie", Source: "/c"})
	_ = s.Add(model.SkillDef{Name: "alpha", Source: "/a"})
	_ = s.Add(model.SkillDef{Name: "bravo", Source: "/b"})

	items := s.List()
	if len(items) != 3 {
		t.Fatalf("expected 3, got %d", len(items))
	}
	if items[0].Name != "alpha" || items[1].Name != "bravo" || items[2].Name != "charlie" {
		t.Errorf("sort order wrong: %v", items)
	}
}

func TestStore_WithSort_Custom(t *testing.T) {
	s := NewStore[model.PromptDef, *model.PromptDef]("prompt").WithSort(func(a, b model.PromptDef) bool {
		if a.Order != b.Order {
			return a.Order < b.Order
		}
		return a.Name < b.Name
	})
	_ = s.Add(model.PromptDef{Name: "b", Order: 1, Source: "/b"})
	_ = s.Add(model.PromptDef{Name: "a", Order: 2, Source: "/a"})
	_ = s.Add(model.PromptDef{Name: "c", Order: 1, Source: "/c"})

	items := s.List()
	if items[0].Name != "b" || items[1].Name != "c" || items[2].Name != "a" {
		t.Errorf("custom sort wrong: %v", items)
	}
}

func TestStore_SetItems_PopulatesNames(t *testing.T) {
	s := NewStore[model.SkillDef, *model.SkillDef]("skill")
	s.SetItems(map[string]model.SkillDef{
		"my-skill": {Source: "/tmp/s.md"},
	})

	got, ok := s.Get("my-skill")
	if !ok {
		t.Fatal("Get returned false after SetItems")
	}
	if got.Name != "my-skill" {
		t.Errorf("Name = %q, want %q", got.Name, "my-skill")
	}
}

func TestStore_SetItems_Nil(t *testing.T) {
	s := NewStore[model.SkillDef, *model.SkillDef]("skill")
	_ = s.Add(model.SkillDef{Name: "x", Source: "/x"})

	s.SetItems(nil)
	if s.Len() != 0 {
		t.Errorf("expected empty after SetItems(nil), got %d", s.Len())
	}
}

func TestStore_Items_RoundTrip(t *testing.T) {
	s := NewStore[model.SkillDef, *model.SkillDef]("skill")
	_ = s.Add(model.SkillDef{Name: "x", Source: "/x"})

	items := s.Items()
	s2 := NewStore[model.SkillDef, *model.SkillDef]("skill")
	s2.SetItems(items)

	got, ok := s2.Get("x")
	if !ok || got.Source != "/x" {
		t.Errorf("round-trip failed: ok=%v, got=%+v", ok, got)
	}
}

func TestStore_ServerDef(t *testing.T) {
	s := NewStore[model.ServerDef, *model.ServerDef]("server")
	srv := model.ServerDef{
		Name:      "github",
		Transport: model.TransportStdio,
		Command:   "gh-mcp",
	}
	if err := s.Add(srv); err != nil {
		t.Fatalf("Add server: %v", err)
	}

	got, ok := s.Get("github")
	if !ok {
		t.Fatal("Get returned false")
	}
	if got.Command != "gh-mcp" {
		t.Errorf("Command = %q", got.Command)
	}
}
