package service

import (
	"fmt"

	"github.com/lcrostarosa/hystak/internal/backup"
	hysterr "github.com/lcrostarosa/hystak/internal/errors"
)

// BackupConfigs backs up all client configs for a project.
func (s *Service) BackupConfigs(projectName string) ([]backup.BackupEntry, error) {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return nil, hysterr.ProjectNotFound(projectName)
	}

	var entries []backup.BackupEntry
	for _, ct := range proj.Clients {
		deployer, ok := s.deployers[ct]
		if !ok {
			continue
		}
		configPath := deployer.ConfigPath(proj.Path)
		entry, err := s.backups.Create(ct, proj.Path, configPath)
		if err != nil {
			return nil, fmt.Errorf("backing up %s config for project %q: %w", ct, projectName, err)
		}
		if entry.BackupPath != "" {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

// ListBackups lists backups for a project's clients.
// Populates SourcePath using the deployer so entries are ready for restore.
func (s *Service) ListBackups(projectName string) ([]backup.BackupEntry, error) {
	proj, ok := s.projects.Get(projectName)
	if !ok {
		return nil, hysterr.ProjectNotFound(projectName)
	}

	var entries []backup.BackupEntry
	for _, ct := range proj.Clients {
		list, err := s.backups.List(ct, proj.Path)
		if err != nil {
			return nil, err
		}
		deployer := s.deployers[ct]
		for i := range list {
			if deployer != nil {
				list[i].SourcePath = deployer.ConfigPath(proj.Path)
			}
		}
		entries = append(entries, list...)
	}

	return entries, nil
}

// ListAllBackups lists all backups across all scopes.
func (s *Service) ListAllBackups() ([]backup.BackupEntry, error) {
	return s.backups.ListAll()
}

// RestoreBackup restores a backup entry.
func (s *Service) RestoreBackup(entry backup.BackupEntry) error {
	return s.backups.Restore(entry)
}
