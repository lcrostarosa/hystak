package service

import (
	"os"
	"path/filepath"
	"sort"

	hysterr "github.com/lcrostarosa/hystak/internal/errors"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/profile"
)

// --- Server CRUD ---

// AddServer adds a server to the registry and saves.
func (s *Service) AddServer(srv model.ServerDef) error {
	if err := s.registry.Add(srv); err != nil {
		return err
	}
	return s.saveRegistry()
}

// UpdateServer updates an existing server and saves.
func (s *Service) UpdateServer(name string, srv model.ServerDef) error {
	if err := s.registry.Update(name, srv); err != nil {
		return err
	}
	return s.saveRegistry()
}

// DeleteServer removes a server from the registry and saves.
func (s *Service) DeleteServer(name string) error {
	if err := s.registry.Delete(name); err != nil {
		return err
	}
	return s.saveRegistry()
}

// ListServers returns all servers sorted by name.
func (s *Service) ListServers() []model.ServerDef {
	return s.registry.List()
}

// GetServer returns a server by name.
func (s *Service) GetServer(name string) (model.ServerDef, bool) {
	return s.registry.Get(name)
}

// --- Skill CRUD ---

// AddSkill adds a skill to the registry and saves.
func (s *Service) AddSkill(skill model.SkillDef) error {
	if err := s.registry.AddSkill(skill); err != nil {
		return err
	}
	return s.saveRegistry()
}

// UpdateSkill updates an existing skill and saves.
func (s *Service) UpdateSkill(name string, skill model.SkillDef) error {
	if err := s.registry.UpdateSkill(name, skill); err != nil {
		return err
	}
	return s.saveRegistry()
}

// DeleteSkill removes a skill from the registry and saves.
func (s *Service) DeleteSkill(name string) error {
	if err := s.registry.DeleteSkill(name); err != nil {
		return err
	}
	return s.saveRegistry()
}

// ListSkills returns all skills sorted by name.
func (s *Service) ListSkills() []model.SkillDef {
	return s.registry.ListSkills()
}

// GetSkill returns a skill by name.
func (s *Service) GetSkill(name string) (model.SkillDef, bool) {
	return s.registry.GetSkill(name)
}

// --- Hook CRUD ---

// AddHook adds a hook to the registry and saves.
func (s *Service) AddHook(hook model.HookDef) error {
	if err := s.registry.AddHook(hook); err != nil {
		return err
	}
	return s.saveRegistry()
}

// UpdateHook updates an existing hook and saves.
func (s *Service) UpdateHook(name string, hook model.HookDef) error {
	if err := s.registry.UpdateHook(name, hook); err != nil {
		return err
	}
	return s.saveRegistry()
}

// DeleteHook removes a hook from the registry and saves.
func (s *Service) DeleteHook(name string) error {
	if err := s.registry.DeleteHook(name); err != nil {
		return err
	}
	return s.saveRegistry()
}

// ListHooks returns all hooks sorted by name.
func (s *Service) ListHooks() []model.HookDef {
	return s.registry.ListHooks()
}

// GetHook returns a hook by name.
func (s *Service) GetHook(name string) (model.HookDef, bool) {
	return s.registry.GetHook(name)
}

// --- Permission CRUD ---

// AddPermission adds a permission to the registry and saves.
func (s *Service) AddPermission(perm model.PermissionRule) error {
	if err := s.registry.AddPermission(perm); err != nil {
		return err
	}
	return s.saveRegistry()
}

// UpdatePermission updates an existing permission and saves.
func (s *Service) UpdatePermission(name string, perm model.PermissionRule) error {
	if err := s.registry.UpdatePermission(name, perm); err != nil {
		return err
	}
	return s.saveRegistry()
}

// DeletePermission removes a permission from the registry and saves.
func (s *Service) DeletePermission(name string) error {
	if err := s.registry.DeletePermission(name); err != nil {
		return err
	}
	return s.saveRegistry()
}

