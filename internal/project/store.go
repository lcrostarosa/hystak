package project

import (
	"fmt"
	"os"
	"sort"

	"github.com/rbbydotdev/hystak/internal/model"
	"github.com/rbbydotdev/hystak/internal/registry"
	"gopkg.in/yaml.v3"
)

// storeFile is the on-disk YAML structure.
type storeFile struct {
	Projects map[string]model.Project `yaml:"projects"`
}

// Store manages project registrations and server assignments.
type Store struct {
	Projects map[string]model.Project
}

// Load reads and parses a projects.yaml file.
// Returns an empty store if the file is empty or does not exist.
func Load(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return empty(), nil
		}
		return nil, fmt.Errorf("reading projects: %w", err)
	}

	if len(data) == 0 {
		return empty(), nil
	}

	var f storeFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing projects: %w", err)
	}

	s := &Store{Projects: f.Projects}
	if s.Projects == nil {
		s.Projects = make(map[string]model.Project)
	}

	// Populate Name field from map key.
	for name, proj := range s.Projects {
		proj.Name = name
		s.Projects[name] = proj
	}

	return s, nil
}

// Save writes the store to a YAML file.
func (s *Store) Save(path string) error {
	f := storeFile{Projects: s.Projects}

	data, err := yaml.Marshal(&f)
	if err != nil {
		return fmt.Errorf("marshaling projects: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing projects: %w", err)
	}

	return nil
}

// Add adds a project to the store. Returns an error if the name already exists.
func (s *Store) Add(proj model.Project) error {
	if _, exists := s.Projects[proj.Name]; exists {
		return fmt.Errorf("project %q already exists", proj.Name)
	}
	s.Projects[proj.Name] = proj
	return nil
}

// Remove deletes a project from the store. Returns an error if not found.
func (s *Store) Remove(name string) error {
	if _, exists := s.Projects[name]; !exists {
		return fmt.Errorf("project %q not found", name)
	}
	delete(s.Projects, name)
	return nil
}

// Get returns a project by name.
func (s *Store) Get(name string) (model.Project, bool) {
	proj, ok := s.Projects[name]
	return proj, ok
}

// List returns all projects sorted by name.
func (s *Store) List() []model.Project {
	projects := make([]model.Project, 0, len(s.Projects))
	for _, proj := range s.Projects {
		projects = append(projects, proj)
	}
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})
	return projects
}

// Assign adds a bare server name to a project's mcps list.
// Returns an error if the project is not found or the server is already assigned.
func (s *Store) Assign(projectName, serverName string) error {
	proj, ok := s.Projects[projectName]
	if !ok {
		return fmt.Errorf("project %q not found", projectName)
	}

	for _, mcp := range proj.MCPs {
		if mcp.Name == serverName {
			return fmt.Errorf("server %q already assigned to project %q", serverName, projectName)
		}
	}

	proj.MCPs = append(proj.MCPs, model.MCPAssignment{Name: serverName})
	s.Projects[projectName] = proj
	return nil
}

// Unassign removes a server from a project's mcps list.
// Returns an error if the project or assignment is not found.
func (s *Store) Unassign(projectName, serverName string) error {
	proj, ok := s.Projects[projectName]
	if !ok {
		return fmt.Errorf("project %q not found", projectName)
	}

	found := false
	mcps := make([]model.MCPAssignment, 0, len(proj.MCPs))
	for _, mcp := range proj.MCPs {
		if mcp.Name == serverName {
			found = true
			continue
		}
		mcps = append(mcps, mcp)
	}

	if !found {
		return fmt.Errorf("server %q not assigned to project %q", serverName, projectName)
	}

	proj.MCPs = mcps
	s.Projects[projectName] = proj
	return nil
}

// SetOverride adds or updates the override for a server in a project's mcps list.
// If the server is not yet assigned, it is added with the override.
func (s *Store) SetOverride(projectName, serverName string, override model.ServerOverride) error {
	proj, ok := s.Projects[projectName]
	if !ok {
		return fmt.Errorf("project %q not found", projectName)
	}

	found := false
	for i, mcp := range proj.MCPs {
		if mcp.Name == serverName {
			proj.MCPs[i].Overrides = &override
			found = true
			break
		}
	}

	if !found {
		proj.MCPs = append(proj.MCPs, model.MCPAssignment{
			Name:      serverName,
			Overrides: &override,
		})
	}

	s.Projects[projectName] = proj
	return nil
}

