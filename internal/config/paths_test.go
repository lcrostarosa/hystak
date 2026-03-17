package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDirRespectsXDG(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	got := ConfigDir()
	want := filepath.Join(tmp, "hystak")
	if got != want {
		t.Errorf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestConfigDirDefaultsToHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")

	got := ConfigDir()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "hystak")
	if got != want {
		t.Errorf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestRegistryPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	got := RegistryPath()
	want := filepath.Join(tmp, "hystak", "registry.yaml")
	if got != want {
		t.Errorf("RegistryPath() = %q, want %q", got, want)
	}
}

func TestProjectsPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	got := ProjectsPath()
	want := filepath.Join(tmp, "hystak", "projects.yaml")
	if got != want {
		t.Errorf("ProjectsPath() = %q, want %q", got, want)
	}
}

func TestEnsureConfigDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

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
}

func TestEnsureConfigDirIdempotent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

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
