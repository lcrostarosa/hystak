package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	hysterr "github.com/lcrostarosa/hystak/internal/errors"
	"gopkg.in/yaml.v3"
)

// IsolationStrategy determines how concurrent sessions are handled.
type IsolationStrategy string

const (
	IsolationNone     IsolationStrategy = "none"
	IsolationWorktree IsolationStrategy = "worktree"
	IsolationLock     IsolationStrategy = "lock"
)

// VanillaName is the reserved name for the built-in empty profile.
const VanillaName = "vanilla"

// Profile is a named loadout — a subset of available tools to enable for a session.
type Profile struct {
	Name        string            `yaml:"-"`
	Description string            `yaml:"description,omitempty"`
	MCPs        []string          `yaml:"mcps,omitempty"`
	Skills      []string          `yaml:"skills,omitempty"`
	Hooks       []string          `yaml:"hooks,omitempty"`
	Permissions []string          `yaml:"permissions,omitempty"`
	Prompts     []string          `yaml:"prompts,omitempty"`
	EnvVars     map[string]string `yaml:"env,omitempty"`
	ClaudeMD    string            `yaml:"claude_md,omitempty"`
	Isolation   IsolationStrategy `yaml:"isolation,omitempty"`
}

// IsVanilla reports whether the profile is the built-in empty profile.
func (p Profile) IsVanilla() bool {
	return p.Name == VanillaName
}

// IsEmpty reports whether the profile has no selections.
func (p Profile) IsEmpty() bool {
	return len(p.MCPs) == 0 &&
		len(p.Skills) == 0 &&
		len(p.Hooks) == 0 &&
		len(p.Permissions) == 0 &&
		len(p.Prompts) == 0 &&
		len(p.EnvVars) == 0 &&
		p.ClaudeMD == ""
}

// Vanilla returns the built-in empty profile.
func Vanilla() Profile {
	return Profile{
		Name:        VanillaName,
		Description: "Empty profile — deploys nothing",
		Isolation:   IsolationNone,
	}
}

// profileFile is the on-disk YAML structure for a profile.
type profileFile struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	MCPs        []string          `yaml:"mcps,omitempty"`
	Skills      []string          `yaml:"skills,omitempty"`
	Hooks       []string          `yaml:"hooks,omitempty"`
	Permissions []string          `yaml:"permissions,omitempty"`
	Prompts     []string          `yaml:"prompts,omitempty"`
	EnvVars     map[string]string `yaml:"env,omitempty"`
	ClaudeMD    string            `yaml:"claude_md,omitempty"`
	Isolation   IsolationStrategy `yaml:"isolation,omitempty"`
}

func toFile(p Profile) profileFile {
	return profileFile{
		Name:        p.Name,
		Description: p.Description,
		MCPs:        p.MCPs,
		Skills:      p.Skills,
		Hooks:       p.Hooks,
		Permissions: p.Permissions,
		Prompts:     p.Prompts,
		EnvVars:     p.EnvVars,
		ClaudeMD:    p.ClaudeMD,
		Isolation:   p.Isolation,
	}
}

func fromFile(f profileFile) Profile {
	return Profile{
		Name:        f.Name,
		Description: f.Description,
		MCPs:        f.MCPs,
		Skills:      f.Skills,
		Hooks:       f.Hooks,
		Permissions: f.Permissions,
		Prompts:     f.Prompts,
		EnvVars:     f.EnvVars,
		ClaudeMD:    f.ClaudeMD,
		Isolation:   f.Isolation,
	}
}

// Manager handles loading and saving profiles from the global profiles directory.
type Manager struct {
	dir string // ~/.hystak/profiles/
}

// NewManager creates a Manager rooted at the given directory.
func NewManager(dir string) *Manager {
	return &Manager{dir: dir}
}

// validate checks basic profile constraints.
func validate(p Profile) error {
	if p.Name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}
	if p.Name == VanillaName {
		return fmt.Errorf("cannot modify the built-in %q profile", VanillaName)
	}
	return nil
}