// SetClients updates the client list for a project.
func (s *Store) SetClients(projectName string, clients []model.ClientType) error {
	proj, ok := s.Projects[projectName]
	if !ok {
		return fmt.Errorf("project %q not found", projectName)
	}
	proj.Clients = clients
	s.Projects[projectName] = proj
	return nil
}

// SetTags updates the tag list for a project.
func (s *Store) SetTags(projectName string, tags []string) error {
	proj, ok := s.Projects[projectName]
	if !ok {
		return fmt.Errorf("project %q not found", projectName)
	}
	proj.Tags = tags
	s.Projects[projectName] = proj
	return nil
}

// ResolveServers expands tags and merges overrides to produce
// the final list of ServerDefs for a project.
//
// Algorithm:
//  1. Expand all tags -> collect server names
//  2. Collect individual mcps server names
//  3. Deduplicate (union)
//  4. For each server, look up registry definition
//  5. Apply overrides (shallow merge)
//  6. Return resolved []ServerDef
func (s *Store) ResolveServers(projectName string, reg *registry.Registry) ([]model.ServerDef, error) {
	proj, ok := s.Projects[projectName]
	if !ok {
		return nil, fmt.Errorf("project %q not found", projectName)
	}

	// Build override lookup from mcps.
	overrides := make(map[string]*model.ServerOverride)
	for _, mcp := range proj.MCPs {
		if mcp.Overrides != nil {
			overrides[mcp.Name] = mcp.Overrides
		}
	}

	// Collect all server names (deduplicated, preserving order).
	seen := make(map[string]bool)
	var serverNames []string

	// 1. Expand tags.
	for _, tag := range proj.Tags {
		names, err := reg.ExpandTag(tag)
		if err != nil {
			return nil, fmt.Errorf("resolving project %q: %w", projectName, err)
		}
		for _, name := range names {
			if !seen[name] {
				seen[name] = true
				serverNames = append(serverNames, name)
			}
		}
	}

	// 2. Collect individual mcps.
	for _, mcp := range proj.MCPs {
		if !seen[mcp.Name] {
			seen[mcp.Name] = true
			serverNames = append(serverNames, mcp.Name)
		}
	}

	// 3. Resolve each server from the registry and apply overrides.
	resolved := make([]model.ServerDef, 0, len(serverNames))
	for _, name := range serverNames {
		srv, ok := reg.Get(name)
		if !ok {
			return nil, fmt.Errorf("resolving project %q: server %q not found in registry", projectName, name)
		}

		if override, hasOverride := overrides[name]; hasOverride {
			srv = applyOverride(srv, override)
		}

		resolved = append(resolved, srv)
	}

	return resolved, nil
}

func empty() *Store {
	return &Store{
		Projects: make(map[string]model.Project),
	}
}

// applyOverride shallow-merges an override onto a server definition.
//   - env: merge maps (override keys win)
//   - headers: merge maps (override keys win)
//   - args: replace entirely
//   - command, url: replace if non-nil
func applyOverride(srv model.ServerDef, override *model.ServerOverride) model.ServerDef {
	if override.Command != nil {
		srv.Command = *override.Command
	}

	if override.URL != nil {
		srv.URL = *override.URL
	}

	if override.Args != nil {
		srv.Args = override.Args
	}

	if override.Env != nil {
		if srv.Env == nil {
			srv.Env = make(map[string]string)
		}
		merged := make(map[string]string, len(srv.Env))
		for k, v := range srv.Env {
			merged[k] = v
		}
		for k, v := range override.Env {
			merged[k] = v
		}
		srv.Env = merged
	}

	if override.Headers != nil {
		if srv.Headers == nil {
			srv.Headers = make(map[string]string)
		}
		merged := make(map[string]string, len(srv.Headers))
		for k, v := range srv.Headers {
			merged[k] = v
		}
		for k, v := range override.Headers {
			merged[k] = v
		}
		srv.Headers = merged
	}

	return srv
}
