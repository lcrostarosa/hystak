package registry

import (
	"fmt"
	"os"
	"sort"

	hysterr "github.com/lcrostarosa/hystak/internal/errors"
	"github.com/lcrostarosa/hystak/internal/model"
	"gopkg.in/yaml.v3"
)

// registryFile is the on-disk YAML structure.
type registryFile struct {
	Servers     map[string]model.ServerDef     `yaml:"servers"`
	Skills      map[string]model.SkillDef      `yaml:"skills,omitempty"`
	Hooks       map[string]model.HookDef       `yaml:"hooks,omitempty"`
	Permissions map[string]model.PermissionRule `yaml:"permissions,omitempty"`
	Templates   map[string]model.TemplateDef   `yaml:"templates,omitempty"`
	Tags        map[string][]string            `yaml:"tags,omitempty"`
}

// Registry manages the central server catalog, skills, hooks, permissions,
// templates, and tag groups.
type Registry struct {
	Servers     map[string]model.ServerDef
	Skills      map[string]model.SkillDef
	Hooks       map[string]model.HookDef
	Permissions map[string]model.PermissionRule
	Templates   map[string]model.TemplateDef
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

	r := &Registry{
		Servers:     f.Servers,
		Skills:      f.Skills,
		Hooks:       f.Hooks,
		Permissions: f.Permissions,
		Templates:   f.Templates,
		Tags:        f.Tags,
	}
	if r.Servers == nil {
		r.Servers = make(map[string]model.ServerDef)
	}
	if r.Skills == nil {
		r.Skills = make(map[string]model.SkillDef)
	}
	if r.Hooks == nil {
		r.Hooks = make(map[string]model.HookDef)
	}
	if r.Permissions == nil {
		r.Permissions = make(map[string]model.PermissionRule)
	}
	if r.Templates == nil {
		r.Templates = make(map[string]model.TemplateDef)
	}
	if r.Tags == nil {
		r.Tags = make(map[string][]string)
	}

	// Populate Name field from map key.
	for name, srv := range r.Servers {
		srv.Name = name
		r.Servers[name] = srv
	}
	for name, skill := range r.Skills {
		skill.Name = name
		r.Skills[name] = skill
	}
	for name, hook := range r.Hooks {
		hook.Name = name
		r.Hooks[name] = hook
	}
	for name, perm := range r.Permissions {
		perm.Name = name
		r.Permissions[name] = perm
	}
	for name, tmpl := range r.Templates {
		tmpl.Name = name
		r.Templates[name] = tmpl
	}

	return r, nil
}

