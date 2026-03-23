package config

import (
	"errors"
	"io/fs"
	"os"

	hysterr "github.com/hystak/hystak/internal/errors"
	"gopkg.in/yaml.v3"
)

// UserConfig holds user preferences from user.yaml.
type UserConfig struct {
	AutoSync     bool   `yaml:"auto_sync"`
	BackupPolicy string `yaml:"backup_policy"`
	MaxBackups   int    `yaml:"max_backups"`
}

// DefaultUserConfig returns the default user configuration.
func DefaultUserConfig() UserConfig {
	return UserConfig{
		AutoSync:     true,
		BackupPolicy: "always",
		MaxBackups:   10,
	}
}

// LoadUserConfig reads user.yaml from the config directory.
// Returns defaults if the file does not exist.
// Returns a ConfigParseError if the file exists but is malformed.
func LoadUserConfig() (UserConfig, error) {
	path := UserConfigPath()
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		// file exists, parse it
	case errors.Is(err, fs.ErrNotExist):
		return DefaultUserConfig(), nil
	default:
		return UserConfig{}, err
	}

	var cfg UserConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return UserConfig{}, &hysterr.ConfigParseError{Path: path, Err: err}
	}
	return cfg, nil
}

// SaveUserConfig writes user.yaml atomically to the config directory.
func SaveUserConfig(cfg UserConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return AtomicWrite(UserConfigPath(), data, 0o644)
}
