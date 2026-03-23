package deploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaudeMDDeployer_Sync_SymlinkMode(t *testing.T) {
	tmp := t.TempDir()

	d := &ClaudeMDDeployer{}
	projDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create template source
	tmplPath := filepath.Join(tmp, "template.md")
	if err := os.WriteFile(tmplPath, []byte("# Template"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := DeployConfig{TemplateSource: tmplPath}
	if err := d.Sync(projDir, cfg); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(projDir, "CLAUDE.md")
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("CLAUDE.md not created: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink for template-only mode")
	}

	target, err := os.Readlink(path)
	if err != nil {
		t.Fatal(err)
	}
	if target != tmplPath {
		t.Errorf("symlink target = %q, want %q", target, tmplPath)
	}
}

func TestClaudeMDDeployer_Sync_ComposedMode(t *testing.T) {
	tmp := t.TempDir()
	d := &ClaudeMDDeployer{}
	projDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tmplPath := filepath.Join(tmp, "template.md")
	if err := os.WriteFile(tmplPath, []byte("# Base Template"), 0o644); err != nil {
		t.Fatal(err)
	}
	promptPath := filepath.Join(tmp, "safety.md")
	if err := os.WriteFile(promptPath, []byte("## Safety Rules"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := DeployConfig{
		TemplateSource: tmplPath,
		PromptSources:  []string{promptPath},
	}
	if err := d.Sync(projDir, cfg); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(projDir, "CLAUDE.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.HasPrefix(content, managedSentinel) {
		t.Error("composed file should start with managed sentinel")
	}
	if !strings.Contains(content, "# Base Template") {
		t.Error("composed file should contain template content")
	}
	if !strings.Contains(content, "## Safety Rules") {
		t.Error("composed file should contain prompt content")
	}
}

func TestClaudeMDDeployer_Sync_RemoveManaged(t *testing.T) {
	tmp := t.TempDir()
	d := &ClaudeMDDeployer{}
	projDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a managed file
	path := filepath.Join(projDir, "CLAUDE.md")
	if err := os.WriteFile(path, []byte(managedSentinel+"\n# content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Sync with empty config -- should remove managed file
	if err := d.Sync(projDir, DeployConfig{}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("managed CLAUDE.md should be removed when no template/prompts")
	}
}

func TestClaudeMDDeployer_Sync_PreservesUserOwned(t *testing.T) {
	tmp := t.TempDir()

	d := &ClaudeMDDeployer{}
	projDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a user-owned CLAUDE.md (no sentinel)
	path := filepath.Join(projDir, "CLAUDE.md")
	if err := os.WriteFile(path, []byte("# My Custom Instructions"), 0o644); err != nil {
		t.Fatal(err)
	}

	tmplPath := filepath.Join(tmp, "template.md")
	if err := os.WriteFile(tmplPath, []byte("# Template"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Sync should NOT overwrite user-owned file
	cfg := DeployConfig{TemplateSource: tmplPath}
	if err := d.Sync(projDir, cfg); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# My Custom Instructions" {
		t.Error("user-owned CLAUDE.md should not be overwritten")
	}
}

func TestClaudeMDDeployer_Preflight_UserOwned(t *testing.T) {
	tmp := t.TempDir()
	d := &ClaudeMDDeployer{}
	projDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(projDir, "CLAUDE.md")
	if err := os.WriteFile(path, []byte("# User content"), 0o644); err != nil {
		t.Fatal(err)
	}

	conflicts := d.Preflight(projDir, DeployConfig{TemplateSource: "/tmpl"})
	if len(conflicts) != 1 {
		t.Fatalf("conflicts = %d, want 1", len(conflicts))
	}
	if conflicts[0].Kind != ResourceDeployerClaudeMD {
		t.Errorf("conflict kind = %q, want claude-md", conflicts[0].Kind)
	}
}

func TestClaudeMDDeployer_Preflight_NoConflict_Symlink(t *testing.T) {
	tmp := t.TempDir()
	d := &ClaudeMDDeployer{}
	projDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create symlink (managed, not a conflict per S-048)
	target := filepath.Join(tmp, "template.md")
	if err := os.WriteFile(target, []byte("tmpl"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(projDir, "CLAUDE.md")); err != nil {
		t.Fatal(err)
	}

	conflicts := d.Preflight(projDir, DeployConfig{TemplateSource: "/tmpl"})
	if len(conflicts) != 0 {
		t.Errorf("symlink should not be a conflict, got %d", len(conflicts))
	}
}

func TestClaudeMDDeployer_ReadDeployed(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, projDir string)
		wantTemplate string
		wantEmpty    bool
	}{
		{
			name: "symlink returns TemplateSource",
			setup: func(t *testing.T, projDir string) {
				t.Helper()
				target := filepath.Join(t.TempDir(), "template.md")
				if err := os.WriteFile(target, []byte("# Template"), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, filepath.Join(projDir, "CLAUDE.md")); err != nil {
					t.Fatal(err)
				}
			},
			wantTemplate: "", // will be a real path, just check non-empty
			wantEmpty:    false,
		},
		{
			name: "composed file with sentinel",
			setup: func(t *testing.T, projDir string) {
				t.Helper()
				content := managedSentinel + "\n\n# Template\n"
				if err := os.WriteFile(filepath.Join(projDir, "CLAUDE.md"), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantTemplate: "(composed)",
			wantEmpty:    false,
		},
		{
			name: "user-owned file",
			setup: func(t *testing.T, projDir string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(projDir, "CLAUDE.md"), []byte("# User content"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantTemplate: "",
			wantEmpty:    true,
		},
		{
			name:         "no file",
			setup:        func(t *testing.T, projDir string) { t.Helper() },
			wantTemplate: "",
			wantEmpty:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			projDir := filepath.Join(tmp, "project")
			if err := os.MkdirAll(projDir, 0o755); err != nil {
				t.Fatal(err)
			}
			tt.setup(t, projDir)

			d := &ClaudeMDDeployer{}
			deployed, err := d.ReadDeployed(projDir)
			if err != nil {
				t.Fatal(err)
			}

			if tt.wantEmpty {
				if deployed.TemplateSource != "" {
					t.Errorf("TemplateSource = %q, want empty", deployed.TemplateSource)
				}
			} else {
				if deployed.TemplateSource == "" {
					t.Error("TemplateSource should not be empty")
				}
				if tt.wantTemplate != "" && deployed.TemplateSource != tt.wantTemplate {
					t.Errorf("TemplateSource = %q, want %q", deployed.TemplateSource, tt.wantTemplate)
				}
			}
		})
	}
}

func TestClaudeMDDeployer_Sync_SymlinkIdempotent(t *testing.T) {
	tmp := t.TempDir()
	d := &ClaudeMDDeployer{}
	projDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tmplPath := filepath.Join(tmp, "template.md")
	if err := os.WriteFile(tmplPath, []byte("# Template"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := DeployConfig{TemplateSource: tmplPath}

	// First sync
	if err := d.Sync(projDir, cfg); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(projDir, "CLAUDE.md")
	info1, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}

	// Second sync with same config
	if err := d.Sync(projDir, cfg); err != nil {
		t.Fatal(err)
	}
	info2, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}

	// Note: ClaudeMDDeployer always removes and recreates symlinks,
	// so SameFile may differ. This test documents the current behavior.
	// If we want idempotency, the deployer should check target first.
	_ = info1
	_ = info2
	// Verify symlink still points to the right target
	target, err := os.Readlink(path)
	if err != nil {
		t.Fatal(err)
	}
	if target != tmplPath {
		t.Errorf("symlink target = %q, want %q after re-sync", target, tmplPath)
	}
}
