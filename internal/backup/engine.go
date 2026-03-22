package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lcrostarosa/hystak/internal/model"
)

// DefaultMaxBackups is the default number of backups to keep per scope.
const DefaultMaxBackups = 10

// BackupEntry describes a single backup file.
type BackupEntry struct {
	Timestamp  time.Time
	ClientType model.ClientType
	Scope      string // "global" or sanitized project path
	SourcePath string // original config file path
	BackupPath string // path to backup copy
}

// Manager handles creating, listing, restoring, and pruning backups.
type Manager struct {
	BackupDir  string
	MaxBackups int
}

// NewManager creates a Manager rooted at backupDir.
func NewManager(backupDir string) *Manager {
	return &Manager{
		BackupDir:  backupDir,
		MaxBackups: DefaultMaxBackups,
	}
}

// Create copies the config file at configPath into the backup directory.
// Returns a no-op (zero BackupEntry, nil error) if the source file does not exist.
// Calls Prune after a successful backup.
func (m *Manager) Create(clientType model.ClientType, projectPath, configPath string) (BackupEntry, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return BackupEntry{}, nil
	}

	scope := scopeDir(projectPath)
	dir := filepath.Join(m.BackupDir, string(clientType), scope)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return BackupEntry{}, fmt.Errorf("creating backup dir: %w", err)
	}

	now := time.Now().Truncate(time.Second)
	ts := now.Format("2006-01-02T15-04-05")
	base := filepath.Base(configPath)
	backupPath := filepath.Join(dir, ts+"."+base)

	if err := copyFile(configPath, backupPath); err != nil {
		return BackupEntry{}, fmt.Errorf("copying config to backup: %w", err)
	}

	entry := BackupEntry{
		Timestamp:  now,
		ClientType: clientType,
		Scope:      scope,
		SourcePath: configPath,
		BackupPath: backupPath,
	}

	if err := m.Prune(m.MaxBackups); err != nil {
		return entry, fmt.Errorf("pruning backups: %w", err)
	}

	return entry, nil
}

// List returns backups for a specific client+project scope, newest first.
func (m *Manager) List(clientType model.ClientType, projectPath string) ([]BackupEntry, error) {
	scope := scopeDir(projectPath)
	dir := filepath.Join(m.BackupDir, string(clientType), scope)
	return listDir(dir, clientType, scope)
}

