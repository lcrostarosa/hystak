package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWrite_CreatesFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.yaml")
	data := []byte("hello: world\n")

	if err := AtomicWrite(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Errorf("content = %q, want %q", string(got), string(data))
	}
}

func TestAtomicWrite_OverwritesExisting(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.yaml")

	if err := AtomicWrite(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := AtomicWrite(path, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Errorf("content = %q, want %q", string(got), "new")
	}
}

func TestAtomicWrite_NoTempFileOnSuccess(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.yaml")

	if err := AtomicWrite(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("expected 1 file, got %d: %v", len(entries), names)
	}
}

func TestAtomicWrite_AppliesPermissions(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "secret.yaml")

	if err := AtomicWrite(path, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	got := info.Mode().Perm()
	if got != 0o600 {
		t.Errorf("permissions = %o, want %o", got, 0o600)
	}
}

func TestAtomicWrite_InvalidDir(t *testing.T) {
	err := AtomicWrite("/nonexistent/dir/file.yaml", []byte("data"), 0o644)
	if err == nil {
		t.Error("expected error for nonexistent directory, got nil")
	}
}
