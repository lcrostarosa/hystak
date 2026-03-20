package profile

import (
	"os"
	"path/filepath"
	"testing"

	hysterr "github.com/lcrostarosa/hystak/internal/errors"
	"gopkg.in/yaml.v3"
)

func tempManager(t *testing.T) *Manager {
	t.Helper()
	dir := t.TempDir()
	return NewManager(filepath.Join(dir, "profiles"))
}

func TestVanilla(t *testing.T) {
	v := Vanilla()
	if v.Name != VanillaName {
		t.Errorf("Vanilla().Name = %q, want %q", v.Name, VanillaName)
	}
	if !v.IsVanilla() {
		t.Error("Vanilla().IsVanilla() = false, want true")
	}
	if !v.IsEmpty() {
		t.Error("Vanilla().IsEmpty() = false, want true")
	}
}

func TestCRUDRoundTrip(t *testing.T) {
	m := tempManager(t)

	p := Profile{
		Name:        "frontend",
		Description: "Frontend dev loadout",
		MCPs:        []string{"browser-mcp", "figma-mcp", "github"},
		Skills:      []string{"react-patterns", "css-review"},
		Hooks:       []string{"lint-frontend"},
		Permissions: []string{"allow-npm", "allow-browser"},
		EnvVars:     map[string]string{"NODE_ENV": "development"},
		ClaudeMD:    "frontend-instructions",
		Isolation:   IsolationNone,
	}

	// Save
	if err := m.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Get
	got, err := m.Get("frontend")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Name != p.Name {
		t.Errorf("Name = %q, want %q", got.Name, p.Name)
	}
	if got.Description != p.Description {
		t.Errorf("Description = %q, want %q", got.Description, p.Description)
	}
	if len(got.MCPs) != 3 {
		t.Errorf("len(MCPs) = %d, want 3", len(got.MCPs))
	}
	if len(got.Skills) != 2 {
		t.Errorf("len(Skills) = %d, want 2", len(got.Skills))
	}
	if len(got.Hooks) != 1 {
		t.Errorf("len(Hooks) = %d, want 1", len(got.Hooks))
	}
	if len(got.Permissions) != 2 {
		t.Errorf("len(Permissions) = %d, want 2", len(got.Permissions))
	}
	if got.EnvVars["NODE_ENV"] != "development" {
		t.Errorf("EnvVars[NODE_ENV] = %q, want %q", got.EnvVars["NODE_ENV"], "development")
	}
	if got.ClaudeMD != "frontend-instructions" {
		t.Errorf("ClaudeMD = %q, want %q", got.ClaudeMD, "frontend-instructions")
	}
	if got.Isolation != IsolationNone {
		t.Errorf("Isolation = %q, want %q", got.Isolation, IsolationNone)
	}

	// Update
	p.Description = "Updated description"
	p.MCPs = []string{"browser-mcp"}
	if err := m.Save(p); err != nil {
		t.Fatalf("Save (update): %v", err)
	}
	got, err = m.Get("frontend")
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Description != "Updated description" {
		t.Errorf("Description after update = %q, want %q", got.Description, "Updated description")
	}
	if len(got.MCPs) != 1 {
		t.Errorf("len(MCPs) after update = %d, want 1", len(got.MCPs))
	}

	// Delete
	if err := m.Delete("frontend"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = m.Get("frontend")
	if !hysterr.IsNotFound(err) {
		t.Errorf("Get after delete: got %v, want NotFoundError", err)
	}
}

func TestVanillaAlwaysExists(t *testing.T) {
	m := tempManager(t)

	p, err := m.Get(VanillaName)
	if err != nil {
		t.Fatalf("Get vanilla: %v", err)
	}
	if !p.IsVanilla() {
		t.Error("Get vanilla: IsVanilla() = false")
	}
	if !p.IsEmpty() {
		t.Error("Get vanilla: IsEmpty() = false")
	}
	if !m.Exists(VanillaName) {
		t.Error("Exists(vanilla) = false")
	}
}

func TestCannotModifyVanilla(t *testing.T) {
	m := tempManager(t)

	err := m.Save(Profile{Name: VanillaName})
	if err == nil {
		t.Error("Save vanilla: expected error, got nil")
	}

	err = m.Delete(VanillaName)
	if err == nil {
		t.Error("Delete vanilla: expected error, got nil")
	}
}

func TestValidateEmptyName(t *testing.T) {
	m := tempManager(t)
	err := m.Save(Profile{})
	if err == nil {
		t.Error("Save empty name: expected error, got nil")
	}
}

func TestGetNotFound(t *testing.T) {
	m := tempManager(t)
	_, err := m.Get("nonexistent")
	if !hysterr.IsNotFound(err) {
		t.Errorf("Get nonexistent: got %v, want NotFoundError", err)
	}
}

func TestDeleteNotFound(t *testing.T) {
	m := tempManager(t)
	err := m.Delete("nonexistent")
	if !hysterr.IsNotFound(err) {
		t.Errorf("Delete nonexistent: got %v, want NotFoundError", err)
	}
}

func TestList(t *testing.T) {
	m := tempManager(t)

	// Empty dir: only vanilla
	profiles, err := m.List()
	if err != nil {
		t.Fatalf("List empty: %v", err)
	}
	if len(profiles) != 1 || profiles[0].Name != VanillaName {
		t.Errorf("List empty: got %d profiles, want 1 (vanilla)", len(profiles))
	}

	// Add two profiles
	if err := m.Save(Profile{Name: "backend", MCPs: []string{"db-mcp"}}); err != nil {
		t.Fatalf("Save backend: %v", err)
	}
	if err := m.Save(Profile{Name: "frontend", MCPs: []string{"browser-mcp"}}); err != nil {
		t.Fatalf("Save frontend: %v", err)
	}

	profiles, err = m.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	// Should be sorted: backend, frontend, vanilla
	if len(profiles) != 3 {
		t.Fatalf("List: got %d profiles, want 3", len(profiles))
	}
	names := make([]string, len(profiles))
	for i, p := range profiles {
		names[i] = p.Name
	}
	want := []string{"backend", "frontend", VanillaName}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("List[%d].Name = %q, want %q", i, n, want[i])
		}
	}
}

