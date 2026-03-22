package project

import (
	"os"
	"path/filepath"
)

// projectMarkers are files/dirs that indicate a directory is a project root.
var projectMarkers = []string{
	".git",
	"package.json",
	"pyproject.toml",
	"go.mod",
	"Cargo.toml",
}

// IsProjectDir reports whether the given directory contains any project markers.
func IsProjectDir(dir string) bool {
	for _, marker := range projectMarkers {
		path := filepath.Join(dir, marker)
		if _, err := os.Lstat(path); err == nil {
			return true
		}
	}
	return false
}

// ProjectNameFromPath returns a project name derived from the directory basename.
func ProjectNameFromPath(dir string) string {
	return filepath.Base(dir)
}
