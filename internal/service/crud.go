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

func (s *Service) AddServer(srv model.ServerDef) error {
	if err := s.registry.Servers.Add(srv); err != nil {
		return err
	}
	return s.saveRegistry()
}

func (s *Service) UpdateServer(name string, srv model.ServerDef) error {
	if err := s.registry.Servers.Update(name, srv); err != nil {
		return err
	}
	return s.saveRegistry()
}

func (s *Service) DeleteServer(name string) error {
	if err := s.registry.DeleteServer(name); err != nil {
		return err
	}
	return s.saveRegistry()
}

func (s *Service) ListServers() []model.ServerDef {
	return s.registry.Servers.List()
}

func (s *Service) GetServer(name string) (model.ServerDef, bool) {
	return s.registry.Servers.Get(name)
}

// --- Skill CRUD ---

func (s *Service) AddSkill(skill model.SkillDef) error {
	if err := s.registry.Skills.Add(skill); err != nil {
		return err
	}
	return s.saveRegistry()
}

func (s *Service) UpdateSkill(name string, skill model.SkillDef) error {
	if err := s.registry.Skills.Update(name, skill); err != nil {
		return err
	}
	return s.saveRegistry()
}

func (s *Service) DeleteSkill(name string) error {
	if err := s.registry.Skills.Delete(name); err != nil {
		return err
	}
	return s.saveRegistry()
}

func (s *Service) ListSkills() []model.SkillDef {
	return s.registry.Skills.List()
}

func (s *Service) GetSkill(name string) (model.SkillDef, bool) {
	return s.registry.Skills.Get(name)
}

// --- Hook CRUD ---

func (s *Service) AddHook(hook model.HookDef) error {
	if err := s.registry.Hooks.Add(hook); err != nil {
		return err
	}
	return s.saveRegistry()
}

func (s *Service) UpdateHook(name string, hook model.HookDef) error {
	if err := s.registry.Hooks.Update(name, hook); err != nil {
		return err
	}
	return s.saveRegistry()
}

func (s *Service) DeleteHook(name string) error {
	if err := s.registry.Hooks.Delete(name); err != nil {
		return err
	}
	return s.saveRegistry()
}

func (s *Service) ListHooks() []model.HookDef {
	return s.registry.Hooks.List()
}

func (s *Service) GetHook(name string) (model.HookDef, bool) {
	return s.registry.Hooks.Get(name)
}

// --- Permission CRUD ---

func (s *Service) AddPermission(perm model.PermissionRule) error {
	if err := s.registry.Permissions.Add(perm); err != nil {
		return err
	}
	return s.saveRegistry()
}

func (s *Service) UpdatePermission(name string, perm model.PermissionRule) error {
	if err := s.registry.Permissions.Update(name, perm); err != nil {
		return err
	}
	return s.saveRegistry()
}

func (s *Service) DeletePermission(name string) error {
	if err := s.registry.Permissions.Delete(name); err != nil {
		return err
	}
	return s.saveRegistry()
}

func (s *Service) ListPermissions() []model.PermissionRule {
	return s.registry.Permissions.List()
}

func (s *Service) GetPermission(name string) (model.PermissionRule, bool) {
	return s.registry.Permissions.Get(name)
}

// --- Template CRUD ---

func (s *Service) AddTemplate(tmpl model.TemplateDef) error {
	if err := s.registry.Templates.Add(tmpl); err != nil {
		return err
	}
	return s.saveRegistry()
}

func (s *Service) UpdateTemplate(name string, tmpl model.TemplateDef) error {
	if err := s.registry.Templates.Update(name, tmpl); err != nil {
		return err
	}
	return s.saveRegistry()
}

func (s *Service) DeleteTemplate(name string) error {
	if err := s.registry.Templates.Delete(name); err != nil {
		return err
	}
	return s.saveRegistry()
}

func (s *Service) ListTemplates() []model.TemplateDef {
	return s.registry.Templates.List()
}

func (s *Service) GetTemplate(name string) (model.TemplateDef, bool) {
	return s.registry.Templates.Get(name)
}

// --- Prompt CRUD ---

func (s *Service) AddPrompt(prompt model.PromptDef) error {
	if err := s.registry.Prompts.Add(prompt); err != nil {
		return err
	}
	return s.saveRegistry()
}

func (s *Service) UpdatePrompt(name string, prompt model.PromptDef) error {
	if err := s.registry.Prompts.Update(name, prompt); err != nil {
		return err
	}
	return s.saveRegistry()
}

func (s *Service) DeletePrompt(name string) error {
	if err := s.registry.Prompts.Delete(name); err != nil {
		return err
	}
	return s.saveRegistry()
}

func (s *Service) ListPrompts() []model.PromptDef {
	return s.registry.Prompts.List()
}

func (s *Service) GetPrompt(name string) (model.PromptDef, bool) {
	return s.registry.Prompts.Get(name)
}

// --- Tag queries ---

func (s *Service) ExpandTag(tag string) ([]string, error) {
	return s.registry.ExpandTag(tag)
}

// --- Project CRUD ---

func (s *Service) AddProject(proj model.Project) error {
	if err := s.projects.Add(proj); err != nil {
		return err
	}
	return s.saveProjects()
}

func (s *Service) DeleteProject(name string) error {
	if err := s.projects.Remove(name); err != nil {
		return err
	}
	return s.saveProjects()
}

func (s *Service) ListProjects() []model.Project {
	return s.projects.List()
}

func (s *Service) GetProject(name string) (model.Project, bool) {
	return s.projects.Get(name)
}

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

func (s *Service) AssignServer(projectName, serverName string) error {
	if err := s.projects.Assign(projectName, serverName); err != nil {
		return err
	}
	return s.saveProjects()
}

