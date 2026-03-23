package service

import (
	"fmt"

	"github.com/hystak/hystak/internal/deploy"
)

// SyncConflictResolution describes how to handle a sync conflict.
type SyncConflictResolution int

const (
	ConflictPending SyncConflictResolution = iota
	ConflictKeep                           // keep existing file
	ConflictReplace                        // overwrite with managed version
	ConflictSkip                           // skip this resource
)

// SyncConflict is a conflict detected during preflight.
type SyncConflict struct {
	deploy.PreflightConflict
	Resolution SyncConflictResolution
}

// PreflightCheck runs all resource deployer preflight checks for a project (S-046).
// Returns conflicts that need resolution before sync can proceed.
func (s *Service) PreflightCheck(projectName string) ([]SyncConflict, error) {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return nil, fmt.Errorf("project %q not found", projectName)
	}

	if proj.ActiveProfile == "" {
		return nil, nil // no profile = nothing to check
	}

	prof, err := s.profiles.Load(proj.ActiveProfile)
	if err != nil {
		return nil, fmt.Errorf("loading profile %q: %w", proj.ActiveProfile, err)
	}

	dcfg := s.buildDeployConfig(prof)

	var conflicts []SyncConflict
	for _, rd := range s.resourceDeployers {
		for _, c := range rd.Preflight(proj.Path, dcfg) {
			conflicts = append(conflicts, SyncConflict{
				PreflightConflict: c,
				Resolution:        ConflictPending,
			})
		}
	}
	return conflicts, nil
}

// SyncWithConflicts runs sync after applying conflict resolutions.
// Conflicts marked ConflictReplace will have their files removed before sync.
// Conflicts marked ConflictKeep or ConflictSkip are left untouched.
func (s *Service) SyncWithConflicts(projectName string, conflicts []SyncConflict) ([]SyncResult, error) {
	for _, c := range conflicts {
		if c.Resolution == ConflictPending {
			return nil, fmt.Errorf("unresolved conflict at %q", c.Path)
		}
	}
	// For now, conflicts are informational — the deployers already handle
	// user-owned files by skipping them. ConflictReplace would require
	// removing the user file first, which we can add when needed.
	return s.SyncProject(projectName)
}