func TestListNoDir(t *testing.T) {
	m := NewManager("/nonexistent/path/profiles")
	profiles, err := m.List()
	if err != nil {
		t.Fatalf("List no dir: %v", err)
	}
	if len(profiles) != 1 || profiles[0].Name != VanillaName {
		t.Errorf("List no dir: got %d profiles, want 1 (vanilla)", len(profiles))
	}
}

func TestExists(t *testing.T) {
	m := tempManager(t)

	if m.Exists("frontend") {
		t.Error("Exists before save: got true, want false")
	}

	if err := m.Save(Profile{Name: "frontend"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if !m.Exists("frontend") {
		t.Error("Exists after save: got false, want true")
	}
}

func TestYAMLMarshalRoundTrip(t *testing.T) {
	p := Profile{
		Name:        "test",
		Description: "Test profile",
		MCPs:        []string{"a", "b"},
		Skills:      []string{"c"},
		Hooks:       []string{"d"},
		Permissions: []string{"e"},
		EnvVars:     map[string]string{"K": "V"},
		ClaudeMD:    "tmpl",
		Isolation:   IsolationWorktree,
	}

	data, err := yaml.Marshal(toFile(p))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var f profileFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	got := fromFile(f)
	if got.Name != p.Name {
		t.Errorf("Name = %q, want %q", got.Name, p.Name)
	}
	if got.Isolation != IsolationWorktree {
		t.Errorf("Isolation = %q, want %q", got.Isolation, IsolationWorktree)
	}
	if len(got.MCPs) != 2 {
		t.Errorf("len(MCPs) = %d, want 2", len(got.MCPs))
	}
	if got.EnvVars["K"] != "V" {
		t.Errorf("EnvVars[K] = %q, want %q", got.EnvVars["K"], "V")
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	m := tempManager(t)

	original := Profile{
		Name:        "frontend",
		Description: "Frontend loadout",
		MCPs:        []string{"browser-mcp", "github"},
		Skills:      []string{"react-patterns"},
		EnvVars:     map[string]string{"NODE_ENV": "dev"},
		Isolation:   IsolationLock,
	}

	if err := m.Save(original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := m.Export("frontend")
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	// Delete the original so Import can succeed
	if err := m.Delete("frontend"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	imported, err := m.Import(data)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	if imported.Name != original.Name {
		t.Errorf("Name = %q, want %q", imported.Name, original.Name)
	}
	if len(imported.MCPs) != 2 {
		t.Errorf("len(MCPs) = %d, want 2", len(imported.MCPs))
	}
	if imported.Isolation != IsolationLock {
		t.Errorf("Isolation = %q, want %q", imported.Isolation, IsolationLock)
	}
}

func TestImportDuplicate(t *testing.T) {
	m := tempManager(t)

	if err := m.Save(Profile{Name: "existing"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data := []byte("name: existing\nmcps: [a]\n")
	_, err := m.Import(data)
	if !hysterr.IsAlreadyExists(err) {
		t.Errorf("Import duplicate: got %v, want AlreadyExistsError", err)
	}
}

func TestImportNoName(t *testing.T) {
	m := tempManager(t)
	data := []byte("mcps: [a]\n")
	_, err := m.Import(data)
	if err == nil {
		t.Error("Import no name: expected error, got nil")
	}
}

func TestImportInvalidYAML(t *testing.T) {
	m := tempManager(t)
	data := []byte(":::invalid:::")
	_, err := m.Import(data)
	if err == nil {
		t.Error("Import invalid YAML: expected error, got nil")
	}
}

func TestImportAsRoundTrip(t *testing.T) {
	m := tempManager(t)

	original := Profile{
		Name:        "frontend",
		Description: "Frontend loadout",
		MCPs:        []string{"browser-mcp", "github"},
		Skills:      []string{"react-patterns"},
		EnvVars:     map[string]string{"NODE_ENV": "dev"},
		Isolation:   IsolationLock,
	}

	if err := m.Save(original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := m.Export("frontend")
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	// ImportAs with a new name should succeed even though "frontend" exists.
	imported, err := m.ImportAs(data, "frontend-copy")
	if err != nil {
		t.Fatalf("ImportAs: %v", err)
	}

	if imported.Name != "frontend-copy" {
		t.Errorf("Name = %q, want %q", imported.Name, "frontend-copy")
	}
	if len(imported.MCPs) != 2 {
		t.Errorf("len(MCPs) = %d, want 2", len(imported.MCPs))
	}
	if imported.Isolation != IsolationLock {
		t.Errorf("Isolation = %q, want %q", imported.Isolation, IsolationLock)
	}

	// Verify it was saved.
	got, err := m.Get("frontend-copy")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Description != original.Description {
		t.Errorf("Description = %q, want %q", got.Description, original.Description)
	}
}

func TestImportAsDuplicate(t *testing.T) {
	m := tempManager(t)

	if err := m.Save(Profile{Name: "existing"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data := []byte("name: other\nmcps: [a]\n")
	_, err := m.ImportAs(data, "existing")
	if !hysterr.IsAlreadyExists(err) {
		t.Errorf("ImportAs duplicate: got %v, want AlreadyExistsError", err)
	}
}

func TestImportAsVanilla(t *testing.T) {
	m := tempManager(t)
	data := []byte("name: test\nmcps: [a]\n")
	_, err := m.ImportAs(data, VanillaName)
	if err == nil {
		t.Error("ImportAs vanilla: expected error, got nil")
	}
}

func TestImportAsEmptyName(t *testing.T) {
	m := tempManager(t)
	data := []byte("name: test\nmcps: [a]\n")
	_, err := m.ImportAs(data, "")
	if err == nil {
		t.Error("ImportAs empty name: expected error, got nil")
	}
}

func TestProfileIsEmpty(t *testing.T) {
	tests := []struct {
		name  string
		p     Profile
		empty bool
	}{
		{"empty", Profile{Name: "x"}, true},
		{"with mcps", Profile{Name: "x", MCPs: []string{"a"}}, false},
		{"with skills", Profile{Name: "x", Skills: []string{"a"}}, false},
		{"with hooks", Profile{Name: "x", Hooks: []string{"a"}}, false},
		{"with perms", Profile{Name: "x", Permissions: []string{"a"}}, false},
		{"with env", Profile{Name: "x", EnvVars: map[string]string{"K": "V"}}, false},
		{"with claudemd", Profile{Name: "x", ClaudeMD: "t"}, false},
		{"isolation only", Profile{Name: "x", Isolation: IsolationWorktree}, true}, // isolation doesn't make it non-empty
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.IsEmpty(); got != tt.empty {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.empty)
			}
		})
	}
}

func TestListSkipsNonYAML(t *testing.T) {
	m := tempManager(t)

	// Create profiles dir and add a non-yaml file
	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(m.dir, "readme.txt"), []byte("not a profile"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := m.Save(Profile{Name: "real"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	profiles, err := m.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	// Should be: real + vanilla
	if len(profiles) != 2 {
		t.Errorf("List: got %d profiles, want 2", len(profiles))
	}
}

func TestListSkipsVanillaFile(t *testing.T) {
	m := tempManager(t)

	// Create a vanilla.yaml file that should be ignored (vanilla is built-in)
	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(m.dir, "vanilla.yaml"), []byte("name: vanilla\nmcps: [fake]\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	profiles, err := m.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	// Only the built-in vanilla should appear, not the file version
	if len(profiles) != 1 {
		t.Fatalf("List: got %d profiles, want 1", len(profiles))
	}
	if !profiles[0].IsVanilla() || !profiles[0].IsEmpty() {
		t.Error("vanilla from List should be the built-in empty one")
	}
}
