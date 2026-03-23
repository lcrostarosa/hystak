package backup

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hystak/hystak/internal/config"
)

// Entry describes a single backup file.
type Entry struct {
	Project   string
	Scope     string
	Timestamp time.Time
	Path      string
}

// Manager handles config backups (S-040, S-066–S-070).
type Manager struct {
	backupsDir string
	nowFunc    func() time.Time
}

// NewManager creates a BackupManager that stores backups in the given directory.
func NewManager(backupsDir string) *Manager {
	return &Manager{
		backupsDir: backupsDir,
		nowFunc:    time.Now,
	}
}

// NewDefaultManager creates a BackupManager using the default backups directory.
func NewDefaultManager() *Manager {
	return NewManager(config.BackupsDir())
}

// BackupBeforeSync copies the current config file to the backups directory (S-040).
// Returns nil if the source file does not exist (nothing to back up).
func (m *Manager) BackupBeforeSync(projectName, configPath string) error {
	data, err := os.ReadFile(configPath)
	switch {
	case err == nil:
		// file exists
	case errors.Is(err, fs.ErrNotExist):
		return nil
	default:
		return fmt.Errorf("reading config for backup: %w", err)
	}

	if err := os.MkdirAll(m.backupsDir, 0o755); err != nil {
		return fmt.Errorf("creating backups directory: %w", err)
	}

	scope := scopeFromPath(configPath)
	ts := m.nowFunc().UTC().Format("2006-01-02T15-04-05")
	filename := fmt.Sprintf("%s--%s--%s.json", projectName, scope, ts)
	backupPath := filepath.Join(m.backupsDir, filename)

	return config.AtomicWrite(backupPath, data, 0o600)
}

// Backup creates a backup of a config file on demand (S-066).
func (m *Manager) Backup(projectName, configPath string) (string, error) {
	if err := m.BackupBeforeSync(projectName, configPath); err != nil {
		return "", err
	}
	scope := scopeFromPath(configPath)
	ts := m.nowFunc().UTC().Format("2006-01-02T15-04-05")
	filename := fmt.Sprintf("%s--%s--%s.json", projectName, scope, ts)
	return filepath.Join(m.backupsDir, filename), nil
}

// List returns all backup entries for a project, sorted newest first (S-067).
// If projectName is empty, returns all backups.
func (m *Manager) List(projectName string) ([]Entry, error) {
	entries, err := os.ReadDir(m.backupsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var result []Entry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		entry, ok := parseBackupFilename(e.Name(), m.backupsDir)
		if !ok {
			continue
		}
		if projectName != "" && entry.Project != projectName {
			continue
		}
		result = append(result, entry)
	}

	// Sort newest first
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})
	return result, nil
}

// Restore copies a backup file back to the original config path (S-068).
func (m *Manager) Restore(backupPath, targetPath string) error {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("reading backup: %w", err)
	}
	return config.AtomicWrite(targetPath, data, 0o644)
}

// LatestForProject returns the most recent backup for a project (S-069).
func (m *Manager) LatestForProject(projectName string) (Entry, bool, error) {
	entries, err := m.List(projectName)
	if err != nil {
		return Entry{}, false, err
	}
	if len(entries) == 0 {
		return Entry{}, false, nil
	}
	return entries[0], true, nil
}

// Prune removes old backups exceeding maxBackups per project+scope (S-070).
func (m *Manager) Prune(maxBackups int) (int, error) {
	if maxBackups <= 0 {
		return 0, nil
	}

	entries, err := m.List("")
	if err != nil {
		return 0, err
	}

	// Group by project+scope
	groups := make(map[string][]Entry)
	for _, e := range entries {
		key := e.Project + "--" + e.Scope
		groups[key] = append(groups[key], e)
	}

	pruned := 0
	for _, group := range groups {
		// Already sorted newest first from List
		if len(group) <= maxBackups {
			continue
		}
		for _, old := range group[maxBackups:] {
			if err := os.Remove(old.Path); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return pruned, fmt.Errorf("removing old backup %q: %w", old.Path, err)
			}
			pruned++
		}
	}
	return pruned, nil
}

// parseBackupFilename extracts project, scope, and timestamp from a backup filename.
// Format: <project>--<scope>--<timestamp>.json
// Uses "--" as delimiter so project names containing underscores are parsed correctly.
func parseBackupFilename(name, dir string) (Entry, bool) {
	if !strings.HasSuffix(name, ".json") {
		return Entry{}, false
	}
	base := strings.TrimSuffix(name, ".json")
	parts := strings.SplitN(base, "--", 3)
	if len(parts) != 3 {
		return Entry{}, false
	}

	ts, err := time.Parse("2006-01-02T15-04-05", parts[2])
	if err != nil {
		return Entry{}, false
	}

	return Entry{
		Project:   parts[0],
		Scope:     parts[1],
		Timestamp: ts,
		Path:      filepath.Join(dir, name),
	}, true
}

// scopeFromPath derives a scope label from the config file path.
func scopeFromPath(path string) string {
	base := filepath.Base(path)
	switch base {
	case ".mcp.json":
		return "mcp"
	case "settings.local.json":
		return "settings"
	case ".claude.json":
		return "claude"
	default:
		return "config"
	}
}
