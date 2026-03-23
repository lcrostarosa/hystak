package service

import (
	"fmt"

	"github.com/hystak/hystak/internal/discovery"
	"github.com/hystak/hystak/internal/model"
)

// ImportResolution describes how to handle an import candidate.
type ImportResolution int

const (
	ImportPending ImportResolution = iota
	ImportKeep                     // keep existing registry entry
	ImportReplace                  // replace with imported
	ImportRename                   // import under a new name
	ImportSkip                     // skip this candidate
)

// ImportCandidate is a discovered server with conflict info and resolution.
type ImportCandidate struct {
	discovery.Candidate
	Conflict   bool
	Resolution ImportResolution
	RenameTo   string // set when Resolution == ImportRename
}

// PrepareImport scans a file and marks candidates that conflict with
// existing registry entries.
func (s *Service) PrepareImport(path string) ([]ImportCandidate, error) {
	candidates, err := discovery.ScanFile(path)
	if err != nil {
		return nil, fmt.Errorf("scanning %q: %w", path, err)
	}

	result := make([]ImportCandidate, len(candidates))
	for i, c := range candidates {
		_, exists := s.registry.Servers.Get(c.Name)
		result[i] = ImportCandidate{
			Candidate:  c,
			Conflict:   exists,
			Resolution: ImportPending,
		}
	}
	return result, nil
}

// ApplyImport processes resolved import candidates. Each candidate must have
// a Resolution set (not ImportPending). Returns the count of imported servers.
func (s *Service) ApplyImport(candidates []ImportCandidate) (int, error) {
	imported := 0
	for _, c := range candidates {
		switch c.Resolution {
		case ImportKeep, ImportSkip:
			continue
		case ImportReplace:
			srv := c.Server
			srv.Name = c.Name
			if _, exists := s.registry.Servers.Get(c.Name); exists {
				if err := s.registry.Servers.Update(srv); err != nil {
					return imported, fmt.Errorf("replacing %q: %w", c.Name, err)
				}
			} else {
				if err := s.registry.Servers.Add(srv); err != nil {
					return imported, fmt.Errorf("adding %q: %w", c.Name, err)
				}
			}
			imported++
		case ImportRename:
			if c.RenameTo == "" {
				return imported, fmt.Errorf("rename target not set for %q", c.Name)
			}
			srv := c.Server
			srv.Name = c.RenameTo
			if err := s.registry.Servers.Add(srv); err != nil {
				return imported, fmt.Errorf("adding renamed %q as %q: %w", c.Name, c.RenameTo, err)
			}
			imported++
		case ImportPending:
			return imported, fmt.Errorf("unresolved candidate %q", c.Name)
		}
	}

	if imported > 0 {
		if err := s.registry.SaveDefault(); err != nil {
			return imported, fmt.Errorf("saving registry: %w", err)
		}
	}
	return imported, nil
}

// DiscoverSkills scans a project's .claude/skills/ directory for unregistered
// skill directories (S-011).
func (s *Service) DiscoverSkills(projectPath string) ([]model.SkillDef, error) {
	candidates, err := discovery.ScanSkills(projectPath)
	if err != nil {
		return nil, err
	}

	var unregistered []model.SkillDef
	for _, skill := range candidates {
		if _, exists := s.registry.Skills.Get(skill.Name); !exists {
			unregistered = append(unregistered, skill)
		}
	}
	return unregistered, nil
}