// ListPermissions returns all permissions sorted by name.
func (s *Service) ListPermissions() []model.PermissionRule {
	return s.registry.ListPermissions()
}

// GetPermission returns a permission by name.
func (s *Service) GetPermission(name string) (model.PermissionRule, bool) {
	return s.registry.GetPermission(name)
}

// --- Template CRUD ---

// AddTemplate adds a template to the registry and saves.
func (s *Service) AddTemplate(tmpl model.TemplateDef) error {
	if err := s.registry.AddTemplate(tmpl); err != nil {
		return err
	}
	return s.saveRegistry()
}

// UpdateTemplate updates an existing template and saves.
func (s *Service) UpdateTemplate(name string, tmpl model.TemplateDef) error {
	if err := s.registry.UpdateTemplate(name, tmpl); err != nil {
		return err
	}
	return s.saveRegistry()
}

// DeleteTemplate removes a template from the registry and saves.
func (s *Service) DeleteTemplate(name string) error {
	if err := s.registry.DeleteTemplate(name); err != nil {
		return err
	}
	return s.saveRegistry()
}

// ListTemplates returns all templates sorted by name.
func (s *Service) ListTemplates() []model.TemplateDef {
	return s.registry.ListTemplates()
}

// GetTemplate returns a template by name.
func (s *Service) GetTemplate(name string) (model.TemplateDef, bool) {
	return s.registry.GetTemplate(name)
}

// --- Prompt CRUD ---

// AddPrompt adds a prompt to the registry and saves.
func (s *Service) AddPrompt(prompt model.PromptDef) error {
	if err := s.registry.AddPrompt(prompt); err != nil {
		return err
	}
	return s.saveRegistry()
}

// UpdatePrompt updates an existing prompt and saves.
func (s *Service) UpdatePrompt(name string, prompt model.PromptDef) error {
	if err := s.registry.UpdatePrompt(name, prompt); err != nil {
		return err
	}
	return s.saveRegistry()
}

// DeletePrompt removes a prompt from the registry and saves.
func (s *Service) DeletePrompt(name string) error {
	if err := s.registry.DeletePrompt(name); err != nil {
		return err
	}
	return s.saveRegistry()
}

// ListPrompts returns all prompts sorted by (Order, Name).
func (s *Service) ListPrompts() []model.PromptDef {
	return s.registry.ListPrompts()
}

// GetPrompt returns a prompt by name.
func (s *Service) GetPrompt(name string) (model.PromptDef, bool) {
	return s.registry.GetPrompt(name)
}

// --- Tag queries ---

// ExpandTag returns the server names for a tag.
func (s *Service) ExpandTag(tag string) ([]string, error) {
	return s.registry.ExpandTag(tag)
}

// --- Project CRUD ---

// AddProject adds a project to the store and saves.
func (s *Service) AddProject(proj model.Project) error {
	if err := s.projects.Add(proj); err != nil {
		return err
	}
	return s.saveProjects()
}

// DeleteProject removes a project from the store and saves.
func (s *Service) DeleteProject(name string) error {
	if err := s.projects.Remove(name); err != nil {
		return err
	}
	return s.saveProjects()
}

// ListProjects returns all projects sorted by name.
func (s *Service) ListProjects() []model.Project {
	return s.projects.List()
}

// GetProject returns a project by name.
func (s *Service) GetProject(name string) (model.Project, bool) {
	return s.projects.Get(name)
}

// ListProjectNames returns sorted project names.
func (s *Service) ListProjectNames() []string {
	projects := s.projects.List()
	names := make([]string, len(projects))
	for i, p := range projects {
		names[i] = p.Name
	}
	sort.Strings(names)
	return names
}

// --- Project Assignment ---

// AssignServer adds a server to a project and saves.
func (s *Service) AssignServer(projectName, serverName string) error {
	if err := s.projects.Assign(projectName, serverName); err != nil {
		return err
	}
	return s.saveProjects()
}

