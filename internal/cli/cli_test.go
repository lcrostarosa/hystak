package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/profile"
	"github.com/spf13/cobra"
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

func runCommandWithInput(t *testing.T, dir string, stdin string, args ...string) (string, error) {
	t.Helper()
	cmd := newRootCmd("test-version", "abc123", "2026-01-01")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(append([]string{"--config-dir", dir}, args...))
	err := cmd.Execute()
	return buf.String(), err
}

// ---- Backup command tests ----

func TestBackupCommand(t *testing.T) {
	dir := setupTestConfig(t)
	// Sync first to create .mcp.json.
	if _, err := runCommand(t, dir, "sync", "myproject"); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	out, err := runCommand(t, dir, "backup", "myproject")
	if err != nil {
		t.Fatalf("backup failed: %v", err)
	}
	if !strings.Contains(out, "backed up") {
		t.Errorf("expected 'backed up' in output, got:\n%s", out)
	}
}

func TestBackupCommand_NoArgs(t *testing.T) {
	dir := setupTestConfig(t)
	_, err := runCommand(t, dir, "backup")
	if err == nil {
		t.Fatal("expected error when no project name and no --all")
	}
}

func TestBackupCommand_All(t *testing.T) {
	dir := setupTestConfig(t)
	if _, err := runCommand(t, dir, "sync", "myproject"); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	out, err := runCommand(t, dir, "backup", "--all")
	if err != nil {
		t.Fatalf("backup --all failed: %v", err)
	}
	if !strings.Contains(out, "myproject") {
		t.Errorf("expected project name in output, got:\n%s", out)
	}
	if !strings.Contains(out, "backed up") {
		t.Errorf("expected 'backed up' in output, got:\n%s", out)
	}
}

func TestBackupCommand_ListEmpty(t *testing.T) {
	dir := setupTestConfig(t)
	out, err := runCommand(t, dir, "backup", "--list", "myproject")
	if err != nil {
		t.Fatalf("backup --list failed: %v", err)
	}
	if !strings.Contains(out, "No backups") {
		t.Errorf("expected 'No backups' message, got:\n%s", out)
	}
}

func TestBackupCommand_ListAfterBackup(t *testing.T) {
	dir := setupTestConfig(t)
	if _, err := runCommand(t, dir, "sync", "myproject"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCommand(t, dir, "backup", "myproject"); err != nil {
		t.Fatal(err)
	}
	out, err := runCommand(t, dir, "backup", "--list", "myproject")
	if err != nil {
		t.Fatalf("backup --list failed: %v", err)
	}
	for _, want := range []string{"TIMESTAMP", "CLIENT", "SCOPE", "PATH"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in list output, got:\n%s", want, out)
		}
	}
}

