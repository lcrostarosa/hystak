package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClaudeMDDeployer_SyncClaudeMD_CreatesSymlink(t *testing.T) {
	projectDir := t.TempDir()

	// Create template source.
	sourceDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "template.md")
	if err := os.WriteFile(sourceFile, []byte("# My Template\nContent here."), 0o644); err != nil {
		t.Fatal(err)
	}

	deployer := &ClaudeMDDeployer{}
	if err := deployer.SyncClaudeMD(projectDir, sourceFile, nil); err != nil {
		t.Fatalf("SyncClaudeMD: %v", err)
	}

	target := filepath.Join(projectDir, "CLAUDE.md")

	// Verify it's a symlink.
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatalf("CLAUDE.md not created: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("CLAUDE.md should be a symlink")
	}

	// Verify symlink target.
	linkTarget, err := os.Readlink(target)
	if err != nil {
		t.Fatalf("reading symlink: %v", err)
	}
	if linkTarget != sourceFile {
		t.Errorf("symlink target = %q, want %q", linkTarget, sourceFile)
	}

	// Verify content is readable.
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading through symlink: %v", err)
	}
	if string(content) != "# My Template\nContent here." {
		t.Errorf("unexpected content: %q", string(content))
	}
}

func TestClaudeMDDeployer_SkipsUserContent(t *testing.T) {
	projectDir := t.TempDir()

	// Create user CLAUDE.md (regular file, no sentinel).
	target := filepath.Join(projectDir, "CLAUDE.md")
	userContent := "# User's own CLAUDE.md\nDo not overwrite."
	if err := os.WriteFile(target, []byte(userContent), 0o644); err != nil {
		t.Fatal(err)
	}

	sourceDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "template.md")
	if err := os.WriteFile(sourceFile, []byte("# Template"), 0o644); err != nil {
		t.Fatal(err)
	}

	deployer := &ClaudeMDDeployer{}
	if err := deployer.SyncClaudeMD(projectDir, sourceFile, nil); err != nil {
		t.Fatalf("SyncClaudeMD: %v", err)
	}

	// Verify user content was preserved.
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != userContent {
		t.Errorf("user CLAUDE.md was overwritten, got: %q", string(content))
	}
}

func TestClaudeMDDeployer_UpdatesExistingSymlink(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	sourceA := filepath.Join(sourceDir, "template-a.md")
	sourceB := filepath.Join(sourceDir, "template-b.md")
	if err := os.WriteFile(sourceA, []byte("Template A"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourceB, []byte("Template B"), 0o644); err != nil {
		t.Fatal(err)
	}

	deployer := &ClaudeMDDeployer{}

	// Deploy with source A.
	if err := deployer.SyncClaudeMD(projectDir, sourceA, nil); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(projectDir, "CLAUDE.md")
	linkTarget, _ := os.Readlink(target)
	if linkTarget != sourceA {
		t.Fatalf("initial symlink = %q, want %q", linkTarget, sourceA)
	}

	// Deploy with source B.
	if err := deployer.SyncClaudeMD(projectDir, sourceB, nil); err != nil {
		t.Fatal(err)
	}

	linkTarget, _ = os.Readlink(target)
	if linkTarget != sourceB {
		t.Errorf("updated symlink = %q, want %q", linkTarget, sourceB)
	}
}

func TestClaudeMDDeployer_SkipsCorrectSymlink(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	sourceFile := filepath.Join(sourceDir, "template.md")
	if err := os.WriteFile(sourceFile, []byte("stable"), 0o644); err != nil {
		t.Fatal(err)
	}

	deployer := &ClaudeMDDeployer{}

	// Deploy once.
	if err := deployer.SyncClaudeMD(projectDir, sourceFile, nil); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(projectDir, "CLAUDE.md")
	info1, _ := os.Lstat(target)

	// Deploy again — should not recreate.
	if err := deployer.SyncClaudeMD(projectDir, sourceFile, nil); err != nil {
		t.Fatal(err)
	}

	info2, _ := os.Lstat(target)
	if !info1.ModTime().Equal(info2.ModTime()) {
		t.Error("symlink was unnecessarily recreated")
	}
}

func TestClaudeMDDeployer_MigratesLegacySentinel(t *testing.T) {
	projectDir := t.TempDir()

	// Create CLAUDE.md with legacy sentinel.
	target := filepath.Join(projectDir, "CLAUDE.md")
	oldContent := legacyHystakSentinel + "\n# Old Template"
	if err := os.WriteFile(target, []byte(oldContent), 0o644); err != nil {
		t.Fatal(err)
	}

	sourceDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "template.md")
	if err := os.WriteFile(sourceFile, []byte("# New Template"), 0o644); err != nil {
		t.Fatal(err)
	}

	deployer := &ClaudeMDDeployer{}
	if err := deployer.SyncClaudeMD(projectDir, sourceFile, nil); err != nil {
		t.Fatal(err)
	}

	// Should now be a symlink.
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("legacy sentinel file should be migrated to symlink")
	}

	linkTarget, _ := os.Readlink(target)
	if linkTarget != sourceFile {
		t.Errorf("symlink target = %q, want %q", linkTarget, sourceFile)
	}
}

