package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanSkills(t *testing.T) {
	tmp := t.TempDir()
	projDir := filepath.Join(tmp, "project")

	// Create two skill directories
	for _, name := range []string{"review", "commit"} {
		dir := filepath.Join(projDir, ".claude", "skills", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# "+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Create a directory without SKILL.md (should be skipped)
	emptyDir := filepath.Join(projDir, ".claude", "skills", "empty")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}

	skills, err := ScanSkills(projDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(skills) != 2 {
		t.Fatalf("skills = %d, want 2", len(skills))
	}

	// Verify sorted order
	if skills[0].Name != "commit" {
		t.Errorf("skills[0].Name = %q, want commit", skills[0].Name)
	}
	if skills[1].Name != "review" {
		t.Errorf("skills[1].Name = %q, want review", skills[1].Name)
	}
}

func TestScanSkills_NoDirectory(t *testing.T) {
	tmp := t.TempDir()
	skills, err := ScanSkills(filepath.Join(tmp, "nonexistent"))
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 0 {
		t.Errorf("skills = %d, want 0 for missing dir", len(skills))
	}
}
