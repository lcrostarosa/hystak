package registry

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/hystak/hystak/internal/config"
	hysterr "github.com/hystak/hystak/internal/errors"
	"github.com/hystak/hystak/internal/model"
	"gopkg.in/yaml.v3"
)

// Registry holds all registered resources persisted to registry.yaml.
type Registry struct {
	Servers     *Store[model.ServerDef, *model.ServerDef]
	Skills      *Store[model.SkillDef, *model.SkillDef]
	Hooks       *Store[model.HookDef, *model.HookDef]
	Permissions *Store[model.PermissionRule, *model.PermissionRule]
	Templates   *Store[model.TemplateDef, *model.TemplateDef]
	Prompts     *Store[model.PromptDef, *model.PromptDef]
	tags        map[string][]string
}

// New creates a new Registry with empty stores.
func New() *Registry {
	return &Registry{
		Servers:     NewStore[model.ServerDef, *model.ServerDef]("server"),
		Skills:      NewStore[model.SkillDef, *model.SkillDef]("skill"),
		Hooks:       NewStore[model.HookDef, *model.HookDef]("hook"),
		Permissions: NewStore[model.PermissionRule, *model.PermissionRule]("permission"),
		Templates:   NewStore[model.TemplateDef, *model.TemplateDef]("template"),
		Prompts: NewStore[model.PromptDef, *model.PromptDef]("prompt").WithSort(
			func(a, b model.PromptDef) int { return a.Order - b.Order },
		),
		tags: make(map[string][]string),
	}
}

// IsEmpty reports whether the registry has no resources at all.
func (r *Registry) IsEmpty() bool {
	return r.Servers.Len() == 0 &&
		r.Skills.Len() == 0 &&
		r.Hooks.Len() == 0 &&
		r.Permissions.Len() == 0 &&
		r.Templates.Len() == 0 &&
		r.Prompts.Len() == 0 &&
		!r.HasTags()
}

// AddTag adds a named tag with the given members.
func (r *Registry) AddTag(name string, members []string) error {
	if name == "" {
		return fmt.Errorf("tag name must not be empty")
	}
	if _, exists := r.tags[name]; exists {
		return &hysterr.AlreadyExists{Kind: "tag", Name: name}
	}
	cp := make([]string, len(members))
	copy(cp, members)
	r.tags[name] = cp
	return nil
}

// GetTag retrieves a tag's members by name.
func (r *Registry) GetTag(name string) ([]string, bool) {
	members, ok := r.tags[name]
	if !ok {
		return nil, false
	}
	cp := make([]string, len(members))
	copy(cp, members)
	return cp, true
}

// DeleteTag removes a tag by name.
func (r *Registry) DeleteTag(name string) error {
	if _, exists := r.tags[name]; !exists {
		return &hysterr.ResourceNotFound{Kind: "tag", Name: name}
	}
	delete(r.tags, name)
	return nil
}

// ListTags returns a copy of all tags.
func (r *Registry) ListTags() map[string][]string {
	cp := make(map[string][]string, len(r.tags))
	for k, v := range r.tags {
		members := make([]string, len(v))
		copy(members, v)
		cp[k] = members
	}
	return cp
}

// HasTags reports whether the registry has any tags.
func (r *Registry) HasTags() bool {
	return len(r.tags) > 0
}

// registryYAML is the on-disk YAML representation of registry.yaml.
// Resources are stored as maps keyed by name.
type registryYAML struct {
	MCPs        map[string]model.ServerDef      `yaml:"mcps,omitempty"`
	Skills      map[string]model.SkillDef       `yaml:"skills,omitempty"`
	Hooks       map[string]model.HookDef        `yaml:"hooks,omitempty"`
	Permissions map[string]model.PermissionRule `yaml:"permissions,omitempty"`
	Templates   map[string]model.TemplateDef    `yaml:"templates,omitempty"`
	Prompts     map[string]model.PromptDef      `yaml:"prompts,omitempty"`
	Tags        map[string]tagYAML              `yaml:"tags,omitempty"`
}

// tagYAML is the on-disk representation of a tag.
type tagYAML struct {
	Members []string `yaml:"members"`
}

// Load reads registry.yaml from the given path.
// Returns an empty registry if the file does not exist.
func Load(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		// file exists
	case errors.Is(err, fs.ErrNotExist):
		return New(), nil
	default:
		return nil, err
	}

	var raw registryYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, &hysterr.ConfigParseError{Path: path, Err: err}
	}

	reg := New()
	reg.Servers.SetItems(raw.MCPs)
	reg.Skills.SetItems(raw.Skills)
	reg.Hooks.SetItems(raw.Hooks)
	reg.Permissions.SetItems(raw.Permissions)
	reg.Templates.SetItems(raw.Templates)
	reg.Prompts.SetItems(raw.Prompts)

	for name, tag := range raw.Tags {
		reg.tags[name] = tag.Members
	}

	return reg, nil
}

// Save writes the registry atomically to the given path.
func (r *Registry) Save(path string) error {
	raw := registryYAML{
		MCPs:        r.Servers.Items(),
		Skills:      r.Skills.Items(),
		Hooks:       r.Hooks.Items(),
		Permissions: r.Permissions.Items(),
		Templates:   r.Templates.Items(),
		Prompts:     r.Prompts.Items(),
	}

	if r.HasTags() {
		raw.Tags = make(map[string]tagYAML, len(r.tags))
		for name, members := range r.tags {
			raw.Tags[name] = tagYAML{Members: members}
		}
	}

	data, err := yaml.Marshal(raw)
	if err != nil {
		return err
	}

	return config.AtomicWrite(path, data, 0o644)
}

// LoadDefault loads registry.yaml from the default config directory path.
func LoadDefault() (*Registry, error) {
	return Load(config.RegistryPath())
}

// SaveDefault saves registry.yaml to the default config directory path.
func (r *Registry) SaveDefault() error {
	return r.Save(config.RegistryPath())
}
