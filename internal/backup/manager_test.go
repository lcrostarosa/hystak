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

	expectedName := "myproject_mcp_2026-03-22T10-30-00.json"
	backupPath := filepath.Join(backupsDir, expectedName)
	data, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("backup file not found: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("backup content = %q, want %q", data, content)
	}

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

func TestManager_List(t *testing.T) {
	tmp := t.TempDir()
	backupsDir := filepath.Join(tmp, "backups")

	times := []time.Time{
		time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 22, 11, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC),
	}

	callIdx := 0
	mgr := &Manager{
		backupsDir: backupsDir,
		nowFunc: func() time.Time {
			t := times[callIdx]
			callIdx++
			return t
		},
	}

	configPath := filepath.Join(tmp, ".mcp.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	for range 3 {
		if err := mgr.BackupBeforeSync("proj", configPath); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := mgr.List("proj")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(entries))
	}
	// Newest first
	if !entries[0].Timestamp.After(entries[1].Timestamp) {
		t.Error("entries should be sorted newest first")
	}
}

func TestManager_List_FilterByProject(t *testing.T) {
	tmp := t.TempDir()
	backupsDir := filepath.Join(tmp, "backups")

	fixedTime := time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC)
	mgr := &Manager{
		backupsDir: backupsDir,
		nowFunc:    func() time.Time { return fixedTime },
	}

	configPath := filepath.Join(tmp, ".mcp.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := mgr.BackupBeforeSync("projA", configPath); err != nil {
		t.Fatal(err)
	}
	if err := mgr.BackupBeforeSync("projB", configPath); err != nil {
		t.Fatal(err)
	}

	entries, err := mgr.List("projA")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("entries for projA = %d, want 1", len(entries))
	}
}

func TestManager_Restore(t *testing.T) {
	tmp := t.TempDir()
	mgr := NewManager(filepath.Join(tmp, "backups"))

	backupPath := filepath.Join(tmp, "backup.json")
	if err := os.WriteFile(backupPath, []byte(`{"restored":true}`), 0o600); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(tmp, "target.json")
	if err := mgr.Restore(backupPath, targetPath); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"restored":true}` {
		t.Errorf("restored content = %q", data)
	}
}

func TestManager_LatestForProject(t *testing.T) {
	tmp := t.TempDir()
	backupsDir := filepath.Join(tmp, "backups")

	times := []time.Time{
		time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 22, 11, 0, 0, 0, time.UTC),
	}
	callIdx := 0
	mgr := &Manager{
		backupsDir: backupsDir,
		nowFunc: func() time.Time {
			t := times[callIdx]
			callIdx++
			return t
		},
	}

	configPath := filepath.Join(tmp, ".mcp.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if err := mgr.BackupBeforeSync("proj", configPath); err != nil {
			t.Fatal(err)
		}
	}

	entry, found, err := mgr.LatestForProject("proj")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected to find latest backup")
	}
	if !entry.Timestamp.Equal(times[1]) {
		t.Errorf("latest timestamp = %v, want %v", entry.Timestamp, times[1])
	}
}

func TestManager_LatestForProject_NotFound(t *testing.T) {
	tmp := t.TempDir()
	mgr := NewManager(filepath.Join(tmp, "backups"))

	_, found, err := mgr.LatestForProject("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("should not find backups for nonexistent project")
	}
}

func TestManager_Prune(t *testing.T) {
	tmp := t.TempDir()
	backupsDir := filepath.Join(tmp, "backups")

	times := []time.Time{
		time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 22, 11, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC),
	}
	callIdx := 0
	mgr := &Manager{
		backupsDir: backupsDir,
		nowFunc: func() time.Time {
			t := times[callIdx]
			callIdx++
			return t
		},
	}

	configPath := filepath.Join(tmp, ".mcp.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	for range 3 {
		if err := mgr.BackupBeforeSync("proj", configPath); err != nil {
			t.Fatal(err)
		}
	}

	// Prune to keep only 2
	pruned, err := mgr.Prune(2)
	if err != nil {
		t.Fatal(err)
	}
	if pruned != 1 {
		t.Errorf("pruned = %d, want 1", pruned)
	}

	remaining, err := mgr.List("proj")
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 2 {
		t.Errorf("remaining = %d, want 2", len(remaining))
	}
}

func TestParseBackupFilename(t *testing.T) {
	tests := []struct {
		name    string
		wantOK  bool
		project string
		scope   string
	}{
		{"myproject_mcp_2026-03-22T10-30-00.json", true, "myproject", "mcp"},
		{"proj_settings_2026-01-01T00-00-00.json", true, "proj", "settings"},
		{"invalid.json", false, "", ""},
		{"not-json.txt", false, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, ok := parseBackupFilename(tt.name, "/backups")
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok {
				if entry.Project != tt.project {
					t.Errorf("project = %q, want %q", entry.Project, tt.project)
				}
				if entry.Scope != tt.scope {
					t.Errorf("scope = %q, want %q", entry.Scope, tt.scope)
				}
			}
		})
	}
}
