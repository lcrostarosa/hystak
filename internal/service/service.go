package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lcrostarosa/hystak/internal/backup"
	"github.com/lcrostarosa/hystak/internal/deploy"
	"github.com/lcrostarosa/hystak/internal/discovery"
	hysterr "github.com/lcrostarosa/hystak/internal/errors"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/profile"
	"github.com/lcrostarosa/hystak/internal/project"
	"github.com/lcrostarosa/hystak/internal/registry"
)

// SyncAction describes what happened to a server during sync.
type SyncAction string

const (
	SyncAdded     SyncAction = "added"
	SyncUpdated   SyncAction = "updated"
	SyncUnchanged SyncAction = "unchanged"
	SyncUnmanaged SyncAction = "unmanaged"
)

// SyncResult reports the outcome for a single server during sync.
type SyncResult struct {
	ServerName string
	Client     model.ClientType
	Action     SyncAction
}

// ImportResolution indicates how to handle an import conflict.
type ImportResolution string

const (
	ImportPending ImportResolution = "pending"
	ImportKeep    ImportResolution = "keep"
	ImportReplace ImportResolution = "replace"
	ImportRename  ImportResolution = "rename"
	ImportSkip    ImportResolution = "skip"
)

// ImportCandidate represents a server discovered during import.
type ImportCandidate struct {
	Name       string
	Server     model.ServerDef
	Conflict   bool
	Resolution ImportResolution
	RenameTo   string
}

// WasImported returns true if this candidate was actually imported
// (non-conflicting, or conflict resolved via replace/rename).
func (c ImportCandidate) WasImported() bool {
	return !c.Conflict || c.Resolution == ImportReplace || c.Resolution == ImportRename
}

// SyncConflict represents a conflict detected during sync preflight.
type SyncConflict struct {
	ResourceType string // "skill", "hook", "permission", "claude_md"
	Name         string
	ExistingPath string // path to the existing file/entry
	Resolution   SyncConflictResolution
}

// SyncConflictResolution describes how a sync conflict should be handled.
type SyncConflictResolution string

const (
	ConflictPending SyncConflictResolution = "pending"
	ConflictKeep    SyncConflictResolution = "keep"
	ConflictReplace SyncConflictResolution = "replace"
	ConflictSkip    SyncConflictResolution = "skip"
)

// effectiveConfig holds the resolved resource names to deploy for a sync operation.
type effectiveConfig struct {
	mcps        []string                          // server names to look up in registry
	overrides   map[string]*model.ServerOverride   // project-level overrides keyed by server name
	skills      []string                          // skill names to look up in registry
	hooks       []string                          // hook names to look up in registry
	permissions []string                          // permission names to look up in registry
	prompts     []string                          // prompt names to look up in registry
	claudeMD    string                            // template name to look up in registry
}

// Service orchestrates registry, projects, and deployers.
type Service struct {
	registry         *registry.Registry
	projects         *project.Store
	deployers        map[model.ClientType]deploy.Deployer
	skillsDeployer   *deploy.SkillsDeployer
	settingsDeployer *deploy.SettingsDeployer
	claudeMDDeployer *deploy.ClaudeMDDeployer
	profiles         *profile.Manager
	backups          *backup.Manager
	configDir        string
}

// New creates a Service by loading registry and projects from configDir.
func New(configDir string) (*Service, error) {
	regPath := filepath.Join(configDir, "registry.yaml")
	projPath := filepath.Join(configDir, "projects.yaml")

	reg, err := registry.Load(regPath)
	if err != nil {
		return nil, fmt.Errorf("loading registry: %w", err)
	}

	proj, err := project.Load(projPath)
	if err != nil {
		return nil, fmt.Errorf("loading projects: %w", err)
	}

	deployers := make(map[model.ClientType]deploy.Deployer)
	for _, ct := range []model.ClientType{model.ClientClaudeCode} {
		d, err := deploy.NewDeployer(ct)
		if err != nil {
			continue
		}
		deployers[ct] = d
	}

	return &Service{
		registry:         reg,
		projects:         proj,
		deployers:        deployers,
		skillsDeployer:   &deploy.SkillsDeployer{},
		settingsDeployer: &deploy.SettingsDeployer{},
		claudeMDDeployer: &deploy.ClaudeMDDeployer{},
		profiles:         profile.NewManager(filepath.Join(configDir, "profiles")),
		backups:          backup.NewManager(filepath.Join(configDir, "backups")),
		configDir:        configDir,
	}, nil
}

