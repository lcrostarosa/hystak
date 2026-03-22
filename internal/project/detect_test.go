package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsProjectDir_WithMarkers(t *testing.T) {
	markers := []string{".git", "package.json", "pyproject.toml", "go.mod", "Cargo.toml"}
	for _, marker := range markers {
		t.Run(marker, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, marker)
			// Create marker: directory for .git, file for others
			if marker == ".git" {
				if err := os.Mkdir(path, 0o755); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if !IsProjectDir(dir) {
				t.Errorf("IsProjectDir() = false with %s marker", marker)
			}
		})
	}
}

func TestIsProjectDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	if IsProjectDir(dir) {
		t.Error("IsProjectDir() = true for empty dir")
	}
}

func TestProjectNameFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/home/user/projects/myapp", "myapp"},
		{"/test", "test"},
		{"/a/b/c", "c"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := ProjectNameFromPath(tt.path); got != tt.want {
				t.Errorf("ProjectNameFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