// UnassignServer removes a server from a project and saves.
func (s *Service) UnassignServer(projectName, serverName string) error {
	if err := s.projects.Unassign(projectName, serverName); err != nil {
		return err
	}
	return s.saveProjects()
}

// SetOverride sets an override for a server in a project and saves.
func (s *Service) SetOverride(projectName, serverName string, override model.ServerOverride) error {
	if err := s.projects.SetOverride(projectName, serverName, override); err != nil {
		return err
	}
	return s.saveProjects()
}

// AssignSkill adds a skill to a project and saves.
func (s *Service) AssignSkill(projectName, skillName string) error {
	if err := s.projects.AssignSkill(projectName, skillName); err != nil {
		return err
	}
	return s.saveProjects()
}

// UnassignSkill removes a skill from a project and saves.
func (s *Service) UnassignSkill(projectName, skillName string) error {
	if err := s.projects.UnassignSkill(projectName, skillName); err != nil {
		return err
	}
	return s.saveProjects()
}

// AssignHook adds a hook to a project and saves.
func (s *Service) AssignHook(projectName, hookName string) error {
	if err := s.projects.AssignHook(projectName, hookName); err != nil {
		return err
	}
	return s.saveProjects()
}

// UnassignHook removes a hook from a project and saves.
func (s *Service) UnassignHook(projectName, hookName string) error {
	if err := s.projects.UnassignHook(projectName, hookName); err != nil {
		return err
	}
	return s.saveProjects()
}

// AssignPermission adds a permission to a project and saves.
func (s *Service) AssignPermission(projectName, permName string) error {
	if err := s.projects.AssignPermission(projectName, permName); err != nil {
		return err
	}
	return s.saveProjects()
}

// UnassignPermission removes a permission from a project and saves.
func (s *Service) UnassignPermission(projectName, permName string) error {
	if err := s.projects.UnassignPermission(projectName, permName); err != nil {
		return err
	}
	return s.saveProjects()
}

// AssignPrompt adds a prompt to a project and saves.
func (s *Service) AssignPrompt(projectName, promptName string) error {
	if err := s.projects.AssignPrompt(projectName, promptName); err != nil {
		return err
	}
	return s.saveProjects()
}

// UnassignPrompt removes a prompt from a project and saves.
func (s *Service) UnassignPrompt(projectName, promptName string) error {
	if err := s.projects.UnassignPrompt(projectName, promptName); err != nil {
		return err
	}
	return s.saveProjects()
}

// SetClaudeMDTemplate sets the template for a project and saves.
func (s *Service) SetClaudeMDTemplate(projectName, templateName string) error {
	if err := s.projects.SetClaudeMDTemplate(projectName, templateName); err != nil {
		return err
	}
	return s.saveProjects()
}

// ClearClaudeMDTemplate removes the template from a project and saves.
func (s *Service) ClearClaudeMDTemplate(projectName string) error {
	if err := s.projects.ClearClaudeMDTemplate(projectName); err != nil {
		return err
	}
	return s.saveProjects()
}

// --- Query helpers (moved from TUI) ---

// CountServerProfileRefs counts how many profiles reference each server
// (including via tag expansion).
func (s *Service) CountServerProfileRefs() map[string]int {
	counts := make(map[string]int)
	for _, proj := range s.projects.List() {
		seen := make(map[string]bool)
		for _, tag := range proj.Tags {
			if names, err := s.registry.ExpandTag(tag); err == nil {
				for _, name := range names {
					if !seen[name] {
						seen[name] = true
						counts[name]++
					}
				}
			}
		}
		for _, mcp := range proj.MCPs {
			if !seen[mcp.Name] {
				seen[mcp.Name] = true
				counts[mcp.Name]++
			}
		}
	}
	return counts
}

