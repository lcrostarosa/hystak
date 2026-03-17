package config

import (
	"os"
	"path/filepath"
)

const appName = "hystak"

// ConfigDir returns the hystak configuration directory.
// Respects XDG_CONFIG_HOME; defaults to ~/.config/hystak/.
func ConfigDir() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			// Fallback; should not happen in practice.
			home = "."
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, appName)
}

// RegistryPath returns the path to registry.yaml.
func RegistryPath() string {
	return filepath.Join(ConfigDir(), "registry.yaml")
}

// ProjectsPath returns the path to projects.yaml.
func ProjectsPath() string {
	return filepath.Join(ConfigDir(), "projects.yaml")
}

// EnsureConfigDir creates the config directory and empty config files if they
// do not already exist.
func EnsureConfigDir() error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	for _, name := range []string{"registry.yaml", "projects.yaml"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			if err := os.WriteFile(p, nil, 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}
