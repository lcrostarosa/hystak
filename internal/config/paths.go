package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const appName = "hystak"

// ConfigDir returns the hystak configuration directory (~/.hystak/).
// If HYSTAK_CONFIG_DIR is set, it is used instead.
func ConfigDir() string {
	if dir := os.Getenv("HYSTAK_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".hystak")
}

// LegacyConfigDir returns the old XDG-based config directory (~/.config/hystak/).
func LegacyConfigDir() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
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

// subdirectories that EnsureConfigDir creates inside the config root.
var subdirs = []string{"profiles", "skills", "templates", "backups"}

// EnsureConfigDir creates the config directory, required subdirectories,
// and empty config files if they do not already exist.
func EnsureConfigDir() error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	for _, sub := range subdirs {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			return err
		}
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

// Migrate checks for the legacy config directory (~/.config/hystak/) and copies
// registry.yaml and projects.yaml to the new location (~/.hystak/) if the new
// directory does not already contain them. Returns a non-empty warning message
// if migration occurred, so the caller can display it to the user.
func Migrate() (warning string, err error) {
	newDir := ConfigDir()
	oldDir := LegacyConfigDir()

	// Nothing to migrate if old dir doesn't exist.
	if _, err := os.Stat(oldDir); os.IsNotExist(err) {
		return "", nil
	}

	// If new dir already has a registry.yaml, assume migration already happened.
	if _, err := os.Stat(filepath.Join(newDir, "registry.yaml")); err == nil {
		return "", nil
	}

	// Ensure new dir structure exists.
	if err := EnsureConfigDir(); err != nil {
		return "", fmt.Errorf("creating new config directory: %w", err)
	}

	// Copy config files from old to new.
	migrated := false
	for _, name := range []string{"registry.yaml", "projects.yaml"} {
		src := filepath.Join(oldDir, name)
		if _, err := os.Stat(src); os.IsNotExist(err) {
			continue
		}
		dst := filepath.Join(newDir, name)
		if err := copyFile(src, dst); err != nil {
			return "", fmt.Errorf("migrating %s: %w", name, err)
		}
		migrated = true
	}

	if migrated {
		return fmt.Sprintf("Migrated configs from %s to %s. The old directory can be removed.", oldDir, newDir), nil
	}
	return "", nil
}

// copyFile copies src to dst, preserving file permissions.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
