package deploy

import (
	"github.com/lcrostarosa/hystak/internal/model"
)

// ResourceDeployerKind identifies the type of resource a deployer handles.
type ResourceDeployerKind string

const (
	DeployerKindSkill    ResourceDeployerKind = "skill"
	DeployerKindSettings ResourceDeployerKind = "settings"
	DeployerKindClaudeMD ResourceDeployerKind = "claude_md"
)

// DeployConfig carries the resolved resource data for a sync operation.
// Each deployer reads only the fields it cares about.
type DeployConfig struct {
	Skills         []model.SkillDef
	Hooks          []model.HookDef
	Permissions    []model.PermissionRule
	TemplateSource string
	PromptSources  []string
}

// ResourceDeployer is the unified interface for deploying non-MCP resources.
// MCP servers remain handled by the existing Deployer interface (per-client).
type ResourceDeployer interface {
	// Kind identifies what this deployer handles.
	Kind() ResourceDeployerKind

	// Sync deploys resources to the target project path.
	Sync(projectPath string, config DeployConfig) error

	// Preflight checks for conflicts without writing anything.
	Preflight(projectPath string, config DeployConfig) []PreflightConflict

	// ReadDeployed reads currently deployed resources from the project directory.
	// Used for two-way sync to detect local changes.
	ReadDeployed(projectPath string) (DeployConfig, error)
}