func (s *Service) UnassignServer(projectName, serverName string) error {
	if err := s.projects.Unassign(projectName, serverName); err != nil {
		return err
	}
	return s.saveProjects()
}

func (s *Service) SetOverride(projectName, serverName string, override model.ServerOverride) error {
	if err := s.projects.SetOverride(projectName, serverName, override); err != nil {
		return err
	}
	return s.saveProjects()
}

func (s *Service) AssignSkill(projectName, skillName string) error {
	if err := s.projects.AssignSkill(projectName, skillName); err != nil {
		return err
	}
	return s.saveProjects()
}

func (s *Service) UnassignSkill(projectName, skillName string) error {
	if err := s.projects.UnassignSkill(projectName, skillName); err != nil {
		return err
	}
	return s.saveProjects()
}

func (s *Service) AssignHook(projectName, hookName string) error {
	if err := s.projects.AssignHook(projectName, hookName); err != nil {
		return err
	}
	return s.saveProjects()
}

func (s *Service) UnassignHook(projectName, hookName string) error {
	if err := s.projects.UnassignHook(projectName, hookName); err != nil {
		return err
	}
	return s.saveProjects()
}

func (s *Service) AssignPermission(projectName, permName string) error {
	if err := s.projects.AssignPermission(projectName, permName); err != nil {
		return err
	}
	return s.saveProjects()
}

func (s *Service) UnassignPermission(projectName, permName string) error {
	if err := s.projects.UnassignPermission(projectName, permName); err != nil {
		return err
	}
	return s.saveProjects()
}

func (s *Service) AssignPrompt(projectName, promptName string) error {
	if err := s.projects.AssignPrompt(projectName, promptName); err != nil {
		return err
	}
	return s.saveProjects()
}

func (s *Service) UnassignPrompt(projectName, promptName string) error {
	if err := s.projects.UnassignPrompt(projectName, promptName); err != nil {
		return err
	}
	return s.saveProjects()
}

func (s *Service) SetClaudeMDTemplate(projectName, templateName string) error {
	if err := s.projects.SetClaudeMDTemplate(projectName, templateName); err != nil {
		return err
	}
	return s.saveProjects()
}

func (s *Service) ClearClaudeMDTemplate(projectName string) error {
	if err := s.projects.ClearClaudeMDTemplate(projectName); err != nil {
		return err
	}
	return s.saveProjects()
}

// --- Query helpers ---

// CountServerProfileRefs counts how many profiles reference each server
// (including via tag expansion). This has special logic and cannot be generic.
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

// countProfileRefs is a generic helper for counting profile references.
func (s *Service) countProfileRefs(getNamesFromProject func(model.Project) []string) map[string]int {
	counts := make(map[string]int)
	for _, proj := range s.projects.List() {
		seen := make(map[string]bool)
		for _, name := range getNamesFromProject(proj) {
			if !seen[name] {
				seen[name] = true
				counts[name]++
			}
		}
	}
	return counts
}

func (s *Service) CountSkillProfileRefs() map[string]int {
	return s.countProfileRefs(func(p model.Project) []string { return p.Skills })
}

func (s *Service) CountHookProfileRefs() map[string]int {
	return s.countProfileRefs(func(p model.Project) []string { return p.Hooks })
}

func (s *Service) CountPermissionProfileRefs() map[string]int {
	return s.countProfileRefs(func(p model.Project) []string { return p.Permissions })
}

func (s *Service) CountPromptProfileRefs() map[string]int {
	return s.countProfileRefs(func(p model.Project) []string { return p.Prompts })
}

func (s *Service) CountTemplateProfileRefs() map[string]int {
	counts := make(map[string]int)
	for _, proj := range s.projects.List() {
		if proj.ClaudeMD != "" {
			counts[proj.ClaudeMD]++
		}
	}
	return counts
}

// CountAssignedServers returns the number of servers assigned to a project.
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

// InstallCatalogSkill writes inline skill content to a file and registers it.
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

func (s *Service) HasLaunched(projectName string) bool {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return false
	}
	return proj.Launched
}

func (s *Service) MarkLaunched(projectName string) error {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return hysterr.ProjectNotFound(projectName)
	}
	proj.Launched = true
	s.projects.Projects[projectName] = proj
	return s.saveProjects()
}

func (s *Service) GetActiveProfile(projectName string) (string, error) {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return "", hysterr.ProjectNotFound(projectName)
	}
	return proj.ActiveProfile, nil
}

func (s *Service) SetActiveProfile(projectName, profileName string) error {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return hysterr.ProjectNotFound(projectName)
	}
	if _, err := s.loadProfile(proj, profileName); err != nil {
		return err
	}
	proj.ActiveProfile = profileName
	s.projects.Projects[projectName] = proj
	return s.saveProjects()
}

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

func (s *Service) ListGlobalProfiles() ([]profile.Profile, error) {
	return s.profiles.List()
}

func (s *Service) ListProjectProfiles(projectName string) (map[string]model.ProjectProfile, error) {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return nil, hysterr.ProjectNotFound(projectName)
	}
	return proj.Profiles, nil
}

func (s *Service) ExportProfile(name string) ([]byte, error) {
	return s.profiles.Export(name)
}

func (s *Service) ImportProfile(data []byte) (*profile.Profile, error) {
	return s.profiles.Import(data)
}

func (s *Service) ImportProfileAs(data []byte, newName string) (*profile.Profile, error) {
	return s.profiles.ImportAs(data, newName)
}

func (s *Service) IsEmpty() bool {
	return s.registry.Servers.Len() == 0 && len(s.projects.Projects) == 0
}

type ConfigScanResult struct {
	Path       string
	Candidates []ImportCandidate
}

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
