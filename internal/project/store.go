package project

import (
	"fmt"
	"maps"
	"os"
	"sort"

	hysterr "github.com/lcrostarosa/hystak/internal/errors"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/registry"
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
		return hysterr.ProjectAlreadyExists(proj.Name)
	}
	s.Projects[proj.Name] = proj
	return nil
}

// Remove deletes a project from the store. Returns an error if not found.
func (s *Store) Remove(name string) error {
	if _, exists := s.Projects[name]; !exists {
		return hysterr.ProjectNotFound(name)
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
		return hysterr.ProjectNotFound(projectName)
	}

	for _, mcp := range proj.MCPs {
		if mcp.Name == serverName {
			return hysterr.ServerAlreadyAssigned(serverName, projectName)
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
		return hysterr.ProjectNotFound(projectName)
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
		return hysterr.ServerNotAssigned(serverName, projectName)
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
		return hysterr.ProjectNotFound(projectName)
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

// AssignSkill adds a skill name to a project's skills list.
// Returns an error if the project is not found or the skill is already assigned.
func (s *Store) AssignSkill(projectName, skillName string) error {
	proj, ok := s.Projects[projectName]
	if !ok {
		return hysterr.ProjectNotFound(projectName)
	}

	for _, sk := range proj.Skills {
		if sk == skillName {
			return hysterr.SkillAlreadyAssigned(skillName, projectName)
		}
	}

	proj.Skills = append(proj.Skills, skillName)
	s.Projects[projectName] = proj
	return nil
}

// UnassignSkill removes a skill from a project's skills list.
// Returns an error if the project or skill assignment is not found.
func (s *Store) UnassignSkill(projectName, skillName string) error {
	proj, ok := s.Projects[projectName]
	if !ok {
		return hysterr.ProjectNotFound(projectName)
	}

	found := false
	skills := make([]string, 0, len(proj.Skills))
	for _, sk := range proj.Skills {
		if sk == skillName {
			found = true
			continue
		}
		skills = append(skills, sk)
	}

	if !found {
		return hysterr.SkillNotAssigned(skillName, projectName)
	}

	proj.Skills = skills
	s.Projects[projectName] = proj
	return nil
}

// AssignHook adds a hook name to a project's hooks list.
// Returns an error if the project is not found or the hook is already assigned.
func (s *Store) AssignHook(projectName, hookName string) error {
	proj, ok := s.Projects[projectName]
	if !ok {
		return hysterr.ProjectNotFound(projectName)
	}

	for _, h := range proj.Hooks {
		if h == hookName {
			return hysterr.HookAlreadyAssigned(hookName, projectName)
		}
	}

	proj.Hooks = append(proj.Hooks, hookName)
	s.Projects[projectName] = proj
	return nil
}

// UnassignHook removes a hook from a project's hooks list.
// Returns an error if the project or hook assignment is not found.
func (s *Store) UnassignHook(projectName, hookName string) error {
	proj, ok := s.Projects[projectName]
	if !ok {
		return hysterr.ProjectNotFound(projectName)
	}

	found := false
	hooks := make([]string, 0, len(proj.Hooks))
	for _, h := range proj.Hooks {
		if h == hookName {
			found = true
			continue
		}
		hooks = append(hooks, h)
	}

	if !found {
		return hysterr.HookNotAssigned(hookName, projectName)
	}

	proj.Hooks = hooks
	s.Projects[projectName] = proj
	return nil
}

// AssignPermission adds a permission name to a project's permissions list.
// Returns an error if the project is not found or the permission is already assigned.
func (s *Store) AssignPermission(projectName, permName string) error {
	proj, ok := s.Projects[projectName]
	if !ok {
		return hysterr.ProjectNotFound(projectName)
	}

	for _, p := range proj.Permissions {
		if p == permName {
			return hysterr.PermissionAlreadyAssigned(permName, projectName)
		}
	}

	proj.Permissions = append(proj.Permissions, permName)
	s.Projects[projectName] = proj
	return nil
}

// UnassignPermission removes a permission from a project's permissions list.
// Returns an error if the project or permission assignment is not found.
func (s *Store) UnassignPermission(projectName, permName string) error {
	proj, ok := s.Projects[projectName]
	if !ok {
		return hysterr.ProjectNotFound(projectName)
	}

	found := false
	perms := make([]string, 0, len(proj.Permissions))
	for _, p := range proj.Permissions {
		if p == permName {
			found = true
			continue
		}
		perms = append(perms, p)
	}

	if !found {
		return hysterr.PermissionNotAssigned(permName, projectName)
	}

	proj.Permissions = perms
	s.Projects[projectName] = proj
	return nil
}

// AssignPrompt adds a prompt name to a project's prompts list.
// Returns an error if the project is not found or the prompt is already assigned.
func (s *Store) AssignPrompt(projectName, promptName string) error {
	proj, ok := s.Projects[projectName]
	if !ok {
		return hysterr.ProjectNotFound(projectName)
	}

	for _, p := range proj.Prompts {
		if p == promptName {
			return hysterr.PromptAlreadyAssigned(promptName, projectName)
		}
	}

	proj.Prompts = append(proj.Prompts, promptName)
	s.Projects[projectName] = proj
	return nil
}

// UnassignPrompt removes a prompt from a project's prompts list.
// Returns an error if the project or prompt assignment is not found.
func (s *Store) UnassignPrompt(projectName, promptName string) error {
	proj, ok := s.Projects[projectName]
	if !ok {
		return hysterr.ProjectNotFound(projectName)
	}

	found := false
	prompts := make([]string, 0, len(proj.Prompts))
	for _, p := range proj.Prompts {
		if p == promptName {
			found = true
			continue
		}
		prompts = append(prompts, p)
	}

	if !found {
		return hysterr.PromptNotAssigned(promptName, projectName)
	}

	proj.Prompts = prompts
	s.Projects[projectName] = proj
	return nil
}

// SetClaudeMDTemplate sets the ClaudeMD template name for a project.
// Returns an error if the project is not found.
func (s *Store) SetClaudeMDTemplate(projectName, templateName string) error {
	proj, ok := s.Projects[projectName]
	if !ok {
		return hysterr.ProjectNotFound(projectName)
	}
	proj.ClaudeMD = templateName
	s.Projects[projectName] = proj
	return nil
}

// ClearClaudeMDTemplate removes the ClaudeMD template assignment from a project.
// Returns an error if the project is not found.
func (s *Store) ClearClaudeMDTemplate(projectName string) error {
	proj, ok := s.Projects[projectName]
	if !ok {
		return hysterr.ProjectNotFound(projectName)
	}
	proj.ClaudeMD = ""
	s.Projects[projectName] = proj
	return nil
}

// SetClients updates the client list for a project.
func (s *Store) SetClients(projectName string, clients []model.ClientType) error {
	proj, ok := s.Projects[projectName]
	if !ok {
		return hysterr.ProjectNotFound(projectName)
	}
	proj.Clients = clients
	s.Projects[projectName] = proj
	return nil
}

// SetTags updates the tag list for a project.
func (s *Store) SetTags(projectName string, tags []string) error {
	proj, ok := s.Projects[projectName]
	if !ok {
		return hysterr.ProjectNotFound(projectName)
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
		return nil, hysterr.ProjectNotFound(projectName)
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
			srv = ApplyOverride(srv, override)
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

// ApplyOverride shallow-merges an override onto a server definition.
//   - env: merge maps (override keys win)
//   - headers: merge maps (override keys win)
//   - args: replace entirely
//   - command, url: replace if non-nil
func ApplyOverride(srv model.ServerDef, override *model.ServerOverride) model.ServerDef {
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
		srv.Env = mergeMaps(srv.Env, override.Env)
	}

	if override.Headers != nil {
		srv.Headers = mergeMaps(srv.Headers, override.Headers)
	}

	return srv
}

// mergeMaps merges base and override maps, with override keys winning.
func mergeMaps(base, override map[string]string) map[string]string {
	merged := make(map[string]string, len(base)+len(override))
	maps.Copy(merged, base)
	maps.Copy(merged, override)
	return merged
}
