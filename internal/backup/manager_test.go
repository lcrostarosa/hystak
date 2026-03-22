package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestManager_BackupBeforeSync(t *testing.T) {
	tmp := t.TempDir()
	backupsDir := filepath.Join(tmp, "backups")
	configPath := filepath.Join(tmp, ".mcp.json")

	// Write a config file to back up
	content := []byte(`{"mcpServers":{}}`)
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	fixedTime := time.Date(2026, 3, 22, 10, 30, 0, 0, time.UTC)
	mgr := &Manager{
		backupsDir: backupsDir,
		nowFunc:    func() time.Time { return fixedTime },
	}

	if err := mgr.BackupBeforeSync("myproject", configPath); err != nil {
		t.Fatal(err)
	}

	// Verify backup file exists
	expectedName := "myproject_mcp_2026-03-22T10-30-00.json"
	backupPath := filepath.Join(backupsDir, expectedName)
	data, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("backup file not found: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("backup content = %q, want %q", data, content)
	}

	// Verify permissions are 0600 (restricted)
	info, err := os.Stat(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("backup permissions = %o, want 0600", perm)
	}
}

func TestManager_BackupBeforeSync_MissingFile(t *testing.T) {
	tmp := t.TempDir()
	mgr := NewManager(filepath.Join(tmp, "backups"))

	// Backing up a nonexistent file should succeed silently
	err := mgr.BackupBeforeSync("proj", filepath.Join(tmp, "nonexistent.json"))
	if err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}
}

func TestManager_BackupBeforeSync_SettingsScope(t *testing.T) {
	tmp := t.TempDir()
	backupsDir := filepath.Join(tmp, "backups")
	configDir := filepath.Join(tmp, ".claude")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "settings.local.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	fixedTime := time.Date(2026, 3, 22, 10, 30, 0, 0, time.UTC)
	mgr := &Manager{
		backupsDir: backupsDir,
		nowFunc:    func() time.Time { return fixedTime },
	}

	if err := mgr.BackupBeforeSync("proj", configPath); err != nil {
		t.Fatal(err)
	}

	// Verify filename uses "settings" scope
	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 backup file, got %d", len(entries))
	}
	if !strings.Contains(entries[0].Name(), "_settings_") {
		t.Errorf("expected settings scope in filename, got: %s", entries[0].Name())
	}
}

func TestScopeFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/project/.mcp.json", "mcp"},
		{"/project/.claude/settings.local.json", "settings"},
		{"/home/user/.claude.json", "claude"},
		{"/some/other/file.json", "config"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := scopeFromPath(tt.path)
			if got != tt.want {
				t.Errorf("scopeFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
