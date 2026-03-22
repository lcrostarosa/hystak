package service

import (
	"fmt"
	"sort"

	hysterr "github.com/hystak/hystak/internal/errors"
	"github.com/hystak/hystak/internal/model"
)

// DiffResult describes the drift state of a single server.
type DiffResult struct {
	ServerName string
	Status     model.DriftStatus
	Expected   model.ServerDef
	Deployed   model.ServerDef
}

// DiffProject compares expected vs deployed servers for a project (S-049).
func (s *Service) DiffProject(name string) ([]DiffResult, error) {
	proj, ok := s.projects.Get(name)
	if !ok {
		return nil, &hysterr.ProjectNotFound{Name: name}
	}

	if proj.ActiveProfile == "" {
		return nil, fmt.Errorf("project %q has no active profile", name)
	}

	prof, err := s.profiles.Load(proj.ActiveProfile)
	if err != nil {
		return nil, fmt.Errorf("loading profile %q: %w", proj.ActiveProfile, err)
	}

	resolved, err := s.resolveServers(prof)
	if err != nil {
		return nil, err
	}

	deployed, err := s.deployer.ReadServers(proj.Path)
	if err != nil {
		return nil, fmt.Errorf("reading deployed servers for %q: %w", name, err)
	}

	return buildDiffResults(resolved, deployed, proj.ManagedMCPs), nil
}

// ListProjects returns all registered projects.
func (s *Service) ListProjects() []model.Project {
	return s.projects.List()
}

// buildDiffResults compares resolved (expected) against deployed servers and
// returns per-server drift results. Uses ServerDef.Equal for semantic
// comparison (S-052).
func buildDiffResults(resolved, deployed map[string]model.ServerDef, managedMCPs []string) []DiffResult {
	prevSet := make(map[string]bool, len(managedMCPs))
	for _, name := range managedMCPs {
		prevSet[name] = true
	}

	var results []DiffResult

	// Check expected servers against deployed
	for name, want := range resolved {
		existing, wasDeployed := deployed[name]
		switch {
		case !wasDeployed:
			results = append(results, DiffResult{
				ServerName: name,
				Status:     model.DriftMissing,
				Expected:   want,
			})
		case want.Equal(existing):
			results = append(results, DiffResult{
				ServerName: name,
				Status:     model.DriftSynced,
				Expected:   want,
				Deployed:   existing,
			})
		default:
			results = append(results, DiffResult{
				ServerName: name,
				Status:     model.DriftDrifted,
				Expected:   want,
				Deployed:   existing,
			})
		}
	}

	// Report unmanaged servers (in deployed but not in resolved)
	for name, srv := range deployed {
		if _, isResolved := resolved[name]; isResolved {
			continue
		}
		results = append(results, DiffResult{
			ServerName: name,
			Status:     model.DriftUnmanaged,
			Deployed:   srv,
		})
	}

	sort.Slice(results, func(i, j int) bool { return results[i].ServerName < results[j].ServerName })
	return results
}
