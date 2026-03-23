package service

import "github.com/hystak/hystak/internal/model"

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

// DeleteServer removes an MCP server from the registry and persists to disk.
// Cascade: unassigns from all profiles is handled at a higher layer.
func (s *Service) DeleteServer(name string) error {
	if err := s.registry.Servers.Delete(name); err != nil {
		return err
	}
	return s.registry.SaveDefault()
}

// GetServer retrieves a single MCP server by name.
func (s *Service) GetServer(name string) (model.ServerDef, bool) {
	return s.registry.Servers.Get(name)
}