// CountSkillProfileRefs counts how many profiles reference each skill.
func (s *Service) CountSkillProfileRefs() map[string]int {
	counts := make(map[string]int)
	for _, proj := range s.projects.List() {
		seen := make(map[string]bool)
		for _, name := range proj.Skills {
			if !seen[name] {
				seen[name] = true
				counts[name]++
			}
		}
	}
	return counts
}

// CountHookProfileRefs counts how many profiles reference each hook.
func (s *Service) CountHookProfileRefs() map[string]int {
	counts := make(map[string]int)
	for _, proj := range s.projects.List() {
		seen := make(map[string]bool)
		for _, name := range proj.Hooks {
			if !seen[name] {
				seen[name] = true
				counts[name]++
			}
		}
	}
	return counts
}

// CountPermissionProfileRefs counts how many profiles reference each permission.
func (s *Service) CountPermissionProfileRefs() map[string]int {
	counts := make(map[string]int)
	for _, proj := range s.projects.List() {
		seen := make(map[string]bool)
		for _, name := range proj.Permissions {
			if !seen[name] {
				seen[name] = true
				counts[name]++
			}
		}
	}
	return counts
}

// CountPromptProfileRefs counts how many profiles reference each prompt.
func (s *Service) CountPromptProfileRefs() map[string]int {
	counts := make(map[string]int)
	for _, proj := range s.projects.List() {
		seen := make(map[string]bool)
		for _, name := range proj.Prompts {
			if !seen[name] {
				seen[name] = true
				counts[name]++
			}
		}
	}
	return counts
}

// CountTemplateProfileRefs counts how many profiles reference each template.
func (s *Service) CountTemplateProfileRefs() map[string]int {
	counts := make(map[string]int)
	for _, proj := range s.projects.List() {
		if proj.ClaudeMD != "" {
			counts[proj.ClaudeMD]++
		}
	}
	return counts
}

// CountAssignedServers returns the number of servers assigned to a project
// (including via tag expansion).
func (s *Service) CountAssignedServers(proj model.Project) int {
	seen := make(map[string]bool)
	for _, tag := range proj.Tags {
		if names, err := s.registry.ExpandTag(tag); err == nil {
			for _, name := range names {
				seen[name] = true
			}
		}
	}
	for _, mcp := range proj.MCPs {
		seen[mcp.Name] = true
	}
	return len(seen)
}

// IsServerAssigned checks if a server is assigned to the given project
// (either directly via MCPs or via tag expansion).
func (s *Service) IsServerAssigned(proj model.Project, serverName string) bool {
	for _, mcp := range proj.MCPs {
		if mcp.Name == serverName {
			return true
		}
	}
	for _, tag := range proj.Tags {
		if names, err := s.registry.ExpandTag(tag); err == nil {
			for _, name := range names {
				if name == serverName {
					return true
				}
			}
		}
	}
	return false
}

// IsServerFromTag checks if a server's assignment comes only from tag expansion
// (not from a direct MCP entry).
func (s *Service) IsServerFromTag(proj model.Project, serverName string) bool {
	for _, mcp := range proj.MCPs {
		if mcp.Name == serverName {
			return false
		}
	}
	for _, tag := range proj.Tags {
		if names, err := s.registry.ExpandTag(tag); err == nil {
			for _, name := range names {
				if name == serverName {
					return true
				}
			}
		}
	}
	return false
}

// InstallCatalogSkill writes inline skill content to a file in the config
// directory and adds the skill to the registry.
func (s *Service) InstallCatalogSkill(name, description, content string) error {
	skillDir := filepath.Join(s.configDir, "skills")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return err
	}
	skillPath := filepath.Join(skillDir, name+".md")
	if err := os.WriteFile(skillPath, []byte(content), 0o644); err != nil {
		return err
	}
	return s.AddSkill(model.SkillDef{
		Name:        name,
		Description: description,
		Source:      skillPath,
	})
}

// --- Profile management ---