// saveRegistry writes the registry back to disk.
func (s *Service) saveRegistry() error {
	return s.registry.Save(filepath.Join(s.configDir, "registry.yaml"))
}

// saveProjects writes the project store back to disk.
func (s *Service) saveProjects() error {
	return s.projects.Save(filepath.Join(s.configDir, "projects.yaml"))
}

// Discover runs the discovery engine against a project path and returns discovered items.
func (s *Service) Discover(projectPath string) *discovery.Items {
	home, _ := os.UserHomeDir()
	claudeHome := filepath.Join(home, ".claude")
	engine := discovery.NewEngine(claudeHome, s.registry)
	return engine.Scan(projectPath)
}

// SyncProject resolves servers for a project and writes them to each configured client.
// If the project has an active profile, config is driven by the profile.
// Projects without a profile that have direct assignments are auto-migrated.
// Unmanaged servers (in client config but not in hystak) are preserved.
func (s *Service) SyncProject(projectName string) ([]SyncResult, error) {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return nil, hysterr.ProjectNotFound(projectName)
	}

	// Auto-migrate legacy projects (have assignments but no active profile).
	if proj.ActiveProfile == "" && hasAssignments(proj) {
		if err := s.migrateToDefaultProfile(projectName); err != nil {
			return nil, fmt.Errorf("migrating project %q to profile: %w", projectName, err)
		}
		proj, _ = s.projects.Get(projectName)
	}

	cfg, err := s.resolveEffectiveConfig(proj)
	if err != nil {
		return nil, err
	}

	return s.syncWithConfig(proj, cfg)
}

// SyncProfile syncs a project using a specific named profile.
func (s *Service) SyncProfile(projectName, profileName string) ([]SyncResult, error) {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return nil, hysterr.ProjectNotFound(projectName)
	}

	prof, err := s.loadProfile(proj, profileName)
	if err != nil {
		return nil, err
	}

	cfg := configFromProfile(proj, prof)
	return s.syncWithConfig(proj, cfg)
}

// SyncProjectToPath syncs a project but deploys to the specified path instead of proj.Path.
// This is used for worktree isolation where configs are deployed to a worktree directory.
func (s *Service) SyncProjectToPath(projectName, deployPath string) ([]SyncResult, error) {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return nil, hysterr.ProjectNotFound(projectName)
	}

	if proj.ActiveProfile == "" && hasAssignments(proj) {
		if err := s.migrateToDefaultProfile(projectName); err != nil {
			return nil, fmt.Errorf("migrating project %q to profile: %w", projectName, err)
		}
		proj, _ = s.projects.Get(projectName)
	}

	// Override the deploy path. syncWithConfig takes proj by value,
	// so this only affects this sync call.
	proj.Path = deployPath

	cfg, err := s.resolveEffectiveConfig(proj)
	if err != nil {
		return nil, err
	}

	return s.syncWithConfig(proj, cfg)
}

// resolveEffectiveConfig determines the config to deploy based on the active profile
// or falls back to direct project assignments.
func (s *Service) resolveEffectiveConfig(proj model.Project) (effectiveConfig, error) {
	if proj.ActiveProfile != "" {
		prof, err := s.loadProfile(proj, proj.ActiveProfile)
		if err != nil {
			return effectiveConfig{}, err
		}
		return configFromProfile(proj, prof), nil
	}
	return s.configFromProject(proj)
}