// Save writes the registry to a YAML file.
func (r *Registry) Save(path string) error {
	f := registryFile{
		Servers:     r.Servers,
		Skills:      r.Skills,
		Hooks:       r.Hooks,
		Permissions: r.Permissions,
		Templates:   r.Templates,
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

// Add adds a server to the registry. Returns an error if the name already exists.
func (r *Registry) Add(server model.ServerDef) error {
	if _, exists := r.Servers[server.Name]; exists {
		return hysterr.ServerAlreadyExists(server.Name)
	}
	r.Servers[server.Name] = server
	return nil
}

// Update replaces an existing server definition. Returns an error if not found.
func (r *Registry) Update(name string, server model.ServerDef) error {
	if _, exists := r.Servers[name]; !exists {
		return hysterr.ServerNotFound(name)
	}
	server.Name = name
	r.Servers[name] = server
	return nil
}

// UpdateSkill replaces an existing skill definition. Returns an error if not found.
func (r *Registry) UpdateSkill(name string, skill model.SkillDef) error {
	if _, exists := r.Skills[name]; !exists {
		return hysterr.SkillNotFound(name)
	}
	skill.Name = name
	r.Skills[name] = skill
	return nil
}

// UpdateHook replaces an existing hook definition. Returns an error if not found.
func (r *Registry) UpdateHook(name string, hook model.HookDef) error {
	if _, exists := r.Hooks[name]; !exists {
		return hysterr.HookNotFound(name)
	}
	hook.Name = name
	r.Hooks[name] = hook
	return nil
}

// UpdatePermission replaces an existing permission rule. Returns an error if not found.
func (r *Registry) UpdatePermission(name string, perm model.PermissionRule) error {
	if _, exists := r.Permissions[name]; !exists {
		return hysterr.PermissionNotFound(name)
	}
	perm.Name = name
	r.Permissions[name] = perm
	return nil
}

// UpdateTemplate replaces an existing template definition. Returns an error if not found.
func (r *Registry) UpdateTemplate(name string, tmpl model.TemplateDef) error {
	if _, exists := r.Templates[name]; !exists {
		return hysterr.TemplateNotFound(name)
	}
	tmpl.Name = name
	r.Templates[name] = tmpl
	return nil
}

// Delete removes a server from the registry.
// Returns an error if the server is referenced by any tag.
func (r *Registry) Delete(name string) error {
	if _, exists := r.Servers[name]; !exists {
		return hysterr.ServerNotFound(name)
	}

	// Check tag references.
	for tag, servers := range r.Tags {
		for _, s := range servers {
			if s == name {
				return hysterr.ServerReferenced(name, tag)
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
		return nil, hysterr.TagNotFound(tag)
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
		return hysterr.TagAlreadyExists(name)
	}
	r.Tags[name] = servers
	return nil
}

// RemoveTag deletes a tag. Returns an error if the tag does not exist.
func (r *Registry) RemoveTag(name string) error {
	if _, exists := r.Tags[name]; !exists {
		return hysterr.TagNotFound(name)
	}
	delete(r.Tags, name)
	return nil
}

// UpdateTag replaces the server list for an existing tag.
// Returns an error if the tag does not exist.
func (r *Registry) UpdateTag(name string, servers []string) error {
	if _, exists := r.Tags[name]; !exists {
		return hysterr.TagNotFound(name)
	}
	r.Tags[name] = servers
	return nil
}

// --- Skills CRUD ---

// AddSkill adds a skill to the registry. Returns an error if the name already exists.
func (r *Registry) AddSkill(skill model.SkillDef) error {
	if _, exists := r.Skills[skill.Name]; exists {
		return hysterr.SkillAlreadyExists(skill.Name)
	}
	r.Skills[skill.Name] = skill
	return nil
}

// GetSkill returns a skill by name.
func (r *Registry) GetSkill(name string) (model.SkillDef, bool) {
	skill, ok := r.Skills[name]
	return skill, ok
}

// ListSkills returns all skills sorted by name.
func (r *Registry) ListSkills() []model.SkillDef {
	skills := make([]model.SkillDef, 0, len(r.Skills))
	for _, s := range r.Skills {
		skills = append(skills, s)
	}
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})
	return skills
}

// DeleteSkill removes a skill from the registry.
func (r *Registry) DeleteSkill(name string) error {
	if _, exists := r.Skills[name]; !exists {
		return hysterr.SkillNotFound(name)
	}
	delete(r.Skills, name)
	return nil
}

// --- Hooks CRUD ---

// AddHook adds a hook to the registry. Returns an error if the name already exists.
func (r *Registry) AddHook(hook model.HookDef) error {
	if _, exists := r.Hooks[hook.Name]; exists {
		return hysterr.HookAlreadyExists(hook.Name)
	}
	r.Hooks[hook.Name] = hook
	return nil
}

// GetHook returns a hook by name.
func (r *Registry) GetHook(name string) (model.HookDef, bool) {
	hook, ok := r.Hooks[name]
	return hook, ok
}

// ListHooks returns all hooks sorted by name.
func (r *Registry) ListHooks() []model.HookDef {
	hooks := make([]model.HookDef, 0, len(r.Hooks))
	for _, h := range r.Hooks {
		hooks = append(hooks, h)
	}
	sort.Slice(hooks, func(i, j int) bool {
		return hooks[i].Name < hooks[j].Name
	})
	return hooks
}

// DeleteHook removes a hook from the registry.
func (r *Registry) DeleteHook(name string) error {
	if _, exists := r.Hooks[name]; !exists {
		return hysterr.HookNotFound(name)
	}
	delete(r.Hooks, name)
	return nil
}

// --- Permissions CRUD ---

// AddPermission adds a permission rule to the registry.
func (r *Registry) AddPermission(perm model.PermissionRule) error {
	if _, exists := r.Permissions[perm.Name]; exists {
		return hysterr.PermissionAlreadyExists(perm.Name)
	}
	r.Permissions[perm.Name] = perm
	return nil
}

// GetPermission returns a permission by name.
func (r *Registry) GetPermission(name string) (model.PermissionRule, bool) {
	perm, ok := r.Permissions[name]
	return perm, ok
}

// ListPermissions returns all permissions sorted by name.
func (r *Registry) ListPermissions() []model.PermissionRule {
	perms := make([]model.PermissionRule, 0, len(r.Permissions))
	for _, p := range r.Permissions {
		perms = append(perms, p)
	}
	sort.Slice(perms, func(i, j int) bool {
		return perms[i].Name < perms[j].Name
	})
	return perms
}

// DeletePermission removes a permission from the registry.
func (r *Registry) DeletePermission(name string) error {
	if _, exists := r.Permissions[name]; !exists {
		return hysterr.PermissionNotFound(name)
	}
	delete(r.Permissions, name)
	return nil
}

// --- Templates CRUD ---

// AddTemplate adds a template to the registry.
func (r *Registry) AddTemplate(tmpl model.TemplateDef) error {
	if _, exists := r.Templates[tmpl.Name]; exists {
		return hysterr.TemplateAlreadyExists(tmpl.Name)
	}
	r.Templates[tmpl.Name] = tmpl
	return nil
}

// GetTemplate returns a template by name.
func (r *Registry) GetTemplate(name string) (model.TemplateDef, bool) {
	tmpl, ok := r.Templates[name]
	return tmpl, ok
}

// ListTemplates returns all templates sorted by name.
func (r *Registry) ListTemplates() []model.TemplateDef {
	tmpls := make([]model.TemplateDef, 0, len(r.Templates))
	for _, t := range r.Templates {
		tmpls = append(tmpls, t)
	}
	sort.Slice(tmpls, func(i, j int) bool {
		return tmpls[i].Name < tmpls[j].Name
	})
	return tmpls
}

// DeleteTemplate removes a template from the registry.
func (r *Registry) DeleteTemplate(name string) error {
	if _, exists := r.Templates[name]; !exists {
		return hysterr.TemplateNotFound(name)
	}
	delete(r.Templates, name)
	return nil
}

func empty() *Registry {
	return &Registry{
		Servers:     make(map[string]model.ServerDef),
		Skills:      make(map[string]model.SkillDef),
		Hooks:       make(map[string]model.HookDef),
		Permissions: make(map[string]model.PermissionRule),
		Templates:   make(map[string]model.TemplateDef),
		Tags:        make(map[string][]string),
	}
}