// HasLaunched reports whether a project has been launched before.
func (s *Service) HasLaunched(projectName string) bool {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return false
	}
	return proj.Launched
}

// MarkLaunched sets the Launched flag on a project and saves.
func (s *Service) MarkLaunched(projectName string) error {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return hysterr.ProjectNotFound(projectName)
	}
	proj.Launched = true
	s.projects.Projects[projectName] = proj
	return s.saveProjects()
}

// GetActiveProfile returns the active profile name for a project.
func (s *Service) GetActiveProfile(projectName string) (string, error) {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return "", hysterr.ProjectNotFound(projectName)
	}
	return proj.ActiveProfile, nil
}

// SetActiveProfile sets the active profile for a project and saves.
func (s *Service) SetActiveProfile(projectName, profileName string) error {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return hysterr.ProjectNotFound(projectName)
	}

	// Verify the profile exists (project-scoped or global).
	if _, err := s.loadProfile(proj, profileName); err != nil {
		return err
	}

	proj.ActiveProfile = profileName
	s.projects.Projects[projectName] = proj
	return s.saveProjects()
}

// SaveProjectProfile saves a profile to a project's inline profiles map and persists.
func (s *Service) SaveProjectProfile(projectName, profileName string, prof profile.Profile) error {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return hysterr.ProjectNotFound(projectName)
	}

	if proj.Profiles == nil {
		proj.Profiles = make(map[string]model.ProjectProfile)
	}
	proj.Profiles[profileName] = model.ProjectProfile{
		Description: prof.Description,
		MCPs:        prof.MCPs,
		Skills:      prof.Skills,
		Hooks:       prof.Hooks,
		Permissions: prof.Permissions,
		Prompts:     prof.Prompts,
		EnvVars:     prof.EnvVars,
		ClaudeMD:    prof.ClaudeMD,
		Isolation:   string(prof.Isolation),
	}
	s.projects.Projects[projectName] = proj
	return s.saveProjects()
}

// --- Profile sharing ---

// ListGlobalProfiles returns all global profiles (including vanilla).
func (s *Service) ListGlobalProfiles() ([]profile.Profile, error) {
	return s.profiles.List()
}

// ListProjectProfiles returns the project-scoped profiles for a project.
func (s *Service) ListProjectProfiles(projectName string) (map[string]model.ProjectProfile, error) {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return nil, hysterr.ProjectNotFound(projectName)
	}
	return proj.Profiles, nil
}

// ExportProfile exports a global profile as YAML bytes.
func (s *Service) ExportProfile(name string) ([]byte, error) {
	return s.profiles.Export(name)
}

// ImportProfile imports a profile from YAML bytes into the global profiles directory.
func (s *Service) ImportProfile(data []byte) (*profile.Profile, error) {
	return s.profiles.Import(data)
}

// ImportProfileAs imports a profile from YAML bytes under a new name.
func (s *Service) ImportProfileAs(data []byte, newName string) (*profile.Profile, error) {
	return s.profiles.ImportAs(data, newName)
}

// IsEmpty returns true when the registry has no servers and no projects exist.
func (s *Service) IsEmpty() bool {
	return len(s.registry.Servers) == 0 && len(s.projects.Projects) == 0
}

// ConfigScanResult represents a discovered config file with its servers.
type ConfigScanResult struct {
	Path       string
	Candidates []ImportCandidate
}

// ScanForConfigs checks well-known locations for existing MCP configs.
// Scans: .mcp.json in cwd, ~/.claude.json.
// Returns results only for files that exist and contain at least one server.
func (s *Service) ScanForConfigs() []ConfigScanResult {
	var paths []string

	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, ".mcp.json"))
	}

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".claude.json"))
	}

	var results []ConfigScanResult
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		candidates, err := s.ImportFromFile(p)
		if err != nil || len(candidates) == 0 {
			continue
		}
		results = append(results, ConfigScanResult{Path: p, Candidates: candidates})
	}
	return results
}
