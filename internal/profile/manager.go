package profile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hystak/hystak/internal/config"
	hysterr "github.com/hystak/hystak/internal/errors"
	"github.com/hystak/hystak/internal/model"
	"gopkg.in/yaml.v3"
)

// emptyProfileName is the reserved name for the built-in empty profile.
const emptyProfileName = "empty"

// EmptyProfile returns the built-in empty profile (S-029).
// It deploys zero configuration — useful for clean Claude Code launches.
func EmptyProfile() model.ProjectProfile {
	return model.ProjectProfile{
		Name:        emptyProfileName,
		Description: "Clean Claude Code launch — no MCPs, skills, hooks, or permissions",
		Scope:       "global",
		MCPs:        []model.MCPAssignment{},
		Skills:      []string{},
		Hooks:       []string{},
		Permissions: []string{},
		Prompts:     []string{},
		Env:         map[string]string{},
		Isolation:   model.IsolationNone,
	}
}

// Manager handles profile CRUD in the profiles/ directory.
type Manager struct {
	dir string
}

// NewManager creates a profile manager for the given profiles directory.
func NewManager(dir string) *Manager {
	return &Manager{dir: dir}
}

// NewDefaultManager creates a profile manager using the default config directory.
func NewDefaultManager() *Manager {
	return NewManager(config.ProfilesDir())
}

// Save writes a profile to disk as <dir>/<name>.yaml.
func (m *Manager) Save(p model.ProjectProfile) error {
	if p.Name == "" {
		return &hysterr.ValidationError{Field: "name", Message: "profile name must not be empty"}
	}

	path, err := m.validateProfilePath(p.Name)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshaling profile %q: %w", p.Name, err)
	}

	return config.AtomicWrite(path, data, 0o644)
}

// Load reads a profile from disk by name.
// Returns the built-in empty profile if name is "empty" and no file exists.
func (m *Manager) Load(name string) (model.ProjectProfile, error) {
	path, err := m.validateProfilePath(name)
	if err != nil {
		return model.ProjectProfile{}, err
	}

	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		// file exists
	case errors.Is(err, fs.ErrNotExist):
		if name == emptyProfileName {
			return EmptyProfile(), nil
		}
		return model.ProjectProfile{}, &hysterr.ResourceNotFound{Kind: "profile", Name: name}
	default:
		return model.ProjectProfile{}, err
	}

	var p model.ProjectProfile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return model.ProjectProfile{}, &hysterr.ConfigParseError{Path: path, Err: err}
	}
	// Ensure name matches filename (in case file was manually edited)
	p.Name = name
	return p, nil
}

// Delete removes a profile from disk.
func (m *Manager) Delete(name string) error {
	path, err := m.validateProfilePath(name)
	if err != nil {
		return err
	}

	_, statErr := os.Stat(path)
	switch {
	case statErr == nil:
		return os.Remove(path)
	case errors.Is(statErr, fs.ErrNotExist):
		return &hysterr.ResourceNotFound{Kind: "profile", Name: name}
	default:
		return statErr
	}
}

// List returns all profile names found in the profiles directory, sorted.
// Always includes "empty" even if no file exists on disk.
func (m *Manager) List() ([]string, error) {
	entries, err := os.ReadDir(m.dir)
	switch {
	case err == nil:
		// dir exists
	case errors.Is(err, fs.ErrNotExist):
		return []string{emptyProfileName}, nil
	default:
		return nil, err
	}

	seen := make(map[string]bool)
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) == ".yaml" {
			n := name[:len(name)-5]
			names = append(names, n)
			seen[n] = true
		}
	}
	if !seen[emptyProfileName] {
		names = append(names, emptyProfileName)
	}
	sort.Strings(names)
	return names, nil
}

// Exists reports whether a profile with the given name exists on disk
// or is the built-in empty profile. Uses three-way stat.
func (m *Manager) Exists(name string) (bool, error) {
	if name == emptyProfileName {
		return true, nil
	}
	path, err := m.validateProfilePath(name)
	if err != nil {
		return false, err
	}
	_, statErr := os.Stat(path)
	switch {
	case statErr == nil:
		return true, nil
	case errors.Is(statErr, fs.ErrNotExist):
		return false, nil
	default:
		return false, statErr
	}
}

// LoadAll reads all profiles from the directory (including built-in empty).
func (m *Manager) LoadAll() ([]model.ProjectProfile, error) {
	names, err := m.List()
	if err != nil {
		return nil, err
	}

	profiles := make([]model.ProjectProfile, 0, len(names))
	for _, name := range names {
		p, err := m.Load(name)
		if err != nil {
			return nil, fmt.Errorf("loading profile %q: %w", name, err)
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (m *Manager) profilePath(name string) string {
	return filepath.Join(m.dir, name+".yaml")
}

// validateProfilePath constructs the profile path and verifies it stays
// within the profiles directory. Returns a ValidationError for path traversal.
func (m *Manager) validateProfilePath(name string) (string, error) {
	p := m.profilePath(name)
	rel, err := filepath.Rel(m.dir, p)
	if err != nil || strings.HasPrefix(rel, "..") || strings.Contains(rel, string(filepath.Separator)+"..") {
		return "", &hysterr.ValidationError{Field: "name", Message: "profile name contains path traversal"}
	}
	return p, nil
}