func TestClaudeMDDeployer_EmptySource(t *testing.T) {
	projectDir := t.TempDir()

	deployer := &ClaudeMDDeployer{}
	if err := deployer.SyncClaudeMD(projectDir, "", nil); err != nil {
		t.Fatalf("SyncClaudeMD with empty source: %v", err)
	}

	target := filepath.Join(projectDir, "CLAUDE.md")
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Error("CLAUDE.md should not be created with empty source")
	}
}

func TestClaudeMDDeployer_SourceMissing(t *testing.T) {
	projectDir := t.TempDir()

	deployer := &ClaudeMDDeployer{}
	err := deployer.SyncClaudeMD(projectDir, "/nonexistent/template.md", nil)
	if err == nil {
		t.Fatal("expected error for missing source")
	}
}

func TestPreflightClaudeMD_NoFile(t *testing.T) {
	projectDir := t.TempDir()
	deployer := &ClaudeMDDeployer{}
	conflict := deployer.PreflightClaudeMD(projectDir)
	if conflict != nil {
		t.Errorf("expected no conflict when CLAUDE.md does not exist, got: %+v", conflict)
	}
}

func TestPreflightClaudeMD_UserFile(t *testing.T) {
	projectDir := t.TempDir()

	target := filepath.Join(projectDir, "CLAUDE.md")
	if err := os.WriteFile(target, []byte("# My Project"), 0o644); err != nil {
		t.Fatal(err)
	}

	deployer := &ClaudeMDDeployer{}
	conflict := deployer.PreflightClaudeMD(projectDir)

	if conflict == nil {
		t.Fatal("expected conflict for unmanaged CLAUDE.md")
	}
	if conflict.ResourceType != "claude_md" {
		t.Errorf("conflict.ResourceType = %q, want %q", conflict.ResourceType, "claude_md")
	}
}

func TestPreflightClaudeMD_Symlink(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	sourceFile := filepath.Join(sourceDir, "template.md")
	if err := os.WriteFile(sourceFile, []byte("template"), 0o644); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(projectDir, "CLAUDE.md")
	if err := os.Symlink(sourceFile, target); err != nil {
		t.Fatal(err)
	}

	deployer := &ClaudeMDDeployer{}
	conflict := deployer.PreflightClaudeMD(projectDir)

	if conflict != nil {
		t.Errorf("expected no conflict for symlinked CLAUDE.md, got: %+v", conflict)
	}
}

func TestPreflightClaudeMD_LegacySentinel(t *testing.T) {
	projectDir := t.TempDir()

	target := filepath.Join(projectDir, "CLAUDE.md")
	if err := os.WriteFile(target, []byte(legacyHystakSentinel+"\n# Template"), 0o644); err != nil {
		t.Fatal(err)
	}

	deployer := &ClaudeMDDeployer{}
	conflict := deployer.PreflightClaudeMD(projectDir)

	if conflict != nil {
		t.Errorf("expected no conflict for legacy sentinel CLAUDE.md, got: %+v", conflict)
	}
}

func TestIsClaudeMDManaged(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	sourceFile := filepath.Join(sourceDir, "template.md")
	if err := os.WriteFile(sourceFile, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	deployer := &ClaudeMDDeployer{}

	// Deploy as symlink.
	if err := deployer.SyncClaudeMD(projectDir, sourceFile, nil); err != nil {
		t.Fatal(err)
	}

	if !deployer.IsClaudeMDManaged(projectDir) {
		t.Error("deployed CLAUDE.md should be managed")
	}

	// Check undeployed directory.
	emptyDir := t.TempDir()
	if deployer.IsClaudeMDManaged(emptyDir) {
		t.Error("nonexistent CLAUDE.md should not be managed")
	}
}

// --- Composition tests ---

func TestClaudeMDDeployer_ComposesTemplateAndPrompts(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	// Create template and prompt files.
	templateFile := filepath.Join(sourceDir, "template.md")
	_ = os.WriteFile(templateFile, []byte("# Template Content"), 0o644)

	prompt1 := filepath.Join(sourceDir, "prompt1.md")
	_ = os.WriteFile(prompt1, []byte("## Prompt One"), 0o644)

	prompt2 := filepath.Join(sourceDir, "prompt2.md")
	_ = os.WriteFile(prompt2, []byte("## Prompt Two"), 0o644)

	deployer := &ClaudeMDDeployer{}
	if err := deployer.SyncClaudeMD(projectDir, templateFile, []string{prompt1, prompt2}); err != nil {
		t.Fatalf("SyncClaudeMD: %v", err)
	}

	target := filepath.Join(projectDir, "CLAUDE.md")
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading CLAUDE.md: %v", err)
	}

	s := string(content)

	// Should start with sentinel.
	if s[:len(legacyHystakSentinel)] != legacyHystakSentinel {
		t.Error("composed file should start with sentinel")
	}

	// Should contain template and both prompts.
	if !contains(s, "# Template Content") {
		t.Error("composed file should contain template content")
	}
	if !contains(s, "## Prompt One") {
		t.Error("composed file should contain prompt 1")
	}
	if !contains(s, "## Prompt Two") {
		t.Error("composed file should contain prompt 2")
	}

	// Should NOT be a symlink.
	info, _ := os.Lstat(target)
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("composed CLAUDE.md should be a regular file, not a symlink")
	}

	// Should be managed.
	if !deployer.IsClaudeMDManaged(projectDir) {
		t.Error("composed CLAUDE.md should be managed")
	}
}