// resolveExpectedServers resolves MCP servers from an effectiveConfig.
func (s *Service) resolveExpectedServers(cfg effectiveConfig) (map[string]model.ServerDef, error) {
	expected := make(map[string]model.ServerDef, len(cfg.mcps))
	for _, name := range cfg.mcps {
		srv, ok := s.registry.Servers.Get(name)
		if !ok {
			return nil, fmt.Errorf("server %q not found in registry", name)
		}
		if ov, has := cfg.overrides[name]; has {
			srv = project.ApplyOverride(srv, ov)
		}
		expected[name] = srv
	}
	return expected, nil
}

// configFromProject builds an effectiveConfig from the project's direct assignments
// (tags + MCPs + skills + hooks + permissions + claudeMD). This is the legacy path
// for projects without an active profile.
func (s *Service) configFromProject(proj model.Project) (effectiveConfig, error) {
	seen := make(map[string]bool)
	var mcps []string

	for _, tag := range proj.Tags {
		names, err := s.registry.ExpandTag(tag)
		if err != nil {
			return effectiveConfig{}, fmt.Errorf("expanding tag %q: %w", tag, err)
		}
		for _, name := range names {
			if !seen[name] {
				seen[name] = true
				mcps = append(mcps, name)
			}
		}
	}
	for _, mcp := range proj.MCPs {
		if !seen[mcp.Name] {
			seen[mcp.Name] = true
			mcps = append(mcps, mcp.Name)
		}
	}

	return effectiveConfig{
		mcps:        mcps,
		overrides:   buildOverrides(proj.MCPs),
		skills:      proj.Skills,
		hooks:       proj.Hooks,
		permissions: proj.Permissions,
		prompts:     proj.Prompts,
		claudeMD:    proj.ClaudeMD,
	}, nil
}

// configFromProfile builds an effectiveConfig from a profile's selections.
// Project-level MCP overrides are preserved.
func configFromProfile(proj model.Project, prof *profile.Profile) effectiveConfig {
	return effectiveConfig{
		mcps:        prof.MCPs,
		overrides:   buildOverrides(proj.MCPs),
		skills:      prof.Skills,
		hooks:       prof.Hooks,
		permissions: prof.Permissions,
		prompts:     prof.Prompts,
		claudeMD:    prof.ClaudeMD,
	}
}

// buildOverrides extracts MCP overrides from project assignments.
func buildOverrides(mcps []model.MCPAssignment) map[string]*model.ServerOverride {
	overrides := make(map[string]*model.ServerOverride)
	for _, mcp := range mcps {
		if mcp.Overrides != nil {
			overrides[mcp.Name] = mcp.Overrides
		}
	}
	return overrides
}

// loadProfile loads a profile by name, checking project-scoped profiles first,
// then global profiles.
func (s *Service) loadProfile(proj model.Project, name string) (*profile.Profile, error) {
	if name == profile.VanillaName {
		v := profile.Vanilla()
		return &v, nil
	}

	// Check project-scoped profiles.
	if pp, ok := proj.Profiles[name]; ok {
		p := &profile.Profile{
			Name:        name,
			Description: pp.Description,
			MCPs:        pp.MCPs,
			Skills:      pp.Skills,
			Hooks:       pp.Hooks,
			Permissions: pp.Permissions,
			EnvVars:     pp.EnvVars,
			ClaudeMD:    pp.ClaudeMD,
			Isolation:   profile.IsolationStrategy(pp.Isolation),
		}
		return p, nil
	}

	// Check global profiles.
	if s.profiles == nil {
		return nil, hysterr.ProfileNotFound(name)
	}
	return s.profiles.Get(name)
}

// hasAssignments returns true if a project has any direct resource assignments.
func hasAssignments(proj model.Project) bool {
	return len(proj.MCPs) > 0 || len(proj.Tags) > 0 || len(proj.Skills) > 0 ||
		len(proj.Hooks) > 0 || len(proj.Permissions) > 0 || len(proj.Prompts) > 0 ||
		proj.ClaudeMD != ""
}

