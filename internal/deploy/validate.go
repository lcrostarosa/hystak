package deploy

import (
	"fmt"
	"path/filepath"
	"strings"
)

// validateSymlinkTarget checks that a symlink target is an absolute path
// without path traversal components. Prevents ../../../etc/passwd style
// attacks while allowing any legitimate absolute path (CS-19).
func validateSymlinkTarget(target string) error {
	if !filepath.IsAbs(target) {
		return fmt.Errorf("symlink target must be absolute path, got %q", target)
	}
	cleaned := filepath.Clean(target)
	if cleaned != target && strings.Contains(target, "..") {
		return fmt.Errorf("symlink target %q contains path traversal", target)
	}
	return nil
}
