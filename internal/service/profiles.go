package service

import (
	"fmt"

	"github.com/hystak/hystak/internal/model"
)

// ListProfileNames returns all available profile names.
func (s *Service) ListProfileNames() ([]string, error) {
	return s.profiles.List()
}

// LoadProfile reads a profile by name.
func (s *Service) LoadProfile(name string) (model.ProjectProfile, error) {
	return s.profiles.Load(name)
}

// SaveProfile writes a profile to disk.
func (s *Service) SaveProfile(prof model.ProjectProfile) error {
	return s.profiles.Save(prof)
}

// ToggleMCP adds or removes an MCP assignment in a profile.
// Returns true if the MCP is now assigned, false if removed.
func (s *Service) ToggleMCP(profileName, mcpName string) (bool, error) {
	prof, err := s.profiles.Load(profileName)
	if err != nil {
		return false, err
	}

	for i, a := range prof.MCPs {
		if a.Name == mcpName {
			prof.MCPs = append(prof.MCPs[:i], prof.MCPs[i+1:]...)
			return false, s.profiles.Save(prof)
		}
	}

	prof.MCPs = append(prof.MCPs, model.MCPAssignment{Name: mcpName})
	return true, s.profiles.Save(prof)
}

// ToggleStringSlice adds or removes a name from a string slice in a profile.
// kind is "skills", "hooks", "permissions", or "prompts".
func (s *Service) ToggleProfileResource(profileName, kind, resourceName string) (bool, error) {
	prof, err := s.profiles.Load(profileName)
	if err != nil {
		return false, err
	}

	var slice *[]string
	switch kind {
	case "skills":
		slice = &prof.Skills
	case "hooks":
		slice = &prof.Hooks
	case "permissions":
		slice = &prof.Permissions
	case "prompts":
		slice = &prof.Prompts
	default:
		return false, fmt.Errorf("unknown resource kind %q", kind)
	}

	for i, n := range *slice {
		if n == resourceName {
			*slice = append((*slice)[:i], (*slice)[i+1:]...)
			return false, s.profiles.Save(prof)
		}
	}

	*slice = append(*slice, resourceName)
	return true, s.profiles.Save(prof)
}

// SetProfileTemplate sets or clears the template for a profile.
func (s *Service) SetProfileTemplate(profileName, templateName string) error {
	prof, err := s.profiles.Load(profileName)
	if err != nil {
		return err
	}
	prof.Template = templateName
	return s.profiles.Save(prof)
}
