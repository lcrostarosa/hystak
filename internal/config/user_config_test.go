package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultUserConfig(t *testing.T) {
	cfg := DefaultUserConfig()

	if !cfg.AutoSync {
		t.Error("AutoSync should default to true")
	}
	if cfg.BackupPolicy != "always" {
		t.Errorf("BackupPolicy = %q, want always", cfg.BackupPolicy)
	}
	if cfg.MaxBackups != 10 {
		t.Errorf("MaxBackups = %d, want 10", cfg.MaxBackups)
	}
	if cfg.AutoUpdate {
		t.Error("AutoUpdate should default to false")
	}
}

func TestLoadUserConfig_MissingFile(t *testing.T) {
	t.Setenv("HYSTAK_CONFIG_DIR", t.TempDir())

	cfg := LoadUserConfig()
	def := DefaultUserConfig()

	if cfg != def {
		t.Errorf("missing file should return defaults, got %+v", cfg)
	}
}

func TestLoadUserConfig_ValidFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HYSTAK_CONFIG_DIR", dir)

	content := `
auto_sync: false
backup_policy: on_change
max_backups: 5
auto_update: true
`
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := LoadUserConfig()

	if cfg.AutoSync {
		t.Error("AutoSync should be false")
	}
	if cfg.BackupPolicy != "on_change" {
		t.Errorf("BackupPolicy = %q", cfg.BackupPolicy)
	}
	if cfg.MaxBackups != 5 {
		t.Errorf("MaxBackups = %d", cfg.MaxBackups)
	}
	if !cfg.AutoUpdate {
		t.Error("AutoUpdate should be true")
	}
}

func TestLoadUserConfig_PartialFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HYSTAK_CONFIG_DIR", dir)

	// Only set auto_sync; other fields should get defaults.
	content := "auto_sync: false\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := LoadUserConfig()

	if cfg.AutoSync {
		t.Error("AutoSync should be false")
	}
	if cfg.BackupPolicy != "always" {
		t.Errorf("BackupPolicy should default, got %q", cfg.BackupPolicy)
	}
	if cfg.MaxBackups != 10 {
		t.Errorf("MaxBackups should default, got %d", cfg.MaxBackups)
	}
}