// path returns the YAML file path for a profile name.
func (m *Manager) path(name string) string {
	return filepath.Join(m.dir, name+".yaml")
}

// Save persists a profile to disk.
func (m *Manager) Save(p Profile) error {
	if err := validate(p); err != nil {
		return err
	}

	data, err := yaml.Marshal(toFile(p))
	if err != nil {
		return fmt.Errorf("marshaling profile %q: %w", p.Name, err)
	}

	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		return fmt.Errorf("creating profiles directory: %w", err)
	}

	if err := os.WriteFile(m.path(p.Name), data, 0o644); err != nil {
		return fmt.Errorf("writing profile %q: %w", p.Name, err)
	}

	return nil
}

// Get returns a profile by name. The vanilla profile is always available.
func (m *Manager) Get(name string) (*Profile, error) {
	if name == VanillaName {
		v := Vanilla()
		return &v, nil
	}

	data, err := os.ReadFile(m.path(name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, hysterr.ProfileNotFound(name)
		}
		return nil, fmt.Errorf("reading profile %q: %w", name, err)
	}

	var f profileFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing profile %q: %w", name, err)
	}

	p := fromFile(f)
	p.Name = name // ensure name matches filename
	return &p, nil
}

// Delete removes a profile from disk.
func (m *Manager) Delete(name string) error {
	if name == VanillaName {
		return fmt.Errorf("cannot delete the built-in %q profile", VanillaName)
	}

	path := m.path(name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return hysterr.ProfileNotFound(name)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("deleting profile %q: %w", name, err)
	}

	return nil
}

// List returns all profiles (including vanilla) sorted by name.
func (m *Manager) List() ([]Profile, error) {
	profiles := []Profile{Vanilla()}

	entries, err := os.ReadDir(m.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return profiles, nil
		}
		return nil, fmt.Errorf("listing profiles: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := filepath.Ext(name)
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		profileName := name[:len(name)-len(ext)]
		if profileName == VanillaName {
			continue // skip any file named vanilla.yaml
		}

		p, err := m.Get(profileName)
		if err != nil {
			continue // skip unreadable profiles
		}
		profiles = append(profiles, *p)
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})

	return profiles, nil
}

// Exists checks if a profile with the given name exists on disk.
func (m *Manager) Exists(name string) bool {
	if name == VanillaName {
		return true
	}
	_, err := os.Stat(m.path(name))
	return err == nil
}

// Export serializes a profile to YAML bytes.
func (m *Manager) Export(name string) ([]byte, error) {
	p, err := m.Get(name)
	if err != nil {
		return nil, err
	}
	data, err := yaml.Marshal(toFile(*p))
	if err != nil {
		return nil, fmt.Errorf("marshaling profile %q: %w", name, err)
	}
	return data, nil
}

// Import deserializes YAML bytes into a profile and saves it.
// Returns an error if a profile with the same name already exists.
func (m *Manager) Import(data []byte) (*Profile, error) {
	var f profileFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing profile YAML: %w", err)
	}

	p := fromFile(f)
	if p.Name == "" {
		return nil, fmt.Errorf("imported profile has no name")
	}

	if m.Exists(p.Name) {
		return nil, hysterr.ProfileAlreadyExists(p.Name)
	}

	if err := m.Save(p); err != nil {
		return nil, err
	}

	return &p, nil
}

// ImportAs deserializes YAML bytes into a profile, renames it, and saves.
// This allows importing a profile under a different name to avoid conflicts.
func (m *Manager) ImportAs(data []byte, newName string) (*Profile, error) {
	var f profileFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing profile YAML: %w", err)
	}

	p := fromFile(f)
	p.Name = newName

	if err := validate(p); err != nil {
		return nil, err
	}

	if m.Exists(p.Name) {
		return nil, hysterr.ProfileAlreadyExists(p.Name)
	}

	if err := m.Save(p); err != nil {
		return nil, err
	}

	return &p, nil
}
