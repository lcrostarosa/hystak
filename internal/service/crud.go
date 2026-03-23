package service

import (
	"errors"
	"fmt"
	"slices"

	"github.com/hystak/hystak/internal/model"
)

// --- Servers ---

// AddServer registers a new MCP server in the registry and persists to disk.
func (s *Service) AddServer(srv model.ServerDef) error {
	if err := s.registry.Servers.Add(srv); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// UpdateServer replaces an existing MCP server in the registry and persists to disk.
func (s *Service) UpdateServer(srv model.ServerDef) error {
	if err := s.registry.Servers.Update(srv); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// DeleteServer removes an MCP server from the registry, cascades to profiles, and persists to disk.
func (s *Service) DeleteServer(name string) error {
	if err := s.registry.Servers.Delete(name); err != nil {
		return err
	}
	if err := s.deleteServerCascade(name); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// GetServer retrieves a single MCP server by name.
func (s *Service) GetServer(name string) (model.ServerDef, bool) {
	return s.registry.Servers.Get(name)
}

// --- Skills ---

// ListSkills returns all registered skills sorted by name.
func (s *Service) ListSkills() []model.SkillDef {
	return s.registry.Skills.List()
}

// AddSkill registers a new skill and persists to disk.
func (s *Service) AddSkill(skill model.SkillDef) error {
	if err := s.registry.Skills.Add(skill); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// UpdateSkill replaces an existing skill and persists to disk.
func (s *Service) UpdateSkill(skill model.SkillDef) error {
	if err := s.registry.Skills.Update(skill); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// DeleteSkill removes a skill, cascades to profiles, and persists to disk.
func (s *Service) DeleteSkill(name string) error {
	if err := s.registry.Skills.Delete(name); err != nil {
		return err
	}
	if err := s.cascadeDeleteFromProfiles(func(p *model.ProjectProfile) {
		p.Skills = slices.DeleteFunc(p.Skills, func(n string) bool { return n == name })
	}); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// GetSkill retrieves a single skill by name.
func (s *Service) GetSkill(name string) (model.SkillDef, bool) {
	return s.registry.Skills.Get(name)
}

// --- Hooks ---

// ListHooks returns all registered hooks sorted by name.
func (s *Service) ListHooks() []model.HookDef {
	return s.registry.Hooks.List()
}

// AddHook registers a new hook and persists to disk.
func (s *Service) AddHook(hook model.HookDef) error {
	if err := s.registry.Hooks.Add(hook); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// UpdateHook replaces an existing hook and persists to disk.
func (s *Service) UpdateHook(hook model.HookDef) error {
	if err := s.registry.Hooks.Update(hook); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// DeleteHook removes a hook, cascades to profiles, and persists to disk.
func (s *Service) DeleteHook(name string) error {
	if err := s.registry.Hooks.Delete(name); err != nil {
		return err
	}
	if err := s.cascadeDeleteFromProfiles(func(p *model.ProjectProfile) {
		p.Hooks = slices.DeleteFunc(p.Hooks, func(n string) bool { return n == name })
	}); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// GetHook retrieves a single hook by name.
func (s *Service) GetHook(name string) (model.HookDef, bool) {
	return s.registry.Hooks.Get(name)
}

// --- Permissions ---

// ListPermissions returns all registered permission rules sorted by name.
func (s *Service) ListPermissions() []model.PermissionRule {
	return s.registry.Permissions.List()
}

// AddPermission registers a new permission rule and persists to disk.
func (s *Service) AddPermission(perm model.PermissionRule) error {
	if err := s.registry.Permissions.Add(perm); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// UpdatePermission replaces an existing permission rule and persists to disk.
func (s *Service) UpdatePermission(perm model.PermissionRule) error {
	if err := s.registry.Permissions.Update(perm); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// DeletePermission removes a permission rule, cascades to profiles, and persists to disk.
func (s *Service) DeletePermission(name string) error {
	if err := s.registry.Permissions.Delete(name); err != nil {
		return err
	}
	if err := s.cascadeDeleteFromProfiles(func(p *model.ProjectProfile) {
		p.Permissions = slices.DeleteFunc(p.Permissions, func(n string) bool { return n == name })
	}); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// GetPermission retrieves a single permission rule by name.
func (s *Service) GetPermission(name string) (model.PermissionRule, bool) {
	return s.registry.Permissions.Get(name)
}

// --- Templates ---

// ListTemplates returns all registered templates sorted by name.
func (s *Service) ListTemplates() []model.TemplateDef {
	return s.registry.Templates.List()
}

// AddTemplate registers a new template and persists to disk.
func (s *Service) AddTemplate(tmpl model.TemplateDef) error {
	if err := s.registry.Templates.Add(tmpl); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// UpdateTemplate replaces an existing template and persists to disk.
func (s *Service) UpdateTemplate(tmpl model.TemplateDef) error {
	if err := s.registry.Templates.Update(tmpl); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// DeleteTemplate removes a template, cascades to profiles, and persists to disk.
func (s *Service) DeleteTemplate(name string) error {
	if err := s.registry.Templates.Delete(name); err != nil {
		return err
	}
	if err := s.cascadeDeleteFromProfiles(func(p *model.ProjectProfile) {
		if p.Template == name {
			p.Template = ""
		}
	}); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// GetTemplate retrieves a single template by name.
func (s *Service) GetTemplate(name string) (model.TemplateDef, bool) {
	return s.registry.Templates.Get(name)
}

// --- Prompts ---

// ListPrompts returns all registered prompt fragments sorted by order.
func (s *Service) ListPrompts() []model.PromptDef {
	return s.registry.Prompts.List()
}

// AddPrompt registers a new prompt fragment and persists to disk.
func (s *Service) AddPrompt(prompt model.PromptDef) error {
	if err := s.registry.Prompts.Add(prompt); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// UpdatePrompt replaces an existing prompt fragment and persists to disk.
func (s *Service) UpdatePrompt(prompt model.PromptDef) error {
	if err := s.registry.Prompts.Update(prompt); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// DeletePrompt removes a prompt fragment, cascades to profiles, and persists to disk.
func (s *Service) DeletePrompt(name string) error {
	if err := s.registry.Prompts.Delete(name); err != nil {
		return err
	}
	if err := s.cascadeDeleteFromProfiles(func(p *model.ProjectProfile) {
		p.Prompts = slices.DeleteFunc(p.Prompts, func(n string) bool { return n == name })
	}); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// GetPrompt retrieves a single prompt fragment by name.
func (s *Service) GetPrompt(name string) (model.PromptDef, bool) {
	return s.registry.Prompts.Get(name)
}

// --- Cascade helpers ---

// cascadeDeleteFromProfiles applies a mutation to all profiles, removing
// references to deleted resources (S-018). Returns an aggregated error
// if any profile load/save fails (CS-1: never swallow errors).
func (s *Service) cascadeDeleteFromProfiles(mutate func(p *model.ProjectProfile)) error {
	names, err := s.profiles.List()
	if err != nil {
		return fmt.Errorf("listing profiles for cascade: %w", err)
	}
	var errs []error
	for _, name := range names {
		prof, err := s.profiles.Load(name)
		if err != nil {
			errs = append(errs, fmt.Errorf("loading profile %q: %w", name, err))
			continue
		}
		mutate(&prof)
		if err := s.profiles.Save(prof); err != nil {
			errs = append(errs, fmt.Errorf("saving profile %q: %w", name, err))
		}
	}
	return errors.Join(errs...)
}

// deleteServerCascade removes MCP references from all profiles.
func (s *Service) deleteServerCascade(name string) error {
	return s.cascadeDeleteFromProfiles(func(p *model.ProjectProfile) {
		p.MCPs = slices.DeleteFunc(p.MCPs, func(a model.MCPAssignment) bool { return a.Name == name })
	})
}
