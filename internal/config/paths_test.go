package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDirRespectsEnvVar(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HYSTAK_CONFIG_DIR", tmp)

	got := ConfigDir()
	if got != tmp {
		t.Errorf("ConfigDir() = %q, want %q", got, tmp)
	}
}

func TestConfigDirDefaultsToHomeHystak(t *testing.T) {
	t.Setenv("HYSTAK_CONFIG_DIR", "")

	got := ConfigDir()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".hystak")
	if got != want {
		t.Errorf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestLegacyConfigDirRespectsXDG(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	got := LegacyConfigDir()
	want := filepath.Join(tmp, "hystak")
	if got != want {
		t.Errorf("LegacyConfigDir() = %q, want %q", got, want)
	}
}

func TestRegistryPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HYSTAK_CONFIG_DIR", tmp)

	got := RegistryPath()
	want := filepath.Join(tmp, "registry.yaml")
	if got != want {
		t.Errorf("RegistryPath() = %q, want %q", got, want)
	}
}

func TestProjectsPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HYSTAK_CONFIG_DIR", tmp)

	got := ProjectsPath()
	want := filepath.Join(tmp, "projects.yaml")
	if got != want {
		t.Errorf("ProjectsPath() = %q, want %q", got, want)
	}
}

func TestEnsureConfigDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HYSTAK_CONFIG_DIR", filepath.Join(tmp, "hystak"))

	if err := EnsureConfigDir(); err != nil {
		t.Fatalf("EnsureConfigDir() error: %v", err)
	}

	dir := filepath.Join(tmp, "hystak")
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("config dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("config dir is not a directory")
	}

	// Check YAML files.
	for _, name := range []string{"registry.yaml", "projects.yaml"} {
		p := filepath.Join(dir, name)
		info, err := os.Stat(p)
		if err != nil {
			t.Errorf("file %s not created: %v", name, err)
		}
		if info.Size() != 0 {
			t.Errorf("file %s should be empty, got size %d", name, info.Size())
		}
	}

	// Check subdirectories.
	for _, sub := range []string{"profiles", "skills", "templates", "backups"} {
		p := filepath.Join(dir, sub)
		info, err := os.Stat(p)
		if err != nil {
			t.Errorf("subdir %s not created: %v", sub, err)
		}
		if !info.IsDir() {
			t.Errorf("subdir %s is not a directory", sub)
		}
	}
}

func TestEnsureConfigDirIdempotent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HYSTAK_CONFIG_DIR", filepath.Join(tmp, "hystak"))

	if err := EnsureConfigDir(); err != nil {
		t.Fatalf("first EnsureConfigDir() error: %v", err)
	}

	// Write content to registry.yaml to verify it's not overwritten.
	regPath := filepath.Join(tmp, "hystak", "registry.yaml")
	if err := os.WriteFile(regPath, []byte("servers: {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureConfigDir(); err != nil {
		t.Fatalf("second EnsureConfigDir() error: %v", err)
	}

	data, err := os.ReadFile(regPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "servers: {}" {
		t.Errorf("registry.yaml was overwritten, got %q", string(data))
	}
}

func TestMigrateFromLegacyDir(t *testing.T) {
	tmp := t.TempDir()
	newDir := filepath.Join(tmp, "new")
	oldDir := filepath.Join(tmp, "old")
	t.Setenv("HYSTAK_CONFIG_DIR", newDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "old-parent"))

	// Set up old dir at the legacy location.
	// LegacyConfigDir will use XDG_CONFIG_HOME, so create it there.
	legacyDir := LegacyConfigDir()
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = oldDir // not used directly — legacy path computed from XDG

	regContent := []byte("servers:\n  test:\n    transport: stdio\n")
	projContent := []byte("projects:\n  demo:\n    path: /tmp/demo\n")
	if err := os.WriteFile(filepath.Join(legacyDir, "registry.yaml"), regContent, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "projects.yaml"), projContent, 0o644); err != nil {
		t.Fatal(err)
	}

	warning, err := Migrate()
	if err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}
	if warning == "" {
		t.Error("Migrate() returned empty warning, expected migration notice")
	}

	// Verify files were copied.
	got, err := os.ReadFile(filepath.Join(newDir, "registry.yaml"))
	if err != nil {
		t.Fatalf("registry.yaml not migrated: %v", err)
	}
	if string(got) != string(regContent) {
		t.Errorf("registry.yaml content mismatch: got %q, want %q", got, regContent)
	}

	got, err = os.ReadFile(filepath.Join(newDir, "projects.yaml"))
	if err != nil {
		t.Fatalf("projects.yaml not migrated: %v", err)
	}
	if string(got) != string(projContent) {
		t.Errorf("projects.yaml content mismatch: got %q, want %q", got, projContent)
	}

	// Verify subdirectories were created.
	for _, sub := range []string{"profiles", "skills", "templates", "backups"} {
		if _, err := os.Stat(filepath.Join(newDir, sub)); err != nil {
			t.Errorf("subdir %s not created during migration: %v", sub, err)
		}
	}

	// Verify old dir is intact.
	if _, err := os.Stat(filepath.Join(legacyDir, "registry.yaml")); err != nil {
		t.Error("old registry.yaml was removed during migration")
	}
}

func TestMigrateNoOldDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HYSTAK_CONFIG_DIR", filepath.Join(tmp, "new"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "nonexistent"))

	warning, err := Migrate()
	if err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}
	if warning != "" {
		t.Errorf("Migrate() returned warning %q, expected empty", warning)
	}
}

func TestMigrateAlreadyMigrated(t *testing.T) {
	tmp := t.TempDir()
	newDir := filepath.Join(tmp, "new")
	t.Setenv("HYSTAK_CONFIG_DIR", newDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "old-parent"))

	// Set up legacy dir.
	legacyDir := LegacyConfigDir()
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "registry.yaml"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Set up new dir with existing registry (already migrated).
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newDir, "registry.yaml"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	warning, err := Migrate()
	if err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}
	if warning != "" {
		t.Errorf("Migrate() returned warning %q, expected empty for already-migrated", warning)
	}

	// Verify new dir was NOT overwritten.
	got, _ := os.ReadFile(filepath.Join(newDir, "registry.yaml"))
	if string(got) != "new" {
		t.Errorf("registry.yaml was overwritten: got %q, want %q", got, "new")
	}
}