// migrateToDefaultProfile creates a "default" project-scoped profile from the
// project's current direct assignments and sets it as active.
func (s *Service) migrateToDefaultProfile(projectName string) error {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return hysterr.ProjectNotFound(projectName)
	}

	// Collect MCP names (expand tags + direct MCPs).
	seen := make(map[string]bool)
	var mcpNames []string
	for _, tag := range proj.Tags {
		names, _ := s.registry.ExpandTag(tag)
		for _, name := range names {
			if !seen[name] {
				seen[name] = true
				mcpNames = append(mcpNames, name)
			}
		}
	}
	for _, mcp := range proj.MCPs {
		if !seen[mcp.Name] {
			seen[mcp.Name] = true
			mcpNames = append(mcpNames, mcp.Name)
		}
	}

	pp := model.ProjectProfile{
		Description: "Auto-generated from project assignments",
		MCPs:        mcpNames,
		Skills:      proj.Skills,
		Hooks:       proj.Hooks,
		Permissions: proj.Permissions,
		Prompts:     proj.Prompts,
		ClaudeMD:    proj.ClaudeMD,
	}

	if proj.Profiles == nil {
		proj.Profiles = make(map[string]model.ProjectProfile)
	}
	proj.Profiles["default"] = pp
	proj.ActiveProfile = "default"
	s.projects.Projects[projectName] = proj

	return s.saveProjects()
}

// syncWithConfig deploys MCP servers, skills, hooks, permissions, and CLAUDE.md
// according to the given effectiveConfig.
func (s *Service) syncWithConfig(proj model.Project, cfg effectiveConfig) ([]SyncResult, error) {
	expected, err := s.resolveExpectedServers(cfg)
	if err != nil {
		return nil, err
	}

	// Build set of previously managed servers (from last sync) so we can
	// remove servers that are no longer in the expected set.
	previouslyManaged := make(map[string]bool, len(proj.ManagedMCPs))
	for _, name := range proj.ManagedMCPs {
		previouslyManaged[name] = true
	}

	var results []SyncResult

	for _, ct := range proj.Clients {
		deployer, ok := s.deployers[ct]
		if !ok {
			return nil, fmt.Errorf("no deployer for client %q", ct)
		}

		if err := deployer.Bootstrap(proj.Path); err != nil {
			return nil, fmt.Errorf("bootstrapping %s for project %q: %w", ct, proj.Name, err)
		}

		deployed, err := deployer.ReadServers(proj.Path)
		if err != nil {
			return nil, fmt.Errorf("reading deployed servers for %s in project %q: %w", ct, proj.Name, err)
		}

		// Back up the current config before writing changes.
		configPath := deployer.ConfigPath(proj.Path)
		if _, err := s.backups.Create(ct, proj.Path, configPath); err != nil {
			return nil, fmt.Errorf("backing up %s config for project %q: %w", ct, proj.Name, err)
		}

		merged := make(map[string]model.ServerDef, len(deployed)+len(expected))

		// Preserve unmanaged servers. Servers that were previously managed
		// by hystak but are no longer in the expected set are removed.
		for name, srv := range deployed {
			if _, isExpected := expected[name]; !isExpected {
				if previouslyManaged[name] {
					// Was managed, no longer expected → remove (don't add to merged).
					continue
				}
				merged[name] = srv
				results = append(results, SyncResult{
					ServerName: name,
					Client:     ct,
					Action:     SyncUnmanaged,
				})
			}
		}

		// Write expected servers, tracking what changed.
		for name, srv := range expected {
			merged[name] = srv
			if prev, wasDeployed := deployed[name]; wasDeployed {
				if prev.Equal(srv) {
					results = append(results, SyncResult{
						ServerName: name,
						Client:     ct,
						Action:     SyncUnchanged,
					})
				} else {
					results = append(results, SyncResult{
						ServerName: name,
						Client:     ct,
						Action:     SyncUpdated,
					})
				}
			} else {
				results = append(results, SyncResult{
					ServerName: name,
					Client:     ct,
					Action:     SyncAdded,
				})
			}
		}

		if err := deployer.WriteServers(proj.Path, merged); err != nil {
			return nil, fmt.Errorf("writing servers for %s in project %q: %w", ct, proj.Name, err)
		}
	}

	// Update the managed MCP set for next sync.
	managedNames := make([]string, 0, len(cfg.mcps))
	managedNames = append(managedNames, cfg.mcps...)
	proj.ManagedMCPs = managedNames
	s.projects.Projects[proj.Name] = proj

	if err := s.saveProjects(); err != nil {
		return nil, fmt.Errorf("saving managed MCPs for project %q: %w", proj.Name, err)
	}

	// Sync skills.
	if err := s.syncSkills(proj.Path, cfg.skills); err != nil {
		return nil, fmt.Errorf("syncing skills for project %q: %w", proj.Name, err)
	}

	// Sync hooks and permissions to settings.local.json.
	if err := s.syncSettings(proj.Path, cfg.hooks, cfg.permissions); err != nil {
		return nil, fmt.Errorf("syncing settings for project %q: %w", proj.Name, err)
	}

	// Sync CLAUDE.md template and prompt fragments.
	if err := s.syncClaudeMD(proj.Path, cfg.claudeMD, cfg.prompts); err != nil {
		return nil, fmt.Errorf("syncing CLAUDE.md for project %q: %w", proj.Name, err)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ServerName < results[j].ServerName
	})

	return results, nil
}

