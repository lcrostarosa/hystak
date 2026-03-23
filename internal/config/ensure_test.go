package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureConfigDir_CreatesAllSubdirs(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HYSTAK_CONFIG_DIR", tmp)

	dir, err := EnsureConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir != tmp {
		t.Errorf("EnsureConfigDir() = %q, want %q", dir, tmp)
	}

	for _, sub := range Subdirs() {
		subPath := filepath.Join(tmp, sub)
		info, err := os.Stat(subPath)
		if err != nil {
			t.Errorf("subdir %q: %v", sub, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("subdir %q is not a directory", sub)
		}
	}
}

func TestEnsureConfigDir_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HYSTAK_CONFIG_DIR", tmp)

	_, err := EnsureConfigDir()
	if err != nil {
		t.Fatal(err)
	}

	// Call again — should not error
	_, err = EnsureConfigDir()
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
}

func TestIsFirstRun_NoDir(t *testing.T) {
	tmp := t.TempDir()
	// Point to a path that doesn't exist
	t.Setenv("HYSTAK_CONFIG_DIR", filepath.Join(tmp, "nonexistent"))

	first, err := IsFirstRun()
	if err != nil {
		t.Fatal(err)
	}
	if !first {
		t.Error("IsFirstRun() = false, want true")
	}
}

func TestIsFirstRun_DirExists(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HYSTAK_CONFIG_DIR", tmp)

	first, err := IsFirstRun()
	if err != nil {
		t.Fatal(err)
	}
	if first {
		t.Error("IsFirstRun() = true, want false")
	}
}