func TestBackupCommand_ListAll(t *testing.T) {
	dir := setupTestConfig(t)
	if _, err := runCommand(t, dir, "sync", "myproject"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCommand(t, dir, "backup", "myproject"); err != nil {
		t.Fatal(err)
	}
	out, err := runCommand(t, dir, "backup", "--list")
	if err != nil {
		t.Fatalf("backup --list (all) failed: %v", err)
	}
	if !strings.Contains(out, "TIMESTAMP") {
		t.Errorf("expected table header in output, got:\n%s", out)
	}
}

func TestBackupCommand_ProjectNotFound(t *testing.T) {
	dir := setupTestConfig(t)
	_, err := runCommand(t, dir, "backup", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
}

// ---- Restore command tests ----

func TestRestoreCommand_NoArgs(t *testing.T) {
	dir := setupTestConfig(t)
	_, err := runCommandWithInput(t, dir, "", "restore")
	if err == nil {
		t.Fatal("expected error when no project name and no --global")
	}
}

func TestRestoreCommand_NoBackups(t *testing.T) {
	dir := setupTestConfig(t)
	out, err := runCommandWithInput(t, dir, "", "restore", "myproject")
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if !strings.Contains(out, "No backups") {
		t.Errorf("expected 'No backups' message, got:\n%s", out)
	}
}

func TestRestoreCommand_WithIndex(t *testing.T) {
	dir := setupTestConfig(t)
	if _, err := runCommand(t, dir, "sync", "myproject"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCommand(t, dir, "backup", "myproject"); err != nil {
		t.Fatal(err)
	}
	out, err := runCommandWithInput(t, dir, "y\n", "restore", "myproject", "--index", "0")
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if !strings.Contains(out, "Restored") {
		t.Errorf("expected 'Restored' in output, got:\n%s", out)
	}
}

func TestRestoreCommand_IndexOutOfRange(t *testing.T) {
	dir := setupTestConfig(t)
	if _, err := runCommand(t, dir, "sync", "myproject"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCommand(t, dir, "backup", "myproject"); err != nil {
		t.Fatal(err)
	}
	_, err := runCommandWithInput(t, dir, "y\n", "restore", "myproject", "--index", "99")
	if err == nil {
		t.Fatal("expected error for out of range index")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Errorf("expected 'out of range' error, got: %v", err)
	}
}

func TestRestoreCommand_Cancelled(t *testing.T) {
	dir := setupTestConfig(t)
	if _, err := runCommand(t, dir, "sync", "myproject"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCommand(t, dir, "backup", "myproject"); err != nil {
		t.Fatal(err)
	}
	out, err := runCommandWithInput(t, dir, "n\n", "restore", "myproject", "--index", "0")
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if !strings.Contains(out, "Cancelled") {
		t.Errorf("expected 'Cancelled' in output, got:\n%s", out)
	}
}

func TestRestoreCommand_InteractiveSelect(t *testing.T) {
	dir := setupTestConfig(t)
	if _, err := runCommand(t, dir, "sync", "myproject"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCommand(t, dir, "backup", "myproject"); err != nil {
		t.Fatal(err)
	}
	out, err := runCommandWithInput(t, dir, "0\ny\n", "restore", "myproject")
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if !strings.Contains(out, "Restored") {
		t.Errorf("expected 'Restored' in output, got:\n%s", out)
	}
}

func TestRestoreCommand_InvalidSelection(t *testing.T) {
	dir := setupTestConfig(t)
	if _, err := runCommand(t, dir, "sync", "myproject"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCommand(t, dir, "backup", "myproject"); err != nil {
		t.Fatal(err)
	}
	_, err := runCommandWithInput(t, dir, "abc\n", "restore", "myproject")
	if err == nil {
		t.Fatal("expected error for invalid selection")
	}
	if !strings.Contains(err.Error(), "invalid selection") {
		t.Errorf("expected 'invalid selection' error, got: %v", err)
	}
}

func TestRestoreCommand_ProjectNotFound(t *testing.T) {
	dir := setupTestConfig(t)
	_, err := runCommandWithInput(t, dir, "", "restore", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
}

func TestRestoreCommand_Global(t *testing.T) {
	dir := setupTestConfig(t)
	// Sync creates project-level backups, not global ones.
	if _, err := runCommand(t, dir, "sync", "myproject"); err != nil {
		t.Fatal(err)
	}
	// restore --global should find no global backups.
	out, err := runCommandWithInput(t, dir, "", "restore", "--global")
	if err != nil {
		t.Fatalf("restore --global failed: %v", err)
	}
	if !strings.Contains(out, "No backups") {
		t.Errorf("expected 'No backups' message for global scope, got:\n%s", out)
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

// ---- Sync --profile tests ----

func TestSyncCommandWithProfile(t *testing.T) {
	dir := setupTestConfig(t)

	// Create a profile-aware project config.
	projDir := filepath.Join(dir, "myproject")
	projYAML := fmt.Sprintf(`projects:
  myproject:
    path: %s
    clients: [claude-code]
    mcps:
      - github
    profiles:
      light:
        mcps: [github]
    active_profile: light
    launched: true
`, projDir)
	if err := os.WriteFile(filepath.Join(dir, "projects.yaml"), []byte(projYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCommand(t, dir, "sync", "myproject", "--profile", "light")
	if err != nil {
		t.Fatalf("sync --profile failed: %v", err)
	}
	if !strings.Contains(out, "github") {
		t.Errorf("expected output to mention 'github', got:\n%s", out)
	}
}

func TestSyncCommandWithInvalidProfile(t *testing.T) {
	dir := setupTestConfig(t)
	_, err := runCommand(t, dir, "sync", "myproject", "--profile", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent profile")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// ---- Run --profile tests ----

func TestRunCommandWithProfileDryRun(t *testing.T) {
	dir := setupTestConfig(t)
	projDir := filepath.Join(dir, "myproject")

	// Create a profile-aware project config.
	projYAML := fmt.Sprintf(`projects:
  myproject:
    path: %s
    clients: [claude-code]
    mcps:
      - github
    profiles:
      light:
        mcps: [github]
`, projDir)
	if err := os.WriteFile(filepath.Join(dir, "projects.yaml"), []byte(projYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	// Sync first so the sync step inside run succeeds.
	_, err := runCommand(t, dir, "sync", "myproject")
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	out, err := runCommand(t, dir, "run", "myproject", "--profile", "light", "--dry-run")
	if err != nil {
		t.Fatalf("run --profile --dry-run failed: %v", err)
	}
	if !strings.Contains(out, "Would run:") {
		t.Errorf("expected 'Would run:' in output, got:\n%s", out)
	}
}

// ---- Configure flag tests ----

func TestConfigureFlagHelpText(t *testing.T) {
	// Verify that --configure flag is available.
	cmd := newRootCmd("test", "abc", "2026-01-01")
	flag := cmd.Flags().Lookup("configure")
	if flag == nil {
		t.Fatal("expected --configure flag on root command")
	}
	if flag.Usage == "" {
		t.Error("expected --configure flag to have usage text")
	}
}

func TestRunCommandProfileFlagHelpText(t *testing.T) {
	cmd := newRootCmd("test", "abc", "2026-01-01")
	var runCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "run" {
			runCmd = c
			break
		}
	}
	if runCmd == nil {
		t.Fatal("expected 'run' subcommand")
	}
	flag := runCmd.Flags().Lookup("profile")
	if flag == nil {
		t.Fatal("expected --profile flag on run command")
	}
}

func TestSyncCommandProfileFlagHelpText(t *testing.T) {
	cmd := newRootCmd("test", "abc", "2026-01-01")
	var syncCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "sync" {
			syncCmd = c
			break
		}
	}
	if syncCmd == nil {
		t.Fatal("expected 'sync' subcommand")
	}
	flag := syncCmd.Flags().Lookup("profile")
	if flag == nil {
		t.Fatal("expected --profile flag on sync command")
	}
}

// ---- Isolation tests ----

func TestGetIsolation_None(t *testing.T) {
	proj := model.Project{Name: "test"}
	if got := getIsolation(proj); got != profile.IsolationNone {
		t.Fatalf("expected none, got %s", got)
	}
}

func TestGetIsolation_NoActiveProfile(t *testing.T) {
	proj := model.Project{
		Name: "test",
		Profiles: map[string]model.ProjectProfile{
			"dev": {Isolation: "worktree"},
		},
	}
	if got := getIsolation(proj); got != profile.IsolationNone {
		t.Fatalf("expected none without active profile, got %s", got)
	}
}

func TestGetIsolation_Worktree(t *testing.T) {
	proj := model.Project{
		Name:          "test",
		ActiveProfile: "dev",
		Profiles: map[string]model.ProjectProfile{
			"dev": {Isolation: "worktree"},
		},
	}
	if got := getIsolation(proj); got != profile.IsolationWorktree {
		t.Fatalf("expected worktree, got %s", got)
	}
}

func TestGetIsolation_Lock(t *testing.T) {
	proj := model.Project{
		Name:          "test",
		ActiveProfile: "dev",
		Profiles: map[string]model.ProjectProfile{
			"dev": {Isolation: "lock"},
		},
	}
	if got := getIsolation(proj); got != profile.IsolationLock {
		t.Fatalf("expected lock, got %s", got)
	}
}

func TestGetIsolation_UnknownFallsToNone(t *testing.T) {
	proj := model.Project{
		Name:          "test",
		ActiveProfile: "dev",
		Profiles: map[string]model.ProjectProfile{
			"dev": {Isolation: "unknown"},
		},
	}
	if got := getIsolation(proj); got != profile.IsolationNone {
		t.Fatalf("expected none for unknown strategy, got %s", got)
	}
}

// ---- Post-exit prompt tests ----

func TestPromptPostExit_Relaunch(t *testing.T) {
	for _, input := range []string{"r\n", "R\n", "relaunch\n"} {
		t.Run(input, func(t *testing.T) {
			reader := strings.NewReader(input)
			w := new(bytes.Buffer)
			action := promptPostExit(reader, w)
			if action != actionRelaunch {
				t.Errorf("input %q: expected actionRelaunch, got %d", input, action)
			}
		})
	}
}

func TestPromptPostExit_Configure(t *testing.T) {
	for _, input := range []string{"c\n", "C\n", "configure\n"} {
		t.Run(input, func(t *testing.T) {
			reader := strings.NewReader(input)
			w := new(bytes.Buffer)
			action := promptPostExit(reader, w)
			if action != actionConfigure {
				t.Errorf("input %q: expected actionConfigure, got %d", input, action)
			}
		})
	}
}

func TestPromptPostExit_Quit(t *testing.T) {
	for _, input := range []string{"q\n", "Q\n", "quit\n", "\n", ""} {
		t.Run(input, func(t *testing.T) {
			reader := strings.NewReader(input)
			w := new(bytes.Buffer)
			action := promptPostExit(reader, w)
			if action != actionQuit {
				t.Errorf("input %q: expected actionQuit, got %d", input, action)
			}
		})
	}
}

func TestPromptPostExit_EmptyInput(t *testing.T) {
	reader := strings.NewReader("")
	w := new(bytes.Buffer)
	action := promptPostExit(reader, w)
	if action != actionQuit {
		t.Errorf("empty input: expected actionQuit, got %d", action)
	}
}

func TestPromptPostExit_ShowsPromptText(t *testing.T) {
	reader := strings.NewReader("q\n")
	w := new(bytes.Buffer)
	promptPostExit(reader, w)
	if !strings.Contains(w.String(), "[R]elaunch") {
		t.Errorf("expected prompt text to contain '[R]elaunch', got: %s", w.String())
	}
	if !strings.Contains(w.String(), "[C]onfigure") {
		t.Errorf("expected prompt text to contain '[C]onfigure', got: %s", w.String())
	}
	if !strings.Contains(w.String(), "[Q]uit") {
		t.Errorf("expected prompt text to contain '[Q]uit', got: %s", w.String())
	}
}

// ---- appendContinue tests ----

func TestAppendContinue_AddsFlag(t *testing.T) {
	args := []string{"--verbose"}
	result := appendContinue(args)
	if len(result) != 2 {
		t.Fatalf("expected 2 args, got %d", len(result))
	}
	if result[1] != "--continue" {
		t.Errorf("expected '--continue' at index 1, got %q", result[1])
	}
}

func TestAppendContinue_NoDuplicate(t *testing.T) {
	args := []string{"--continue", "--verbose"}
	result := appendContinue(args)
	if len(result) != 2 {
		t.Fatalf("expected 2 args (no duplicate), got %d", len(result))
	}
}

func TestAppendContinue_EmptyArgs(t *testing.T) {
	result := appendContinue(nil)
	if len(result) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(result))
	}
	if result[0] != "--continue" {
		t.Errorf("expected '--continue', got %q", result[0])
	}
}

func TestAppendContinue_DoesNotMutateOriginal(t *testing.T) {
	original := []string{"--verbose"}
	result := appendContinue(original)
	if len(original) != 1 {
		t.Fatalf("original slice was mutated: len=%d", len(original))
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 args, got %d", len(result))
	}
}

// ---- Dry-run still works with new loop code ----

func TestRunCommandDryRunStillWorks(t *testing.T) {
	dir := setupTestConfig(t)

	// Sync first.
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
	// Dry-run should NOT contain --continue (first launch, not relaunch).
	if strings.Contains(out, "--continue") {
		t.Errorf("dry-run first launch should not contain --continue, got:\n%s", out)
	}
}

// ---- Profile command tests ----

func setupTestConfigWithProfiles(t *testing.T) string {
	t.Helper()
	dir := setupTestConfig(t)

	// Create a global profile.
	profilesDir := filepath.Join(dir, "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	profileYAML := `name: frontend
description: Frontend dev loadout
mcps:
  - browser-mcp
  - github
skills:
  - react-patterns
env:
  NODE_ENV: development
isolation: none
`
	if err := os.WriteFile(filepath.Join(profilesDir, "frontend.yaml"), []byte(profileYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	return dir
}

func TestProfileListCommand(t *testing.T) {
	dir := setupTestConfigWithProfiles(t)
	out, err := runCommand(t, dir, "profile", "list")
	if err != nil {
		t.Fatalf("profile list failed: %v", err)
	}
	// Should show table header and vanilla + frontend.
	for _, want := range []string{"NAME", "SCOPE", "vanilla", "frontend", "global"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got:\n%s", want, out)
		}
	}
}

func TestProfileListCommandEmpty(t *testing.T) {
	dir := t.TempDir()
	out, err := runCommand(t, dir, "profile", "list")
	if err != nil {
		t.Fatalf("profile list failed: %v", err)
	}
	// Should at least show vanilla.
	if !strings.Contains(out, "vanilla") {
		t.Errorf("expected 'vanilla' in output, got:\n%s", out)
	}
}

func TestProfileListCommandWithProject(t *testing.T) {
	dir := setupTestConfigWithProfiles(t)
	projDir := filepath.Join(dir, "myproject")

	// Add project-scoped profiles.
	projYAML := fmt.Sprintf(`projects:
  myproject:
    path: %s
    clients: [claude-code]
    mcps:
      - github
    profiles:
      light:
        description: Light profile
        mcps: [github]
    active_profile: light
    launched: true
`, projDir)
	if err := os.WriteFile(filepath.Join(dir, "projects.yaml"), []byte(projYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCommand(t, dir, "profile", "list", "--project", "myproject")
	if err != nil {
		t.Fatalf("profile list --project failed: %v", err)
	}
	if !strings.Contains(out, "light") {
		t.Errorf("expected 'light' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "myproject") {
		t.Errorf("expected 'myproject' in scope column, got:\n%s", out)
	}
}

func TestProfileExportCommand(t *testing.T) {
	dir := setupTestConfigWithProfiles(t)
	out, err := runCommand(t, dir, "profile", "export", "frontend")
	if err != nil {
		t.Fatalf("profile export failed: %v", err)
	}
	// Should contain YAML fields.
	for _, want := range []string{"name: frontend", "browser-mcp", "react-patterns"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got:\n%s", want, out)
		}
	}
}

func TestProfileExportCommandToFile(t *testing.T) {
	dir := setupTestConfigWithProfiles(t)
	outFile := filepath.Join(t.TempDir(), "exported.yaml")
	out, err := runCommand(t, dir, "profile", "export", "frontend", "-o", outFile)
	if err != nil {
		t.Fatalf("profile export -o failed: %v", err)
	}
	if !strings.Contains(out, "exported to") {
		t.Errorf("expected 'exported to' message, got:\n%s", out)
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading exported file: %v", err)
	}
	if !strings.Contains(string(data), "name: frontend") {
		t.Errorf("exported file should contain profile data, got:\n%s", string(data))
	}
}

func TestProfileExportCommandNotFound(t *testing.T) {
	dir := setupTestConfigWithProfiles(t)
	_, err := runCommand(t, dir, "profile", "export", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent profile")
	}
}

func TestProfileImportCommand(t *testing.T) {
	dir := setupTestConfigWithProfiles(t)

	// Create a YAML file to import.
	importFile := filepath.Join(t.TempDir(), "backend.yaml")
	importYAML := `name: backend
description: Backend loadout
mcps:
  - db-mcp
skills:
  - go-patterns
isolation: lock
`
	if err := os.WriteFile(importFile, []byte(importYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCommand(t, dir, "profile", "import", importFile)
	if err != nil {
		t.Fatalf("profile import failed: %v", err)
	}
	if !strings.Contains(out, "imported") {
		t.Errorf("expected 'imported' message, got:\n%s", out)
	}

	// Verify the profile was saved.
	out2, err := runCommand(t, dir, "profile", "list")
	if err != nil {
		t.Fatalf("profile list after import failed: %v", err)
	}
	if !strings.Contains(out2, "backend") {
		t.Errorf("expected 'backend' in list after import, got:\n%s", out2)
	}
}

func TestProfileImportCommandConflict(t *testing.T) {
	dir := setupTestConfigWithProfiles(t)

	// Try importing a profile with the same name as existing.
	importFile := filepath.Join(t.TempDir(), "frontend.yaml")
	importYAML := `name: frontend
description: Conflicting profile
mcps:
  - other-mcp
`
	if err := os.WriteFile(importFile, []byte(importYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := runCommand(t, dir, "profile", "import", importFile)
	if err == nil {
		t.Fatal("expected error for conflicting profile name")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestProfileImportCommandAsRename(t *testing.T) {
	dir := setupTestConfigWithProfiles(t)

	// Import a conflicting profile under a different name.
	importFile := filepath.Join(t.TempDir(), "frontend.yaml")
	importYAML := `name: frontend
description: Conflicting profile
mcps:
  - other-mcp
`
	if err := os.WriteFile(importFile, []byte(importYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCommand(t, dir, "profile", "import", importFile, "--as", "frontend-v2")
	if err != nil {
		t.Fatalf("profile import --as failed: %v", err)
	}
	if !strings.Contains(out, "frontend-v2") {
		t.Errorf("expected 'frontend-v2' in output, got:\n%s", out)
	}

	// Verify it was saved.
	out2, err := runCommand(t, dir, "profile", "list")
	if err != nil {
		t.Fatalf("profile list after import --as failed: %v", err)
	}
	if !strings.Contains(out2, "frontend-v2") {
		t.Errorf("expected 'frontend-v2' in list, got:\n%s", out2)
	}
}

func TestProfileImportCommandFileNotFound(t *testing.T) {
	dir := setupTestConfigWithProfiles(t)
	_, err := runCommand(t, dir, "profile", "import", "/nonexistent/file.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestProfileExportImportRoundTrip(t *testing.T) {
	dir := setupTestConfigWithProfiles(t)

	// Export the frontend profile.
	exportFile := filepath.Join(t.TempDir(), "frontend-export.yaml")
	_, err := runCommand(t, dir, "profile", "export", "frontend", "-o", exportFile)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	// Import under a new name.
	out, err := runCommand(t, dir, "profile", "import", exportFile, "--as", "frontend-clone")
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if !strings.Contains(out, "frontend-clone") {
		t.Errorf("expected 'frontend-clone' in output, got:\n%s", out)
	}

	// Export the clone and verify content matches.
	out2, err := runCommand(t, dir, "profile", "export", "frontend-clone")
	if err != nil {
		t.Fatalf("export clone failed: %v", err)
	}
	for _, want := range []string{"browser-mcp", "react-patterns", "NODE_ENV"} {
		if !strings.Contains(out2, want) {
			t.Errorf("expected %q in cloned profile export, got:\n%s", want, out2)
		}
	}
}