// syncSkills resolves skill names from the registry and deploys skill files.
func (s *Service) syncSkills(projectPath string, skillNames []string) error {
	if s.skillsDeployer == nil {
		return nil
	}
	if len(skillNames) == 0 {
		if projectPath != "" {
			// Clean up any previously managed skills.
			return s.skillsDeployer.SyncSkills(projectPath, nil)
		}
		return nil
	}

	skills := make([]model.SkillDef, 0, len(skillNames))
	for _, name := range skillNames {
		skill, ok := s.registry.Skills.Get(name)
		if !ok {
			// Not in registry — likely discovered from the filesystem
			// and already deployed. Skip silently.
			continue
		}
		skills = append(skills, skill)
	}

	return s.skillsDeployer.SyncSkills(projectPath, skills)
}

// syncSettings resolves hooks and permissions from the registry and deploys to settings.local.json.
func (s *Service) syncSettings(projectPath string, hookNames, permNames []string) error {
	if s.settingsDeployer == nil {
		return nil
	}

	if len(hookNames) == 0 && len(permNames) == 0 {
		return nil
	}

	hooks := make([]model.HookDef, 0, len(hookNames))
	for _, name := range hookNames {
		hook, ok := s.registry.Hooks.Get(name)
		if !ok {
			// Not in registry — likely discovered from the filesystem
			// and already deployed. Skip silently.
			continue
		}
		hooks = append(hooks, hook)
	}

	permissions := make([]model.PermissionRule, 0, len(permNames))
	for _, name := range permNames {
		perm, ok := s.registry.Permissions.Get(name)
		if !ok {
			// Not in registry — likely discovered from the filesystem
			// and already deployed. Skip silently.
			continue
		}
		permissions = append(permissions, perm)
	}

	return s.settingsDeployer.SyncSettings(projectPath, hooks, permissions)
}

// syncClaudeMD resolves the template and prompt fragments from the registry and deploys CLAUDE.md.
func (s *Service) syncClaudeMD(projectPath string, templateName string, promptNames []string) error {
	if s.claudeMDDeployer == nil {
		return nil
	}

	var templateSource string
	if templateName != "" {
		tmpl, ok := s.registry.Templates.Get(templateName)
		if !ok {
			return hysterr.TemplateNotFound(templateName)
		}
		templateSource = tmpl.Source
	}

	promptSources := s.resolvePromptSources(promptNames)

	if templateSource == "" && len(promptSources) == 0 {
		return nil
	}

	return s.claudeMDDeployer.SyncClaudeMD(projectPath, templateSource, promptSources)
}

