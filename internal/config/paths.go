package config

import (
	"os"
	"path/filepath"
)

// overrideDir is set by CLI --config-dir flag via OverrideDir.
var overrideDir string

// Subdirectories under the hystak config root.
var subdirs = []string{
	"profiles",
	"backups",
	"skills",
	"templates",
	"prompts",
}

// OverrideDir sets a programmatic override for the config directory.
// This takes highest priority in Dir() resolution.
func OverrideDir(dir string) {
	overrideDir = dir
}

// Dir returns the hystak configuration directory.
// Priority: OverrideDir > HYSTAK_CONFIG_DIR > XDG_CONFIG_HOME/hystak > ~/.hystak.
func Dir() string {
	if overrideDir != "" {
		return overrideDir
	}
	if d := os.Getenv("HYSTAK_CONFIG_DIR"); d != "" {
		return d
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "hystak")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".hystak")
	}
	return filepath.Join(home, ".hystak")
}

// RegistryPath returns the path to registry.yaml.
func RegistryPath() string {
	return filepath.Join(Dir(), "registry.yaml")
}

// ProjectsPath returns the path to projects.yaml.
func ProjectsPath() string {
	return filepath.Join(Dir(), "projects.yaml")
}

// UserConfigPath returns the path to user.yaml.
func UserConfigPath() string {
	return filepath.Join(Dir(), "user.yaml")
}

// KeysConfigPath returns the path to keys.yaml.
func KeysConfigPath() string {
	return filepath.Join(Dir(), "keys.yaml")
}

// ProfilesDir returns the path to the profiles subdirectory.
func ProfilesDir() string {
	return filepath.Join(Dir(), "profiles")
}

// BackupsDir returns the path to the backups subdirectory.
func BackupsDir() string {
	return filepath.Join(Dir(), "backups")
}

// Subdirs returns all required subdirectories under the config root.
func Subdirs() []string {
	return subdirs
}
