package service

import (
	"github.com/lcrostarosa/hystak/internal/backup"
	"github.com/lcrostarosa/hystak/internal/deploy"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/profile"
	"github.com/lcrostarosa/hystak/internal/project"
	"github.com/lcrostarosa/hystak/internal/registry"
)

// NewForTest constructs a Service from pre-built components.
// Intended for use in tests outside the service package.
func NewForTest(
	reg *registry.Registry,
	store *project.Store,
	deployers map[model.ClientType]deploy.Deployer,
	backups *backup.Manager,
	configDir string,
	profiles *profile.Manager,
) *Service {
	return &Service{
		registry:  reg,
		projects:  store,
		deployers: deployers,
		resourceDeployers: []deploy.ResourceDeployer{
			&deploy.SkillsDeployer{},
			&deploy.SettingsDeployer{},
			&deploy.ClaudeMDDeployer{},
		},
		profiles:  profiles,
		backups:   backups,
		configDir: configDir,
	}
}
