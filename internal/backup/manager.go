package backup

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/hystak/hystak/internal/config"
)

// Manager handles config backups before sync operations (S-040).
type Manager struct {
	backupsDir string
	nowFunc    func() time.Time // injectable for testing
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

// BackupBeforeSync copies the current config file to the backups directory.
// The backup is named <project>_<scope>_<timestamp>.json.
// Returns nil if the source file does not exist (nothing to back up).
func (m *Manager) BackupBeforeSync(projectName, configPath string) error {
	data, err := os.ReadFile(configPath)
	switch {
	case err == nil:
		// file exists
	case errors.Is(err, fs.ErrNotExist):
		return nil // nothing to back up
	default:
		return fmt.Errorf("reading config for backup: %w", err)
	}

	if err := os.MkdirAll(m.backupsDir, 0o755); err != nil {
		return fmt.Errorf("creating backups directory: %w", err)
	}

	scope := scopeFromPath(configPath)
	ts := m.nowFunc().UTC().Format("2006-01-02T15-04-05")
	filename := fmt.Sprintf("%s_%s_%s.json", projectName, scope, ts)
	backupPath := filepath.Join(m.backupsDir, filename)

	// Write backup with restricted permissions (may contain secrets)
	return config.AtomicWrite(backupPath, data, 0o600)
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