func TestClaudeMDDeployer_PromptsOnly(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	prompt := filepath.Join(sourceDir, "guardrails.md")
	_ = os.WriteFile(prompt, []byte("# Safety First"), 0o644)

	deployer := &ClaudeMDDeployer{}
	if err := deployer.SyncClaudeMD(projectDir, "", []string{prompt}); err != nil {
		t.Fatalf("SyncClaudeMD: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	s := string(content)

	if !contains(s, legacyHystakSentinel) {
		t.Error("should start with sentinel")
	}
	if !contains(s, "# Safety First") {
		t.Error("should contain prompt content")
	}
}

func TestClaudeMDDeployer_SymlinkToComposed(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	templateFile := filepath.Join(sourceDir, "template.md")
	_ = os.WriteFile(templateFile, []byte("# Template"), 0o644)

	prompt := filepath.Join(sourceDir, "prompt.md")
	_ = os.WriteFile(prompt, []byte("# Prompt"), 0o644)

	deployer := &ClaudeMDDeployer{}

	// First deploy as symlink.
	if err := deployer.SyncClaudeMD(projectDir, templateFile, nil); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(projectDir, "CLAUDE.md")
	info, _ := os.Lstat(target)
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("should be a symlink initially")
	}

	// Now deploy with prompts — should transition to composed file.
	if err := deployer.SyncClaudeMD(projectDir, templateFile, []string{prompt}); err != nil {
		t.Fatal(err)
	}
	info, _ = os.Lstat(target)
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("should be a regular file after adding prompts")
	}

	content, _ := os.ReadFile(target)
	if !contains(string(content), "# Prompt") {
		t.Error("composed file should contain prompt content")
	}
}

func TestClaudeMDDeployer_ComposedToSymlink(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	templateFile := filepath.Join(sourceDir, "template.md")
	_ = os.WriteFile(templateFile, []byte("# Template"), 0o644)

	prompt := filepath.Join(sourceDir, "prompt.md")
	_ = os.WriteFile(prompt, []byte("# Prompt"), 0o644)

	deployer := &ClaudeMDDeployer{}

	// First deploy as composed.
	if err := deployer.SyncClaudeMD(projectDir, templateFile, []string{prompt}); err != nil {
		t.Fatal(err)
	}

	// Now deploy without prompts — should transition back to symlink.
	if err := deployer.SyncClaudeMD(projectDir, templateFile, nil); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(projectDir, "CLAUDE.md")
	info, _ := os.Lstat(target)
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("should revert to symlink when prompts are removed")
	}

	linkTarget, _ := os.Readlink(target)
	if linkTarget != templateFile {
		t.Errorf("symlink target = %q, want %q", linkTarget, templateFile)
	}
}

func TestClaudeMDDeployer_SkipsUserContentWithPrompts(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	// Create user-owned CLAUDE.md.
	target := filepath.Join(projectDir, "CLAUDE.md")
	userContent := "# User's CLAUDE.md"
	_ = os.WriteFile(target, []byte(userContent), 0o644)

	prompt := filepath.Join(sourceDir, "prompt.md")
	_ = os.WriteFile(prompt, []byte("# Prompt"), 0o644)

	deployer := &ClaudeMDDeployer{}
	if err := deployer.SyncClaudeMD(projectDir, "", []string{prompt}); err != nil {
		t.Fatal(err)
	}

	// User content should be preserved.
	content, _ := os.ReadFile(target)
	if string(content) != userContent {
		t.Errorf("user content was overwritten, got: %q", string(content))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
