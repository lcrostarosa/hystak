package deploy

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hystak/hystak/internal/config"
)

// managedSentinel marks a CLAUDE.md file as managed by hystak.
const managedSentinel = "<!-- managed by hystak -->"

// Compile-time interface check.
var _ ResourceDeployer = (*ClaudeMDDeployer)(nil)

// ClaudeMDDeployer deploys CLAUDE.md to the project root (S-045).
// Template only = symlink. Template + prompts = composed file with sentinel.
// User-owned CLAUDE.md (no sentinel) is never overwritten.
type ClaudeMDDeployer struct{}

func (d *ClaudeMDDeployer) Kind() ResourceDeployerKind {
	return ResourceDeployerClaudeMD
}

// Sync deploys CLAUDE.md to the project root.
func (d *ClaudeMDDeployer) Sync(projectPath string, cfg DeployConfig) error {
	path := filepath.Join(projectPath, "CLAUDE.md")

	// No template and no prompts: remove managed file if present
	if cfg.TemplateSource == "" && len(cfg.PromptSources) == 0 {
		return d.removeIfManaged(path)
	}

	// Template only, no prompts: symlink mode
	if cfg.TemplateSource != "" && len(cfg.PromptSources) == 0 {
		return d.deploySymlink(path, cfg.TemplateSource)
	}

	// Template + prompts (or prompts only): composed mode
	return d.deployComposed(path, cfg.TemplateSource, cfg.PromptSources)
}

func (d *ClaudeMDDeployer) deploySymlink(path, target string) error {
	info, err := os.Lstat(path)
	switch {
	case err == nil:
		if info.Mode()&os.ModeSymlink != 0 {
			// Existing symlink — remove and replace
			if err := os.Remove(path); err != nil {
				return err
			}
		} else {
			// Regular file — check if managed
			if !d.isManaged(path) {
				return nil // user-owned, don't overwrite
			}
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	case errors.Is(err, fs.ErrNotExist):
		// will create
	default:
		return err
	}

	if err := validateSymlinkTarget(target); err != nil {
		return err
	}
	return os.Symlink(target, path)
}

func (d *ClaudeMDDeployer) deployComposed(path, templateSource string, promptSources []string) error {
	// Check if user-owned
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			// Remove existing symlink to replace with composed file
			if err := os.Remove(path); err != nil {
				return err
			}
		} else if !d.isManaged(path) {
			return nil // user-owned, don't overwrite
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	var b strings.Builder
	b.WriteString(managedSentinel + "\n\n")

	// Include template content
	if templateSource != "" {
		content, err := os.ReadFile(templateSource)
		if err != nil {
			return fmt.Errorf("reading template %q: %w", templateSource, err)
		}
		b.Write(content)
		b.WriteString("\n\n")
	}

	// Include prompt fragments
	for _, src := range promptSources {
		content, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("reading prompt %q: %w", src, err)
		}
		b.WriteString("---\n\n")
		b.Write(content)
		b.WriteString("\n\n")
	}

	return config.AtomicWrite(path, []byte(b.String()), 0o644)
}

func (d *ClaudeMDDeployer) removeIfManaged(path string) error {
	info, err := os.Lstat(path)
	switch {
	case err == nil:
		if info.Mode()&os.ModeSymlink != 0 {
			return os.Remove(path) // symlink = managed
		}
		if d.isManaged(path) {
			return os.Remove(path) // has sentinel = managed
		}
		return nil // user-owned, leave it
	case errors.Is(err, fs.ErrNotExist):
		return nil
	default:
		return err
	}
}

// isManaged checks whether a CLAUDE.md file contains the managed sentinel.
func (d *ClaudeMDDeployer) isManaged(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.HasPrefix(string(data), managedSentinel)
}

// Preflight checks for user-owned CLAUDE.md (no sentinel, not a symlink).
func (d *ClaudeMDDeployer) Preflight(projectPath string, cfg DeployConfig) []PreflightConflict {
	if cfg.TemplateSource == "" && len(cfg.PromptSources) == 0 {
		return nil
	}
	path := filepath.Join(projectPath, "CLAUDE.md")
	info, err := os.Lstat(path)
	if err != nil {
		return nil
	}
	// Symlinks are not conflicts (S-048)
	if info.Mode()&os.ModeSymlink != 0 {
		return nil
	}
	// Check for sentinel
	if d.isManaged(path) {
		return nil
	}
	return []PreflightConflict{{
		Path:    path,
		Kind:    ResourceDeployerClaudeMD,
		Message: "CLAUDE.md exists and is not managed by hystak (no sentinel marker)",
	}}
}

// ReadDeployed reads the current CLAUDE.md state.
func (d *ClaudeMDDeployer) ReadDeployed(projectPath string) (DeployConfig, error) {
	path := filepath.Join(projectPath, "CLAUDE.md")
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return DeployConfig{}, nil
		}
		return DeployConfig{}, err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			return DeployConfig{}, err
		}
		return DeployConfig{TemplateSource: target}, nil
	}

	// Composed file — just report that it exists
	if d.isManaged(path) {
		return DeployConfig{TemplateSource: "(composed)"}, nil
	}

	return DeployConfig{}, nil
}
