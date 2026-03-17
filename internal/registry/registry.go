package registry

import (
	"fmt"
	"os"
	"sort"

	"github.com/lcrostarosa/hystak/internal/model"
	"gopkg.in/yaml.v3"
)

// registryFile is the on-disk YAML structure.
type registryFile struct {
	Servers map[string]model.ServerDef `yaml:"servers"`
	Tags    map[string][]string        `yaml:"tags,omitempty"`
}

// Registry manages the central server catalog and tag groups.
type Registry struct {
	Servers map[string]model.ServerDef
	Tags    map[string][]string
}

// Load reads and parses a registry.yaml file.
// Returns an empty registry if the file is empty or does not exist.
func Load(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return empty(), nil
		}
		return nil, fmt.Errorf("reading registry: %w", err)
	}

	if len(data) == 0 {
		return empty(), nil
	}

	var f registryFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing registry: %w", err)
	}

	r := &Registry{
		Servers: f.Servers,
		Tags:    f.Tags,
	}
	if r.Servers == nil {
		r.Servers = make(map[string]model.ServerDef)
	}
	if r.Tags == nil {
		r.Tags = make(map[string][]string)
	}

	// Populate Name field from map key.
	for name, srv := range r.Servers {
		srv.Name = name
		r.Servers[name] = srv
	}

	return r, nil
}

// Save writes the registry to a YAML file.
func (r *Registry) Save(path string) error {
	f := registryFile{
		Servers: r.Servers,
		Tags:    r.Tags,
	}

	data, err := yaml.Marshal(&f)
	if err != nil {
		return fmt.Errorf("marshaling registry: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing registry: %w", err)
	}

	return nil
}

// Add adds a server to the registry. Returns an error if the name already exists.
func (r *Registry) Add(server model.ServerDef) error {
	if _, exists := r.Servers[server.Name]; exists {
		return fmt.Errorf("server %q already exists", server.Name)
	}
	r.Servers[server.Name] = server
	return nil
}

// Update replaces an existing server definition. Returns an error if not found.
func (r *Registry) Update(name string, server model.ServerDef) error {
	if _, exists := r.Servers[name]; !exists {
		return fmt.Errorf("server %q not found", name)
	}
	server.Name = name
	r.Servers[name] = server
	return nil
}

// Delete removes a server from the registry.
// Returns an error if the server is referenced by any tag.
func (r *Registry) Delete(name string) error {
	if _, exists := r.Servers[name]; !exists {
		return fmt.Errorf("server %q not found", name)
	}

	// Check tag references.
	for tag, servers := range r.Tags {
		for _, s := range servers {
			if s == name {
				return fmt.Errorf("cannot delete server %q: referenced by tag %q", name, tag)
			}
		}
	}

	delete(r.Servers, name)
	return nil
}

// Get returns a server by name.
func (r *Registry) Get(name string) (model.ServerDef, bool) {
	srv, ok := r.Servers[name]
	return srv, ok
}

// List returns all servers sorted by name.
func (r *Registry) List() []model.ServerDef {
	servers := make([]model.ServerDef, 0, len(r.Servers))
	for _, srv := range r.Servers {
		servers = append(servers, srv)
	}
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Name < servers[j].Name
	})
	return servers
}

// ExpandTag returns the server names for a tag.
// Returns an error if the tag is unknown or references a missing server.
func (r *Registry) ExpandTag(tag string) ([]string, error) {
	servers, ok := r.Tags[tag]
	if !ok {
		return nil, fmt.Errorf("tag %q not found", tag)
	}

	for _, name := range servers {
		if _, exists := r.Servers[name]; !exists {
			return nil, fmt.Errorf("tag %q references missing server %q", tag, name)
		}
	}

	return servers, nil
}

// AddTag creates a new tag with the given server names.
// Returns an error if the tag already exists.
func (r *Registry) AddTag(name string, servers []string) error {
	if _, exists := r.Tags[name]; exists {
		return fmt.Errorf("tag %q already exists", name)
	}
	r.Tags[name] = servers
	return nil
}

// RemoveTag deletes a tag. Returns an error if the tag does not exist.
func (r *Registry) RemoveTag(name string) error {
	if _, exists := r.Tags[name]; !exists {
		return fmt.Errorf("tag %q not found", name)
	}
	delete(r.Tags, name)
	return nil
}

// UpdateTag replaces the server list for an existing tag.
// Returns an error if the tag does not exist.
func (r *Registry) UpdateTag(name string, servers []string) error {
	if _, exists := r.Tags[name]; !exists {
		return fmt.Errorf("tag %q not found", name)
	}
	r.Tags[name] = servers
	return nil
}

func empty() *Registry {
	return &Registry{
		Servers: make(map[string]model.ServerDef),
		Tags:    make(map[string][]string),
	}
}
