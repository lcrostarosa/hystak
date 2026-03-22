package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// UserConfig holds user-level preferences loaded from ~/.hystak/config.yaml.
type UserConfig struct {
	AutoSync     bool   `yaml:"auto_sync"`      // sync all projects on startup (default: true)
	BackupPolicy string `yaml:"backup_policy"`   // "always" (default), "on_change", "never"
	MaxBackups   int    `yaml:"max_backups"`      // per-scope backup retention (default: 10)
	AutoUpdate   bool   `yaml:"auto_update"`      // auto-accept local changes during two-way sync
}

// DefaultUserConfig returns the default configuration.
func DefaultUserConfig() UserConfig {
	return UserConfig{
		AutoSync:     true,
		BackupPolicy: "always",
		MaxBackups:   10,
		AutoUpdate:   false,
	}
}

// UserConfigPath returns the path to config.yaml.
func UserConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

// LoadUserConfig reads config.yaml from the config directory.
// Returns defaults if the file does not exist or is empty.
func LoadUserConfig() UserConfig {
	cfg := DefaultUserConfig()

	data, err := os.ReadFile(UserConfigPath())
	if err != nil || len(data) == 0 {
		return cfg
	}

	_ = yaml.Unmarshal(data, &cfg)

	// Apply defaults for zero values.
	if cfg.BackupPolicy == "" {
		cfg.BackupPolicy = "always"
	}
	if cfg.MaxBackups == 0 {
		cfg.MaxBackups = 10
	}

	return cfg
}
