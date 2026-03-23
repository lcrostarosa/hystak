package deploy

import "github.com/hystak/hystak/internal/model"

// ResourceDeployerKind identifies a resource deployer type.
type ResourceDeployerKind string

const (
	ResourceDeployerSkills   ResourceDeployerKind = "skills"
	ResourceDeployerSettings ResourceDeployerKind = "settings"
	ResourceDeployerClaudeMD ResourceDeployerKind = "claude-md"
)

// DeployConfig carries resolved resource data to deployers.
type DeployConfig struct {
	Skills         []model.SkillDef
	Hooks          []model.HookDef
	Permissions    []model.PermissionRule
	TemplateSource string
	PromptSources  []string
}

// PreflightConflict describes a conflict detected before sync.
type PreflightConflict struct {
	Path    string
	Kind    ResourceDeployerKind
	Message string
}

// ResourceDeployer is the interface for non-MCP resource deployment
// (skills, settings, CLAUDE.md). Concrete implementations are P2.
type ResourceDeployer interface {
	Kind() ResourceDeployerKind
	Sync(projectPath string, config DeployConfig) error
	Preflight(projectPath string, config DeployConfig) []PreflightConflict
	ReadDeployed(projectPath string) (DeployConfig, error)
}
