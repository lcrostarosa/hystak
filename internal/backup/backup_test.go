package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lcrostarosa/hystak/internal/model"
)

func TestCreateBackup(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(filepath.Join(dir, "backups"))

	// Create a fake config file.
	configPath := filepath.Join(dir, ".mcp.json")
	os.WriteFile(configPath, []byte(`{"mcpServers":{}}`), 0o644)

	entry, err := m.Create(model.ClientClaudeCode, "/home/user/proj", configPath)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if entry.BackupPath == "" {
		t.Fatal("expected non-empty BackupPath")
	}

	data, err := os.ReadFile(entry.BackupPath)
	if err != nil {
		t.Fatalf("reading backup: %v", err)
	}
	if string(data) != `{"mcpServers":{}}` {
		t.Errorf("backup content = %q, want %q", data, `{"mcpServers":{}}`)
	}
}

func TestCreateNoFileNoop(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(filepath.Join(dir, "backups"))

	entry, err := m.Create(model.ClientClaudeCode, "/home/user/proj", filepath.Join(dir, "nonexistent.json"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if entry.BackupPath != "" {
		t.Errorf("expected empty BackupPath for nonexistent source, got %q", entry.BackupPath)
	}
}

func TestListOrdering(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(filepath.Join(dir, "backups"))
	scope := "projects/home_user_proj"
	backupDir := filepath.Join(dir, "backups", "claude-code", scope)
	os.MkdirAll(backupDir, 0o755)

	// Create files with different timestamps.
	names := []string{
		"2026-03-18T10-00-00..mcp.json",
		"2026-03-18T12-00-00..mcp.json",
		"2026-03-18T11-00-00..mcp.json",
	}
	for _, name := range names {
		os.WriteFile(filepath.Join(backupDir, name), []byte("{}"), 0o644)
	}

	entries, err := m.List(model.ClientClaudeCode, "/home/user/proj")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}
	// Newest first.
	if entries[0].Timestamp.Hour() != 12 {
		t.Errorf("first entry hour = %d, want 12", entries[0].Timestamp.Hour())
	}
	if entries[2].Timestamp.Hour() != 10 {
		t.Errorf("last entry hour = %d, want 10", entries[2].Timestamp.Hour())
	}
}

func TestListEmpty(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(filepath.Join(dir, "backups"))

	entries, err := m.List(model.ClientClaudeCode, "/nonexistent")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}

func TestRestore(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(filepath.Join(dir, "backups"))

	// Create original config.
	configPath := filepath.Join(dir, ".mcp.json")
	os.WriteFile(configPath, []byte(`{"original":true}`), 0o644)

	// Create a backup.
	entry, err := m.Create(model.ClientClaudeCode, "/home/user/proj", configPath)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Modify the config.
	os.WriteFile(configPath, []byte(`{"modified":true}`), 0o644)

	// Restore.
	if err := m.Restore(entry); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading restored config: %v", err)
	}
	if string(data) != `{"original":true}` {
		t.Errorf("restored content = %q, want %q", data, `{"original":true}`)
	}
}

func TestRestoreCreatesSafetyBackup(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(filepath.Join(dir, "backups"))

	configPath := filepath.Join(dir, ".mcp.json")
	os.WriteFile(configPath, []byte(`{"v1":true}`), 0o644)

	entry, err := m.Create(model.ClientClaudeCode, "/home/user/proj", configPath)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Modify config so the safety backup captures this version.
	os.WriteFile(configPath, []byte(`{"v2":true}`), 0o644)

	if err := m.Restore(entry); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	// Check that a pre-restore safety backup exists.
	backupDir := filepath.Dir(entry.BackupPath)
	files, _ := os.ReadDir(backupDir)
	var found bool
	for _, f := range files {
		if strings.Contains(f.Name(), "pre-restore") {
			found = true
			data, _ := os.ReadFile(filepath.Join(backupDir, f.Name()))
			if string(data) != `{"v2":true}` {
				t.Errorf("safety backup content = %q, want %q", data, `{"v2":true}`)
			}
		}
	}
	if !found {
		t.Error("no safety backup found after restore")
	}
}

func TestPrune(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(filepath.Join(dir, "backups"))
	scope := "projects/home_user_proj"
	backupDir := filepath.Join(dir, "backups", "claude-code", scope)
	os.MkdirAll(backupDir, 0o755)

	// Create 5 backup files.
	for i := 0; i < 5; i++ {
		ts := time.Date(2026, 3, 18, 10+i, 0, 0, 0, time.UTC)
		name := ts.Format("2006-01-02T15-04-05") + "..mcp.json"
		os.WriteFile(filepath.Join(backupDir, name), []byte("{}"), 0o644)
	}

	if err := m.Prune(3); err != nil {
		t.Fatalf("Prune: %v", err)
	}

	files, _ := os.ReadDir(backupDir)
	if len(files) != 3 {
		t.Errorf("got %d files after prune, want 3", len(files))
	}
	// Should keep newest (14, 13, 12).
	if files[0].Name()[:19] != "2026-03-18T12-00-00" {
		t.Errorf("oldest kept = %s, want 2026-03-18T12-00-00", files[0].Name()[:19])
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/home/user/proj", "home_user_proj"},
		{"/", ""},
		{"relative/path", "relative_path"},
		{"/a/b/c/d", "a_b_c_d"},
	}

	for _, tt := range tests {
		got := sanitizePath(tt.input)
		if got != tt.want {
			t.Errorf("sanitizePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestScopeDir(t *testing.T) {
	tests := []struct {
		projectPath string
		want        string
	}{
		{"", "global"},
		{"~", "global"},
		{"/home/user/proj", "projects/home_user_proj"},
	}

	for _, tt := range tests {
		got := scopeDir(tt.projectPath)
		if got != tt.want {
			t.Errorf("scopeDir(%q) = %q, want %q", tt.projectPath, got, tt.want)
		}
	}
}

func TestListAll(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(filepath.Join(dir, "backups"))

	// Create backups in two scopes.
	for _, scope := range []string{"global", "projects/myproj"} {
		d := filepath.Join(dir, "backups", "claude-code", scope)
		os.MkdirAll(d, 0o755)
		os.WriteFile(filepath.Join(d, "2026-03-18T10-00-00..mcp.json"), []byte("{}"), 0o644)
	}

	entries, err := m.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("got %d entries, want 2", len(entries))
	}
}
