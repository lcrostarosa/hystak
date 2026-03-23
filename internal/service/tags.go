package service

import (
	"fmt"
	"sort"
)

// ListTags returns all tag names and their members.
func (s *Service) ListTags() map[string][]string {
	return s.registry.ListTags()
}

// AddTag creates a new tag with the given members. Validates that all
// members exist in the registry (S-022).
func (s *Service) AddTag(name string, members []string) error {
	if err := s.validateTagMembers(members); err != nil {
		return err
	}
	if err := s.registry.AddTag(name, members); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// UpdateTag replaces a tag's members. Validates members exist (S-022).
func (s *Service) UpdateTag(name string, members []string) error {
	if err := s.validateTagMembers(members); err != nil {
		return err
	}
	if err := s.registry.DeleteTag(name); err != nil {
		return err
	}
	if err := s.registry.AddTag(name, members); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// DeleteTag removes a tag and persists.
func (s *Service) DeleteTag(name string) error {
	if err := s.registry.DeleteTag(name); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// GetTag retrieves a tag's members.
func (s *Service) GetTag(name string) ([]string, bool) {
	return s.registry.GetTag(name)
}

// validateTagMembers checks that all member names exist in the server
// registry (S-022: dangling tag reference error).
func (s *Service) validateTagMembers(members []string) error {
	for _, name := range members {
		if _, ok := s.registry.Servers.Get(name); !ok {
			return fmt.Errorf("tag references non-existent server %q", name)
		}
	}
	return nil
}

// ExpandTags resolves tag references in a profile into additional MCP
// server names (S-021). Returns a deduplicated, sorted list of all MCP
// names (from direct assignments + tag expansion).
func (s *Service) ExpandTags(directNames []string, tagNames []string) ([]string, error) {
	seen := make(map[string]bool, len(directNames))
	for _, name := range directNames {
		seen[name] = true
	}

	for _, tagName := range tagNames {
		members, ok := s.registry.GetTag(tagName)
		if !ok {
			return nil, fmt.Errorf("tag %q not found", tagName)
		}
		for _, member := range members {
			if _, ok := s.registry.Servers.Get(member); !ok {
				return nil, fmt.Errorf("tag %q references non-existent server %q", tagName, member)
			}
			seen[member] = true
		}
	}

	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	sort.Strings(result)
	return result, nil
}
