package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	regYAML := `servers:
  github:
    transport: stdio
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN}"
  remote-api:
    transport: http
    url: "https://mcp.example.com/mcp"
    headers:
      Authorization: "Bearer ${API_TOKEN}"
tags:
  core: [github]
`
	projDir := filepath.Join(dir, "myproject")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	projYAML := fmt.Sprintf(`projects:
  myproject:
    path: %s
    clients: [claude-code]
    mcps:
      - github
`, projDir)

	if err := os.WriteFile(filepath.Join(dir, "registry.yaml"), []byte(regYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "projects.yaml"), []byte(projYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	return dir
}

func runCommand(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := newRootCmd("test-version", "abc123", "2026-01-01")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(append([]string{"--config-dir", dir}, args...))
	err := cmd.Execute()
	return buf.String(), err
}

func TestListCommand(t *testing.T) {
	dir := setupTestConfig(t)
	out, err := runCommand(t, dir, "list")
	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}
	for _, want := range []string{"github", "remote-api", "stdio", "http", "NAME"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got:\n%s", want, out)
		}
	}
}

func TestListCommandEmpty(t *testing.T) {
	dir := t.TempDir()
	out, err := runCommand(t, dir, "list")
	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}
	if !strings.Contains(out, "No servers") {
		t.Errorf("expected 'No servers' message, got:\n%s", out)
	}
}

func TestVersionCommand(t *testing.T) {
	cmd := newRootCmd("1.2.3", "deadbeef", "2026-03-17")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "1.2.3") {
		t.Errorf("expected version '1.2.3' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "deadbeef") {
		t.Errorf("expected commit 'deadbeef' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "2026-03-17") {
		t.Errorf("expected date '2026-03-17' in output, got:\n%s", out)
	}
}

func TestSyncCommand(t *testing.T) {
	dir := setupTestConfig(t)
	projDir := filepath.Join(dir, "myproject")

	out, err := runCommand(t, dir, "sync", "myproject")
	if err != nil {
		t.Fatalf("sync command failed: %v", err)
	}
	if !strings.Contains(out, "github") {
		t.Errorf("expected sync output to contain 'github', got:\n%s", out)
	}
	if !strings.Contains(out, "myproject") {
		t.Errorf("expected sync output to contain project name, got:\n%s", out)
	}

	// Verify .mcp.json was created.
	mcpPath := filepath.Join(projDir, ".mcp.json")
	if _, err := os.Stat(mcpPath); os.IsNotExist(err) {
		t.Errorf("expected .mcp.json to be created at %s", mcpPath)
	}
}

func TestSyncCommandNoArgs(t *testing.T) {
	dir := setupTestConfig(t)
	_, err := runCommand(t, dir, "sync")
	if err == nil {
		t.Error("expected error when no project name and no --all flag")
	}
}

func TestSyncAllCommand(t *testing.T) {
	dir := setupTestConfig(t)
	out, err := runCommand(t, dir, "sync", "--all")
	if err != nil {
		t.Fatalf("sync --all failed: %v", err)
	}
	if !strings.Contains(out, "myproject") {
		t.Errorf("expected output to contain 'myproject', got:\n%s", out)
	}
}

func TestDiffCommand(t *testing.T) {
	dir := setupTestConfig(t)

	// Diff before sync should show drift (no .mcp.json exists).
	out, err := runCommand(t, dir, "diff", "myproject")
	if err != nil {
		t.Fatalf("diff command failed: %v", err)
	}
	// Should show a diff since the file doesn't exist.
	if strings.Contains(out, "No drift detected") {
		t.Errorf("expected drift to be detected before sync, got:\n%s", out)
	}

	// Sync first, then diff should show no drift.
	_, err = runCommand(t, dir, "sync", "myproject")
	if err != nil {
		t.Fatalf("sync command failed: %v", err)
	}
	out, err = runCommand(t, dir, "diff", "myproject")
	if err != nil {
		t.Fatalf("diff command (after sync) failed: %v", err)
	}
	if !strings.Contains(out, "No drift detected") {
		t.Errorf("expected no drift after sync, got:\n%s", out)
	}
}

func TestOverrideCommand(t *testing.T) {
	dir := setupTestConfig(t)
	out, err := runCommand(t, dir, "override", "myproject", "github", "--env", "GITHUB_TOKEN=my-token")
	if err != nil {
		t.Fatalf("override command failed: %v", err)
	}
	if !strings.Contains(out, "Override set") {
		t.Errorf("expected 'Override set' message, got:\n%s", out)
	}
}

func TestOverrideCommandNoFlags(t *testing.T) {
	dir := setupTestConfig(t)
	_, err := runCommand(t, dir, "override", "myproject", "github")
	if err == nil {
		t.Error("expected error when no override flags provided")
	}
}

func TestRunCommandNoArgs(t *testing.T) {
	dir := setupTestConfig(t)
	_, err := runCommand(t, dir, "run")
	if err == nil {
		t.Fatal("expected error when no project name given")
	}
	if !strings.Contains(err.Error(), "project name required") {
		t.Errorf("expected 'project name required' error, got: %v", err)
	}
}

func TestRunCommandNotFound(t *testing.T) {
	dir := setupTestConfig(t)
	_, err := runCommand(t, dir, "run", "nonexistent-project")
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestRunCommandDryRun(t *testing.T) {
	dir := setupTestConfig(t)

	// Sync first so that the sync step inside run succeeds.
	_, err := runCommand(t, dir, "sync", "myproject")
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	out, err := runCommand(t, dir, "run", "myproject", "--dry-run")
	if err != nil {
		t.Fatalf("run --dry-run failed: %v", err)
	}
	if !strings.Contains(out, "Would run:") {
		t.Errorf("expected 'Would run:' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Directory:") {
		t.Errorf("expected 'Directory:' in output, got:\n%s", out)
	}
}

func TestRootCommandNonTTY(t *testing.T) {
	dir := setupTestConfig(t)
	// When stdout is not a TTY (as in tests), root command should show help.
	out, err := runCommand(t, dir)
	if err != nil {
		t.Fatalf("root command failed: %v", err)
	}
	if !strings.Contains(out, "hystak") {
		t.Errorf("expected help output to contain 'hystak', got:\n%s", out)
	}
}
