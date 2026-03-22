package project

import (
	"errors"
	"io/fs"
	"os"

	"github.com/hystak/hystak/internal/config"
	hysterr "github.com/hystak/hystak/internal/errors"
	"github.com/hystak/hystak/internal/model"
	"github.com/hystak/hystak/internal/registry"
	"gopkg.in/yaml.v3"
)

// Store manages project registrations persisted to projects.yaml.
// It wraps a generic registry.Store for CRUD and adds project-specific methods.
type Store struct {
	inner *registry.Store[model.Project, *model.Project]
}

// NewStore creates an empty project store.
func NewStore() *Store {
	return &Store{
		inner: registry.NewStore[model.Project, *model.Project]("project"),
	}
}

// projectsYAML is the on-disk representation of projects.yaml.
type projectsYAML struct {
	Projects map[string]projectYAML `yaml:"projects,omitempty"`
}

// projectYAML is a single project entry in YAML (name is the map key).
type projectYAML struct {
	Path          string   `yaml:"path"`
	ActiveProfile string   `yaml:"active_profile,omitempty"`
	ManagedMCPs   []string `yaml:"managed_mcps,omitempty"`
}

// Add registers a new project. Validates that path is non-empty.
func (s *Store) Add(p model.Project) error {
	if p.Path == "" {
		return &hysterr.ValidationError{Field: "path", Message: "project path must not be empty"}
	}
	return s.inner.Add(p)
}

// Get retrieves a project by name.
func (s *Store) Get(name string) (model.Project, bool) {
	return s.inner.Get(name)
}

// Update replaces an existing project.
func (s *Store) Update(p model.Project) error {
	return s.inner.Update(p)
}

// Delete removes a project by name.
func (s *Store) Delete(name string) error {
	return s.inner.Delete(name)
}

// List returns all projects sorted by name.
func (s *Store) List() []model.Project {
	return s.inner.List()
}

// Len returns the number of registered projects.
func (s *Store) Len() int {
	return s.inner.Len()
}

// SetActiveProfile sets the active profile for a project.
func (s *Store) SetActiveProfile(projectName, profileName string) error {
	p, exists := s.inner.Get(projectName)
	if !exists {
		return &hysterr.ProjectNotFound{Name: projectName}
	}
	p.ActiveProfile = profileName
	return s.inner.Update(p)
}

// SetManagedMCPs updates the tracked managed MCPs for a project.
func (s *Store) SetManagedMCPs(projectName string, mcps []string) error {
	p, exists := s.inner.Get(projectName)
	if !exists {
		return &hysterr.ProjectNotFound{Name: projectName}
	}
	cp := make([]string, len(mcps))
	copy(cp, mcps)
	p.ManagedMCPs = cp
	return s.inner.Update(p)
}

// FindByPath returns the project registered at the given path, if any.
func (s *Store) FindByPath(path string) (model.Project, bool) {
	for _, p := range s.inner.List() {
		if p.Path == path {
			return p, true
		}
	}
	return model.Project{}, false
}

// Load reads projects.yaml from the given path.
// Returns an empty store if the file does not exist.
func Load(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		// file exists
	case errors.Is(err, fs.ErrNotExist):
		return NewStore(), nil
	default:
		return nil, err
	}

	var raw projectsYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, &hysterr.ConfigParseError{Path: path, Err: err}
	}

	store := NewStore()
	items := make(map[string]model.Project, len(raw.Projects))
	for name, py := range raw.Projects {
		items[name] = model.Project{
			Path:          py.Path,
			ActiveProfile: py.ActiveProfile,
			ManagedMCPs:   py.ManagedMCPs,
		}
	}
	store.inner.SetItems(items)

	return store, nil
}

// Save writes projects.yaml atomically to the given path.
func (s *Store) Save(path string) error {
	raw := projectsYAML{}

	if s.Len() > 0 {
		items := s.inner.Items()
		raw.Projects = make(map[string]projectYAML, len(items))
		for name, p := range items {
			raw.Projects[name] = projectYAML{
				Path:          p.Path,
				ActiveProfile: p.ActiveProfile,
				ManagedMCPs:   p.ManagedMCPs,
			}
		}
	}

	data, err := yaml.Marshal(raw)
	if err != nil {
		return err
	}

	return config.AtomicWrite(path, data, 0o644)
}

// LoadDefault loads projects.yaml from the default config directory.
func LoadDefault() (*Store, error) {
	return Load(config.ProjectsPath())
}

// SaveDefault saves projects.yaml to the default config directory.
func (s *Store) SaveDefault() error {
	return s.Save(config.ProjectsPath())
}