// resolvePromptSources resolves prompt names to source file paths, sorted by (Order, Name)
// with deduplication. Missing registry entries are silently skipped.
func (s *Service) resolvePromptSources(promptNames []string) []string {
	if len(promptNames) == 0 {
		return nil
	}

	type entry struct {
		order  int
		name   string
		source string
	}

	seen := make(map[string]bool)
	var entries []entry
	for _, name := range promptNames {
		if seen[name] {
			continue
		}
		seen[name] = true
		prompt, ok := s.registry.Prompts.Get(name)
		if !ok {
			continue // not in registry, skip silently
		}
		source := prompt.Source
		// Expand relative paths to be relative to configDir.
		if source != "" && !filepath.IsAbs(source) && source[0] != '~' {
			source = filepath.Join(s.configDir, source)
		}
		entries = append(entries, entry{prompt.Order, prompt.Name, source})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].order != entries[j].order {
			return entries[i].order < entries[j].order
		}
		return entries[i].name < entries[j].name
	})

	sources := make([]string, len(entries))
	for i, e := range entries {
		sources[i] = e.source
	}
	return sources
}

// PreviewComposedPrompts reads and composes prompt fragment contents for preview
// without deploying. Returns the composed text that would be written to CLAUDE.md.
func (s *Service) PreviewComposedPrompts(promptNames []string, templateName string) (string, error) {
	var buf strings.Builder

	if templateName != "" {
		tmpl, ok := s.registry.Templates.Get(templateName)
		if !ok {
			return "", hysterr.TemplateNotFound(templateName)
		}
		source := tmpl.Source
		if source != "" && !filepath.IsAbs(source) && source[0] != '~' {
			source = filepath.Join(s.configDir, source)
		}
		content, err := os.ReadFile(source)
		if err != nil {
			return "", fmt.Errorf("reading template %q: %w", source, err)
		}
		buf.Write(content)
		buf.WriteString("\n\n")
	}

	promptSources := s.resolvePromptSources(promptNames)
	for _, ps := range promptSources {
		content, err := os.ReadFile(ps)
		if err != nil {
			return "", fmt.Errorf("reading prompt %q: %w", ps, err)
		}
		buf.Write(content)
		buf.WriteString("\n\n")
	}

	return buf.String(), nil
}

// SyncAll syncs all projects and returns results keyed by project name.
func (s *Service) SyncAll() (map[string][]SyncResult, error) {
	all := make(map[string][]SyncResult)
	for _, proj := range s.projects.List() {
		results, err := s.SyncProject(proj.Name)
		if err != nil {
			return nil, fmt.Errorf("syncing project %q: %w", proj.Name, err)
		}
		all[proj.Name] = results
	}
	return all, nil
}

// PreflightSync detects resource conflicts for a project without writing anything.
// Returns conflicts for skills, hooks, permissions, and CLAUDE.md that exist in
// the project directory but were not placed by hystak.
func (s *Service) PreflightSync(projectName string) ([]SyncConflict, error) {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return nil, hysterr.ProjectNotFound(projectName)
	}

	cfg, err := s.resolveEffectiveConfig(proj)
	if err != nil {
		return nil, err
	}

	var conflicts []SyncConflict

	// Check skills.
	if s.skillsDeployer != nil && len(cfg.skills) > 0 {
		skills := make([]model.SkillDef, 0, len(cfg.skills))
		for _, name := range cfg.skills {
			if skill, ok := s.registry.Skills.Get(name); ok {
				skills = append(skills, skill)
			}
		}
		for _, c := range s.skillsDeployer.PreflightSkills(proj.Path, skills) {
			conflicts = append(conflicts, SyncConflict{
				ResourceType: c.ResourceType,
				Name:         c.Name,
				ExistingPath: c.ExistingPath,
				Resolution:   ConflictPending,
			})
		}
	}

	// Check settings (hooks/permissions).
	if s.settingsDeployer != nil {
		hooks := make([]model.HookDef, 0, len(cfg.hooks))
		for _, name := range cfg.hooks {
			if hook, ok := s.registry.Hooks.Get(name); ok {
				hooks = append(hooks, hook)
			}
		}
		permissions := make([]model.PermissionRule, 0, len(cfg.permissions))
		for _, name := range cfg.permissions {
			if perm, ok := s.registry.Permissions.Get(name); ok {
				permissions = append(permissions, perm)
			}
		}
		for _, c := range s.settingsDeployer.PreflightSettings(proj.Path, hooks, permissions) {
			conflicts = append(conflicts, SyncConflict{
				ResourceType: c.ResourceType,
				Name:         c.Name,
				ExistingPath: c.ExistingPath,
				Resolution:   ConflictPending,
			})
		}
	}

	// Check CLAUDE.md.
	if s.claudeMDDeployer != nil && cfg.claudeMD != "" {
		if c := s.claudeMDDeployer.PreflightClaudeMD(proj.Path); c != nil {
			conflicts = append(conflicts, SyncConflict{
				ResourceType: c.ResourceType,
				Name:         c.Name,
				ExistingPath: c.ExistingPath,
				Resolution:   ConflictPending,
			})
		}
	}

	return conflicts, nil
}

