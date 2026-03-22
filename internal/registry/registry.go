package registry

import (
	"fmt"
	"os"

	hysterr "github.com/lcrostarosa/hystak/internal/errors"
	"github.com/lcrostarosa/hystak/internal/model"
	"gopkg.in/yaml.v3"
)

// registryFile is the on-disk YAML structure (unchanged for backward compatibility).
type registryFile struct {
	Servers     map[string]model.ServerDef     `yaml:"servers"`
	Skills      map[string]model.SkillDef      `yaml:"skills,omitempty"`
	Hooks       map[string]model.HookDef       `yaml:"hooks,omitempty"`
	Permissions map[string]model.PermissionRule `yaml:"permissions,omitempty"`
	Templates   map[string]model.TemplateDef   `yaml:"templates,omitempty"`
	Prompts     map[string]model.PromptDef     `yaml:"prompts,omitempty"`
	Tags        map[string][]string            `yaml:"tags,omitempty"`
}

// Registry manages the central server catalog, skills, hooks, permissions,
// templates, prompts, and tag groups.
type Registry struct {
	Servers     *Store[model.ServerDef, *model.ServerDef]
	Skills      *Store[model.SkillDef, *model.SkillDef]
	Hooks       *Store[model.HookDef, *model.HookDef]
	Permissions *Store[model.PermissionRule, *model.PermissionRule]
	Templates   *Store[model.TemplateDef, *model.TemplateDef]
	Prompts     *Store[model.PromptDef, *model.PromptDef]
	Tags        map[string][]string
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

	r := empty()
	r.Servers.SetItems(f.Servers)
	r.Skills.SetItems(f.Skills)
	r.Hooks.SetItems(f.Hooks)
	r.Permissions.SetItems(f.Permissions)
	r.Templates.SetItems(f.Templates)
	r.Prompts.SetItems(f.Prompts)
	if f.Tags != nil {
		r.Tags = f.Tags
	}

	return r, nil
}

// Save writes the registry to a YAML file.
func (r *Registry) Save(path string) error {
	f := registryFile{
		Servers:     r.Servers.Items(),
		Skills:      r.Skills.Items(),
		Hooks:       r.Hooks.Items(),
		Permissions: r.Permissions.Items(),
		Templates:   r.Templates.Items(),
		Prompts:     r.Prompts.Items(),
		Tags:        r.Tags,
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

// DeleteServer removes a server from the registry.
// Returns an error if the server is referenced by any tag.
// This method stays on Registry (not on Store) because it cross-references Tags.
func (r *Registry) DeleteServer(name string) error {
	if _, exists := r.Servers.Get(name); !exists {
		return hysterr.ServerNotFound(name)
	}

	for tag, servers := range r.Tags {
		for _, s := range servers {
			if s == name {
				return hysterr.ServerReferenced(name, tag)
			}
		}
	}

	return r.Servers.Delete(name)
}

// ExpandTag returns the server names for a tag.
// Returns an error if the tag is unknown or references a missing server.
func (r *Registry) ExpandTag(tag string) ([]string, error) {
	servers, ok := r.Tags[tag]
	if !ok {
		return nil, hysterr.TagNotFound(tag)
	}

	for _, name := range servers {
		if _, exists := r.Servers.Get(name); !exists {
			return nil, fmt.Errorf("tag %q references missing server %q", tag, name)
		}
	}

	return servers, nil
}

// AddTag creates a new tag with the given server names.
func (r *Registry) AddTag(name string, servers []string) error {
	if _, exists := r.Tags[name]; exists {
		return hysterr.TagAlreadyExists(name)
	}
	r.Tags[name] = servers
	return nil
}

// RemoveTag deletes a tag.
func (r *Registry) RemoveTag(name string) error {
	if _, exists := r.Tags[name]; !exists {
		return hysterr.TagNotFound(name)
	}
	delete(r.Tags, name)
	return nil
}

// UpdateTag replaces the server list for an existing tag.
func (r *Registry) UpdateTag(name string, servers []string) error {
	if _, exists := r.Tags[name]; !exists {
		return hysterr.TagNotFound(name)
	}
	r.Tags[name] = servers
	return nil
}

func empty() *Registry {
	return &Registry{
		Servers:     NewStore[model.ServerDef, *model.ServerDef]("server"),
		Skills:      NewStore[model.SkillDef, *model.SkillDef]("skill"),
		Hooks:       NewStore[model.HookDef, *model.HookDef]("hook"),
		Permissions: NewStore[model.PermissionRule, *model.PermissionRule]("permission"),
		Templates:   NewStore[model.TemplateDef, *model.TemplateDef]("template"),
		Prompts: NewStore[model.PromptDef, *model.PromptDef]("prompt").WithSort(func(a, b model.PromptDef) bool {
			if a.Order != b.Order {
				return a.Order < b.Order
			}
			return a.Name < b.Name
		}),
		Tags: make(map[string][]string),
	}
}
