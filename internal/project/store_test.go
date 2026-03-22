package project

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	hysterr "github.com/hystak/hystak/internal/errors"
	"github.com/hystak/hystak/internal/model"
)

func TestStore_Add_Get(t *testing.T) {
	s := NewStore()
	p := model.Project{Name: "myproject", Path: "/test/myproject"}
	if err := s.Add(p); err != nil {
		t.Fatal(err)
	}

	got, ok := s.Get("myproject")
	if !ok {
		t.Fatal("Get returned false")
	}
	if got.Name != "myproject" || got.Path != "/test/myproject" {
		t.Errorf("got %+v, want name=myproject path=/test/myproject", got)
	}
}

func TestStore_Add_Duplicate(t *testing.T) {
	s := NewStore()
	p := model.Project{Name: "myproject", Path: "/test/myproject"}
	if err := s.Add(p); err != nil {
		t.Fatal(err)
	}
	err := s.Add(p)
	if err == nil {
		t.Fatal("expected error for duplicate")
	}
	if _, ok := err.(*hysterr.AlreadyExists); !ok {
		t.Errorf("expected *AlreadyExists, got %T", err)
	}
}

func TestStore_Add_EmptyName(t *testing.T) {
	s := NewStore()
	err := s.Add(model.Project{Path: "/test"})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, ok := err.(*hysterr.ValidationError); !ok {
		t.Errorf("expected *ValidationError, got %T", err)
	}
}

func TestStore_Add_EmptyPath(t *testing.T) {
	s := NewStore()
	err := s.Add(model.Project{Name: "myproject"})
	if err == nil {
		t.Fatal("expected error for empty path")
	}
	if _, ok := err.(*hysterr.ValidationError); !ok {
		t.Errorf("expected *ValidationError, got %T", err)
	}
}

func TestStore_Get_NotFound(t *testing.T) {
	s := NewStore()
	_, ok := s.Get("nonexistent")
	if ok {
		t.Error("Get returned true for nonexistent project")
	}
}