// DriftReport compares expected (registry+overrides) against deployed servers
// for each client in a project, returning per-server drift status.
func (s *Service) DriftReport(projectName string) ([]model.ServerDriftReport, error) {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return nil, hysterr.ProjectNotFound(projectName)
	}

	cfg, err := s.resolveEffectiveConfig(proj)
	if err != nil {
		return nil, err
	}

	expected, err := s.resolveExpectedServers(cfg)
	if err != nil {
		return nil, err
	}

	var reports []model.ServerDriftReport

	for _, ct := range proj.Clients {
		deployer, ok := s.deployers[ct]
		if !ok {
			continue
		}

		deployed, err := deployer.ReadServers(proj.Path)
		if err != nil {
			// Config doesn't exist: all expected servers are missing.
			for name, exp := range expected {
				expCopy := exp
				reports = append(reports, model.ServerDriftReport{
					ServerName: name,
					Status:     model.DriftMissing,
					Expected:   &expCopy,
					Deployed:   nil,
				})
			}
			continue
		}

		for name, exp := range expected {
			expCopy := exp
			dep, ok := deployed[name]
			if !ok {
				reports = append(reports, model.ServerDriftReport{
					ServerName: name,
					Status:     model.DriftMissing,
					Expected:   &expCopy,
					Deployed:   nil,
				})
			} else {
				depCopy := dep
				if exp.Equal(dep) {
					reports = append(reports, model.ServerDriftReport{
						ServerName: name,
						Status:     model.DriftSynced,
						Expected:   &expCopy,
						Deployed:   &depCopy,
					})
				} else {
					reports = append(reports, model.ServerDriftReport{
						ServerName: name,
						Status:     model.DriftDrifted,
						Expected:   &expCopy,
						Deployed:   &depCopy,
					})
				}
			}
		}

		// Flag unmanaged servers.
		for name, dep := range deployed {
			if _, isExpected := expected[name]; !isExpected {
				depCopy := dep
				reports = append(reports, model.ServerDriftReport{
					ServerName: name,
					Status:     model.DriftUnmanaged,
					Expected:   nil,
					Deployed:   &depCopy,
				})
			}
		}
	}

	sort.Slice(reports, func(i, j int) bool {
		return reports[i].ServerName < reports[j].ServerName
	})

	return reports, nil
}

// DriftReportAll returns drift reports for all projects.
func (s *Service) DriftReportAll() (map[string][]model.ServerDriftReport, error) {
	all := make(map[string][]model.ServerDriftReport)
	for _, proj := range s.projects.List() {
		reports, err := s.DriftReport(proj.Name)
		if err != nil {
			return nil, fmt.Errorf("drift report for project %q: %w", proj.Name, err)
		}
		all[proj.Name] = reports
	}
	return all, nil
}

