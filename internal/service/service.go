package service

import (
	"fmt"
	"sort"

	"github.com/hystak/hystak/internal/deploy"
	hysterr "github.com/hystak/hystak/internal/errors"
	"github.com/hystak/hystak/internal/model"
	"github.com/hystak/hystak/internal/profile"
	"github.com/hystak/hystak/internal/project"
	"github.com/hystak/hystak/internal/registry"
)

// SyncAction describes what happened to a server during sync.
type SyncAction string

const (
	SyncAdded     SyncAction = "added"
	SyncUpdated   SyncAction = "updated"
	SyncUnchanged SyncAction = "unchanged"
	SyncRemoved   SyncAction = "removed"
	SyncUnmanaged SyncAction = "unmanaged"
)

// SyncResult is the outcome of syncing a single server.
type SyncResult struct {
	Name   string
	Action SyncAction
}

// Service orchestrates sync, diff, import, and discovery operations.
type Service struct {
	Registry *registry.Registry
	Projects *project.Store
	Profiles *profile.Manager
	deployer deploy.Deployer
}

// New creates a new Service.
func New(reg *registry.Registry, proj *project.Store, prof *profile.Manager, dep deploy.Deployer) *Service {
	return &Service{
		Registry: reg,
		Projects: proj,
		Profiles: prof,
		deployer: dep,
	}
}

// SyncProject syncs the active profile for a project (S-033).
// It resolves the profile, looks up servers, applies overrides,
// deploys to the client config, and returns per-server results.
func (s *Service) SyncProject(projectName string) ([]SyncResult, error) {
	proj, ok := s.Projects.Get(projectName)
	if !ok {
		return nil, &hysterr.ProjectNotFound{Name: projectName}
	}

	if proj.ActiveProfile == "" {
		return nil, fmt.Errorf("project %q has no active profile", projectName)
	}

	prof, err := s.Profiles.Load(proj.ActiveProfile)
	if err != nil {
		return nil, fmt.Errorf("loading profile %q: %w", proj.ActiveProfile, err)
	}

	// Resolve servers from profile MCPs
	resolved, err := s.resolveServers(prof)
	if err != nil {
		return nil, err
	}

	// Bootstrap the config file if needed
	if err := s.deployer.Bootstrap(proj.Path); err != nil {
		return nil, fmt.Errorf("bootstrapping config for %q: %w", projectName, err)
	}

	// Read current deployed state
	deployed, err := s.deployer.ReadServers(proj.Path)
	if err != nil {
		return nil, fmt.Errorf("reading deployed servers for %q: %w", projectName, err)
	}

	// Build sync results before writing
	results := s.buildSyncResults(resolved, deployed, proj.ManagedMCPs)

	// Merge: start with unmanaged servers, then overlay resolved managed servers
	merged := s.mergeServers(resolved, deployed, proj.ManagedMCPs)

	// Write the merged server map
	if err := s.deployer.WriteServers(proj.Path, merged); err != nil {
		return nil, fmt.Errorf("writing servers for %q: %w", projectName, err)
	}

	// Update managed MCPs tracking (sorted for deterministic YAML output)
	newManagedNames := make([]string, 0, len(resolved))
	for name := range resolved {
		newManagedNames = append(newManagedNames, name)
	}
	sort.Strings(newManagedNames)
	if err := s.Projects.SetManagedMCPs(projectName, newManagedNames); err != nil {
		return nil, err
	}

	return results, nil
}

// resolveServers looks up each MCP in the profile, applies overrides, and
// returns the final server map. Returns an error if any server is not in
// the registry (S-041).
func (s *Service) resolveServers(prof model.ProjectProfile) (map[string]model.ServerDef, error) {
	result := make(map[string]model.ServerDef, len(prof.MCPs))

	for _, assignment := range prof.MCPs {
		srv, ok := s.Registry.Servers.Get(assignment.Name)
		if !ok {
			return nil, &hysterr.ServerNotFound{Name: assignment.Name}
		}
		resolved := assignment.Overrides.Apply(srv)
		result[assignment.Name] = resolved
	}

	return result, nil
}

// mergeServers combines resolved managed servers with unmanaged servers from
// the deployed config. Previously managed servers not in resolved are removed (S-039).
// Unmanaged servers are preserved (S-038).
func (s *Service) mergeServers(resolved, deployed map[string]model.ServerDef, previouslyManaged []string) map[string]model.ServerDef {
	prevSet := make(map[string]bool, len(previouslyManaged))
	for _, name := range previouslyManaged {
		prevSet[name] = true
	}

	merged := make(map[string]model.ServerDef, len(resolved)+len(deployed))

	// Keep unmanaged servers (not previously managed by hystak)
	for name, srv := range deployed {
		if !prevSet[name] {
			if _, isResolved := resolved[name]; !isResolved {
				merged[name] = srv
			}
		}
	}

	// Add all resolved managed servers
	for name, srv := range resolved {
		merged[name] = srv
	}

	return merged
}

// buildSyncResults computes per-server sync actions by comparing resolved
// (what we want) against deployed (what's on disk) and previouslyManaged
// (what hystak previously owned).
func (s *Service) buildSyncResults(resolved, deployed map[string]model.ServerDef, previouslyManaged []string) []SyncResult {
	var results []SyncResult

	// Check resolved servers against deployed
	for name, want := range resolved {
		existing, wasDeployed := deployed[name]
		if !wasDeployed {
			results = append(results, SyncResult{Name: name, Action: SyncAdded})
		} else if want.Equal(existing) {
			results = append(results, SyncResult{Name: name, Action: SyncUnchanged})
		} else {
			results = append(results, SyncResult{Name: name, Action: SyncUpdated})
		}
	}

	// Check previously managed servers that are no longer in the profile
	prevSet := make(map[string]bool, len(previouslyManaged))
	for _, name := range previouslyManaged {
		prevSet[name] = true
	}
	for name := range prevSet {
		if _, stillManaged := resolved[name]; !stillManaged {
			results = append(results, SyncResult{Name: name, Action: SyncRemoved})
		}
	}

	// Report unmanaged servers (in deployed but never managed by hystak)
	for name := range deployed {
		if _, isResolved := resolved[name]; isResolved {
			continue
		}
		if prevSet[name] {
			continue // already reported as removed
		}
		results = append(results, SyncResult{Name: name, Action: SyncUnmanaged})
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })
	return results
}