func TestStore_Update(t *testing.T) {
	s := NewStore()
	if err := s.Add(model.Project{Name: "myproject", Path: "/old"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(model.Project{Name: "myproject", Path: "/new"}); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get("myproject")
	if got.Path != "/new" {
		t.Errorf("Path = %q, want /new", got.Path)
	}
}

func TestStore_Update_NotFound(t *testing.T) {
	s := NewStore()
	err := s.Update(model.Project{Name: "nonexistent", Path: "/x"})
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*hysterr.ResourceNotFound); !ok {
		t.Errorf("expected *ResourceNotFound, got %T", err)
	}
}

func TestStore_Delete(t *testing.T) {
	s := NewStore()
	if err := s.Add(model.Project{Name: "myproject", Path: "/test"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("myproject"); err != nil {
		t.Fatal(err)
	}
	if s.Len() != 0 {
		t.Errorf("Len() = %d, want 0", s.Len())
	}
}

func TestStore_Delete_NotFound(t *testing.T) {
	s := NewStore()
	err := s.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*hysterr.ResourceNotFound); !ok {
		t.Errorf("expected *ResourceNotFound, got %T", err)
	}
}

func TestStore_List_SortedByName(t *testing.T) {
	s := NewStore()
	if err := s.Add(model.Project{Name: "zebra", Path: "/z"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(model.Project{Name: "alpha", Path: "/a"}); err != nil {
		t.Fatal(err)
	}
	list := s.List()
	if len(list) != 2 {
		t.Fatalf("List() = %d items, want 2", len(list))
	}
	if list[0].Name != "alpha" || list[1].Name != "zebra" {
		t.Errorf("List() not sorted: %v", list)
	}
}

func TestStore_SetActiveProfile(t *testing.T) {
	s := NewStore()
	if err := s.Add(model.Project{Name: "myproject", Path: "/test"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetActiveProfile("myproject", "dev"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get("myproject")
	if got.ActiveProfile != "dev" {
		t.Errorf("ActiveProfile = %q, want dev", got.ActiveProfile)
	}
}

func TestStore_SetActiveProfile_NotFound(t *testing.T) {
	s := NewStore()
	err := s.SetActiveProfile("nonexistent", "dev")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStore_SetManagedMCPs(t *testing.T) {
	s := NewStore()
	if err := s.Add(model.Project{Name: "myproject", Path: "/test"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetManagedMCPs("myproject", []string{"github", "postgres"}); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get("myproject")
	want := []string{"github", "postgres"}
	if !reflect.DeepEqual(got.ManagedMCPs, want) {
		t.Errorf("ManagedMCPs = %v, want %v", got.ManagedMCPs, want)
	}
}

func TestStore_SetManagedMCPs_ReturnsCopy(t *testing.T) {
	s := NewStore()
	if err := s.Add(model.Project{Name: "myproject", Path: "/test"}); err != nil {
		t.Fatal(err)
	}
	input := []string{"github"}
	if err := s.SetManagedMCPs("myproject", input); err != nil {
		t.Fatal(err)
	}
	input[0] = "mutated"
	got, _ := s.Get("myproject")
	if got.ManagedMCPs[0] != "github" {
		t.Error("SetManagedMCPs did not copy the input slice")
	}
}

func TestStore_FindByPath(t *testing.T) {
	s := NewStore()
	if err := s.Add(model.Project{Name: "myproject", Path: "/test/myproject"}); err != nil {
		t.Fatal(err)
	}
	got, ok := s.FindByPath("/test/myproject")
	if !ok {
		t.Fatal("FindByPath returned false")
	}
	if got.Name != "myproject" {
		t.Errorf("Name = %q, want myproject", got.Name)
	}
}

func TestStore_FindByPath_NotFound(t *testing.T) {
	s := NewStore()
	_, ok := s.FindByPath("/nonexistent")
	if ok {
		t.Error("FindByPath returned true for nonexistent path")
	}
}

func TestStore_SaveLoad_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "projects.yaml")

	original := NewStore()
	origProject := model.Project{
		Name:          "interview-platform",
		Path:          "/test/interview-platform",
		ActiveProfile: "dev",
		ManagedMCPs:   []string{"github", "postgres"},
	}
	if err := original.Add(origProject); err != nil {
		t.Fatal(err)
	}
	if err := original.Add(model.Project{
		Name:          "personal-site",
		Path:          "/test/personal-site",
		ActiveProfile: "default",
		ManagedMCPs:   []string{"github"},
	}); err != nil {
		t.Fatal(err)
	}

	if err := original.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("count", func(t *testing.T) {
		if loaded.Len() != 2 {
			t.Fatalf("Len() = %d, want 2", loaded.Len())
		}
	})

	t.Run("interview-platform", func(t *testing.T) {
		got, ok := loaded.Get("interview-platform")
		if !ok {
			t.Fatal("missing project")
		}
		if !reflect.DeepEqual(got, origProject) {
			t.Errorf("mismatch:\n  got:  %+v\n  want: %+v", got, origProject)
		}
	})

	t.Run("personal-site", func(t *testing.T) {
		got, ok := loaded.Get("personal-site")
		if !ok {
			t.Fatal("missing project")
		}
		if got.Path != "/test/personal-site" {
			t.Errorf("Path = %q", got.Path)
		}
		if got.ActiveProfile != "default" {
			t.Errorf("ActiveProfile = %q", got.ActiveProfile)
		}
	})
}

func TestStore_Load_NonexistentFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "nonexistent.yaml")

	store, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if store.Len() != 0 {
		t.Error("loading nonexistent file should return empty store")
	}
}

func TestStore_Load_MalformedYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "projects.yaml")

	if err := writeFile(t, path, "projects: [bad"); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}

func writeFile(t *testing.T, path, content string) error {
	t.Helper()
	return os.WriteFile(path, []byte(content), 0o644)
}
