package service

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rbbydotdev/hystak/internal/deploy"
	"github.com/rbbydotdev/hystak/internal/model"
	"github.com/rbbydotdev/hystak/internal/project"
	"github.com/rbbydotdev/hystak/internal/registry"
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

// Service orchestrates registry, projects, and deployers.
type Service struct {
	Registry  *registry.Registry
	Projects  *project.Store
	Deployers map[model.ClientType]deploy.Deployer
	ConfigDir string
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
		Registry:  reg,
		Projects:  proj,
		Deployers: deployers,
		ConfigDir: configDir,
	}, nil
}

// SaveRegistry writes the registry back to disk.
func (s *Service) SaveRegistry() error {
	return s.Registry.Save(filepath.Join(s.ConfigDir, "registry.yaml"))
}

// SaveProjects writes the project store back to disk.
func (s *Service) SaveProjects() error {
	return s.Projects.Save(filepath.Join(s.ConfigDir, "projects.yaml"))
}

// SyncProject resolves servers for a project and writes them to each configured client.
// Unmanaged servers (in client config but not in hystak) are preserved.
func (s *Service) SyncProject(projectName string) ([]SyncResult, error) {
	proj, ok := s.Projects.Get(projectName)
	if !ok {
		return nil, fmt.Errorf("project %q not found", projectName)
	}

	resolved, err := s.Projects.ResolveServers(projectName, s.Registry)
	if err != nil {
		return nil, err
	}

	expected := make(map[string]model.ServerDef, len(resolved))
	for _, srv := range resolved {
		expected[srv.Name] = srv
	}

	var results []SyncResult

	for _, ct := range proj.Clients {
		deployer, ok := s.Deployers[ct]
		if !ok {
			return nil, fmt.Errorf("no deployer for client %q", ct)
		}

		if err := deployer.Bootstrap(proj.Path); err != nil {
			return nil, fmt.Errorf("bootstrapping %s for project %q: %w", ct, projectName, err)
		}

		deployed, err := deployer.ReadServers(proj.Path)
		if err != nil {
			return nil, fmt.Errorf("reading deployed servers for %s in project %q: %w", ct, projectName, err)
		}

		merged := make(map[string]model.ServerDef, len(deployed)+len(expected))

		// Preserve unmanaged servers.
		for name, srv := range deployed {
			if _, isManaged := expected[name]; !isManaged {
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
				if serversEqual(prev, srv) {
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
			return nil, fmt.Errorf("writing servers for %s in project %q: %w", ct, projectName, err)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ServerName < results[j].ServerName
	})

	return results, nil
}

// SyncAll syncs all projects and returns results keyed by project name.
func (s *Service) SyncAll() (map[string][]SyncResult, error) {
	all := make(map[string][]SyncResult)
	for _, proj := range s.Projects.List() {
		results, err := s.SyncProject(proj.Name)
		if err != nil {
			return nil, fmt.Errorf("syncing project %q: %w", proj.Name, err)
		}
		all[proj.Name] = results
	}
	return all, nil
}

// DriftReport compares expected (registry+overrides) against deployed servers
// for each client in a project, returning per-server drift status.
func (s *Service) DriftReport(projectName string) ([]model.ServerDriftReport, error) {
	proj, ok := s.Projects.Get(projectName)
	if !ok {
		return nil, fmt.Errorf("project %q not found", projectName)
	}

	resolved, err := s.Projects.ResolveServers(projectName, s.Registry)
	if err != nil {
		return nil, err
	}

	expected := make(map[string]model.ServerDef, len(resolved))
	for _, srv := range resolved {
		expected[srv.Name] = srv
	}

	var reports []model.ServerDriftReport

	for _, ct := range proj.Clients {
		deployer, ok := s.Deployers[ct]
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
				if serversEqual(exp, dep) {
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
	for _, proj := range s.Projects.List() {
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

	deployer, ok := s.Deployers[ct]
	if !ok {
		return nil, fmt.Errorf("no deployer for client %q", ct)
	}

	servers, err := deployer.ReadServers(projectPath)
	if err != nil {
		return nil, fmt.Errorf("reading servers from %s: %w", configPath, err)
	}

	var candidates []ImportCandidate
	for name, srv := range servers {
		_, conflict := s.Registry.Get(name)
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
				if err := s.Registry.Update(c.Name, c.Server); err != nil {
					return fmt.Errorf("replacing server %q: %w", c.Name, err)
				}
			case ImportRename:
				srv := c.Server
				srv.Name = c.RenameTo
				if err := s.Registry.Add(srv); err != nil {
					return fmt.Errorf("adding renamed server %q: %w", c.RenameTo, err)
				}
			default:
				continue // unresolved conflicts are skipped
			}
		} else {
			if err := s.Registry.Add(c.Server); err != nil {
				return fmt.Errorf("adding server %q: %w", c.Name, err)
			}
		}
	}

	return s.SaveRegistry()
}

// Diff generates a unified diff string between deployed and expected server configs
// for each client in a project.
func (s *Service) Diff(projectName string) (string, error) {
	proj, ok := s.Projects.Get(projectName)
	if !ok {
		return "", fmt.Errorf("project %q not found", projectName)
	}

	resolved, err := s.Projects.ResolveServers(projectName, s.Registry)
	if err != nil {
		return "", err
	}

	expected := make(map[string]model.ServerDef, len(resolved))
	for _, srv := range resolved {
		expected[srv.Name] = srv
	}

	var diffs []string

	for _, ct := range proj.Clients {
		deployer, ok := s.Deployers[ct]
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

// serversEqual performs semantic comparison of two server definitions.
// Compares transport, command, args, env, url, and headers.
// Ignores Name and Description (registry-only metadata).
func serversEqual(a, b model.ServerDef) bool {
	if a.Transport != b.Transport {
		return false
	}
	if a.Command != b.Command {
		return false
	}
	if a.URL != b.URL {
		return false
	}
	if !sliceEqual(a.Args, b.Args) {
		return false
	}
	if !mapEqual(a.Env, b.Env) {
		return false
	}
	if !mapEqual(a.Headers, b.Headers) {
		return false
	}
	return true
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func mapEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
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