// ListAll returns all backups across all scopes, newest first.
func (m *Manager) ListAll() ([]BackupEntry, error) {
	var entries []BackupEntry

	clientDirs, err := os.ReadDir(m.BackupDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	for _, cd := range clientDirs {
		if !cd.IsDir() {
			continue
		}
		ct := model.ClientType(cd.Name())
		clientPath := filepath.Join(m.BackupDir, cd.Name())

		scopeDirs, err := walkScopeDirs(clientPath)
		if err != nil {
			continue
		}
		for _, sd := range scopeDirs {
			dirEntries, err := listDir(sd.path, ct, sd.scope)
			if err != nil {
				continue
			}
			entries = append(entries, dirEntries...)
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	return entries, nil
}

// Restore copies a backup back to its source path.
// Creates a safety backup of the current config first.
func (m *Manager) Restore(entry BackupEntry) error {
	// Safety backup of current config before overwriting.
	if _, err := os.Stat(entry.SourcePath); err == nil {
		safetyDir := filepath.Dir(entry.BackupPath)
		ts := time.Now().Truncate(time.Second).Format("2006-01-02T15-04-05")
		base := filepath.Base(entry.SourcePath)
		safetyPath := filepath.Join(safetyDir, ts+".pre-restore."+base)
		if err := copyFile(entry.SourcePath, safetyPath); err != nil {
			return fmt.Errorf("creating safety backup: %w", err)
		}
	}

	if err := copyFile(entry.BackupPath, entry.SourcePath); err != nil {
		return fmt.Errorf("restoring backup: %w", err)
	}

	return nil
}

// Prune keeps the newest maxPerScope backups in each scope directory and deletes the rest.
func (m *Manager) Prune(maxPerScope int) error {
	clientDirs, err := os.ReadDir(m.BackupDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, cd := range clientDirs {
		if !cd.IsDir() {
			continue
		}
		clientPath := filepath.Join(m.BackupDir, cd.Name())

		scopeDirs, err := walkScopeDirs(clientPath)
		if err != nil {
			continue
		}
		for _, sd := range scopeDirs {
			if err := pruneDir(sd.path, maxPerScope); err != nil {
				return err
			}
		}
	}

	return nil
}

// scopeDir returns the backup subdirectory name for a project path.
// Empty or "~" means global scope; otherwise the path is sanitized.
func scopeDir(projectPath string) string {
	if projectPath == "" || projectPath == "~" {
		return "global"
	}
	return filepath.Join("projects", sanitizePath(projectPath))
}

// sanitizePath converts a filesystem path to a safe directory name.
// Leading separators are stripped, remaining separators become underscores.
func sanitizePath(p string) string {
	p = filepath.Clean(p)
	p = strings.TrimLeft(p, string(filepath.Separator))
	return strings.ReplaceAll(p, string(filepath.Separator), "_")
}

type scopeEntry struct {
	path  string // absolute filesystem path
	scope string // scope name (e.g. "global" or "projects/myproj")
}

// walkScopeDirs finds all scope directories under a client directory.
// Handles both flat scopes (global/) and nested scopes (projects/<name>/).
func walkScopeDirs(clientPath string) ([]scopeEntry, error) {
	entries, err := os.ReadDir(clientPath)
	if err != nil {
		return nil, err
	}

	var result []scopeEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dirPath := filepath.Join(clientPath, e.Name())
		if e.Name() == "projects" {
			// Nested: projects/<name>/
			subs, err := os.ReadDir(dirPath)
			if err != nil {
				continue
			}
			for _, sub := range subs {
				if sub.IsDir() {
					result = append(result, scopeEntry{
						path:  filepath.Join(dirPath, sub.Name()),
						scope: filepath.Join("projects", sub.Name()),
					})
				}
			}
		} else {
			result = append(result, scopeEntry{
				path:  dirPath,
				scope: e.Name(),
			})
		}
	}
	return result, nil
}

// listDir reads backup files from a single scope directory, newest first.
func listDir(dir string, clientType model.ClientType, scope string) ([]BackupEntry, error) {
	files, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var entries []BackupEntry
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		ts, ok := parseTimestamp(f.Name())
		if !ok {
			continue
		}
		entries = append(entries, BackupEntry{
			Timestamp:  ts,
			ClientType: clientType,
			Scope:      scope,
			BackupPath: filepath.Join(dir, f.Name()),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	return entries, nil
}

// parseTimestamp extracts the timestamp from a backup filename.
// Expected format: "2006-01-02T15-04-05.filename" or "2006-01-02T15-04-05.pre-restore.filename"
func parseTimestamp(name string) (time.Time, bool) {
	if len(name) < 19 {
		return time.Time{}, false
	}
	ts, err := time.Parse("2006-01-02T15-04-05", name[:19])
	if err != nil {
		return time.Time{}, false
	}
	return ts, true
}

// pruneDir removes the oldest files in dir, keeping at most max.
func pruneDir(dir string, max int) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	// Filter to regular files only.
	var regular []os.DirEntry
	for _, f := range files {
		if !f.IsDir() {
			regular = append(regular, f)
		}
	}

	if len(regular) <= max {
		return nil
	}

	// Sort by name descending (newest first since names start with timestamp).
	sort.Slice(regular, func(i, j int) bool {
		return regular[i].Name() > regular[j].Name()
	})

	for _, f := range regular[max:] {
		if err := os.Remove(filepath.Join(dir, f.Name())); err != nil {
			return err
		}
	}

	return nil
}

// copyFile copies src to dst, preserving permissions.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Close()
}