// ImportFromFile reads servers from a client config file and returns import candidates.
// Candidates include conflict status when a server name already exists in the registry.
func (s *Service) ImportFromFile(configPath string) ([]ImportCandidate, error) {
	ct, projectPath, err := detectClientType(configPath)
	if err != nil {
		return nil, err
	}

	deployer, ok := s.deployers[ct]
	if !ok {
		return nil, fmt.Errorf("no deployer for client %q", ct)
	}

	servers, err := deployer.ReadServers(projectPath)
	if err != nil {
		return nil, fmt.Errorf("reading servers from %s: %w", configPath, err)
	}

	var candidates []ImportCandidate
	for name, srv := range servers {
		_, conflict := s.registry.Servers.Get(name)
		candidates = append(candidates, ImportCandidate{
			Name:       name,
			Server:     srv,
			Conflict:   conflict,
			Resolution: ImportPending,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Name < candidates[j].Name
	})

	return candidates, nil
}

// ApplyImport adds imported servers to the registry based on their resolution.
// Non-conflicting servers are added directly. Conflicting servers are handled
// according to their Resolution field.
func (s *Service) ApplyImport(candidates []ImportCandidate) error {
	for _, c := range candidates {
		if c.Conflict {
			switch c.Resolution {
			case ImportKeep, ImportSkip:
				continue
			case ImportReplace:
				if err := s.registry.Servers.Update(c.Name, c.Server); err != nil {
					return fmt.Errorf("replacing server %q: %w", c.Name, err)
				}
			case ImportRename:
				srv := c.Server
				srv.Name = c.RenameTo
				if err := s.registry.Servers.Add(srv); err != nil {
					return fmt.Errorf("adding renamed server %q: %w", c.RenameTo, err)
				}
			default:
				continue // unresolved conflicts are skipped
			}
		} else {
			if err := s.registry.Servers.Add(c.Server); err != nil {
				return fmt.Errorf("adding server %q: %w", c.Name, err)
			}
		}
	}

	return s.saveRegistry()
}

// Diff generates a unified diff string between deployed and expected server configs
// for each client in a project.
func (s *Service) Diff(projectName string) (string, error) {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return "", hysterr.ProjectNotFound(projectName)
	}

	cfg, err := s.resolveEffectiveConfig(proj)
	if err != nil {
		return "", err
	}

	expected, err := s.resolveExpectedServers(cfg)
	if err != nil {
		return "", err
	}

	var diffs []string

	for _, ct := range proj.Clients {
		deployer, ok := s.deployers[ct]
		if !ok {
			continue
		}

		deployed, err := deployer.ReadServers(proj.Path)
		if err != nil {
			deployed = make(map[string]model.ServerDef)
		}

		deployedJSON := serversToJSON(deployed)
		expectedJSON := serversToJSON(expected)

		if deployedJSON != expectedJSON {
			configPath := deployer.ConfigPath(proj.Path)
			diff := unifiedDiff(
				strings.Split(deployedJSON, "\n"),
				strings.Split(expectedJSON, "\n"),
				"deployed: "+configPath,
				"expected: "+configPath,
			)
			diffs = append(diffs, diff)
		}
	}

	return strings.Join(diffs, "\n"), nil
}

// detectClientType determines the client type and project path from a config file path.
func detectClientType(configPath string) (model.ClientType, string, error) {
	base := filepath.Base(configPath)
	dir := filepath.Dir(configPath)

	switch base {
	case ".mcp.json":
		return model.ClientClaudeCode, dir, nil
	case ".claude.json":
		return model.ClientClaudeCode, "", nil
	default:
		return "", "", fmt.Errorf("cannot determine client type from file %q", configPath)
	}
}

// serversToJSON formats a server map as deterministic pretty-printed JSON for diffing.
func serversToJSON(servers map[string]model.ServerDef) string {
	type jsonServer struct {
		Type    string            `json:"type"`
		Command string            `json:"command,omitempty"`
		Args    []string          `json:"args,omitempty"`
		Env     map[string]string `json:"env,omitempty"`
		URL     string            `json:"url,omitempty"`
		Headers map[string]string `json:"headers,omitempty"`
	}

	out := make(map[string]jsonServer, len(servers))
	for name, srv := range servers {
		out[name] = jsonServer{
			Type:    string(srv.Transport),
			Command: srv.Command,
			Args:    srv.Args,
			Env:     srv.Env,
			URL:     srv.URL,
			Headers: srv.Headers,
		}
	}

	data, _ := json.MarshalIndent(out, "", "  ")
	return string(data)
}
