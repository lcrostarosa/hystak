package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// EnsureConfigDir creates the hystak config directory and all required
// subdirectories if they do not exist. Returns the config directory path.
func EnsureConfigDir() (string, error) {
	dir := Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	for _, sub := range subdirs {
		subPath := filepath.Join(dir, sub)
		if err := os.MkdirAll(subPath, 0o755); err != nil {
			return "", err
		}
	}
	return dir, nil
}

// IsFirstRun reports whether the config directory has never been initialized.
// Returns true if the directory does not exist, false if it does, or an error
// for unexpected filesystem failures.
func IsFirstRun() (bool, error) {
	_, err := os.Stat(Dir())
	switch {
	case err == nil:
		return false, nil
	case errors.Is(err, fs.ErrNotExist):
		return true, nil
	default:
		return false, err
	}
}
