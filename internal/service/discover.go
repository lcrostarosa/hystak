package service

import (
	"fmt"
	"os"

	"github.com/hystak/hystak/internal/discovery"
)

// AutoDiscover scans the project path for unregistered MCP servers
// and imports them silently into the registry (S-007).
// Discovery errors are non-blocking: they are reported to stderr
// but do not halt execution (S-008).
func (s *Service) AutoDiscover(projectPath string) ([]discovery.Candidate, error) {
	candidates, err := discovery.ScanAll(projectPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: auto-discovery: %v\n", err)
		return nil, nil
	}

	var imported []discovery.Candidate
	for _, c := range candidates {
		if _, exists := s.registry.Servers.Get(c.Name); exists {
			continue
		}
		if err := s.registry.Servers.Add(c.Server); err != nil {
			fmt.Fprintf(os.Stderr, "warning: auto-import %q: %v\n", c.Name, err)
			continue
		}
		imported = append(imported, c)
	}

	return imported, nil
}
