package catalog

import (
	_ "embed"

	"github.com/lcrostarosa/hystak/internal/model"
	"gopkg.in/yaml.v3"
)

//go:embed catalog.yaml
var catalogData []byte

// MCPEntry is a catalog entry for an MCP server.
type MCPEntry struct {
	model.ServerDef `yaml:",inline"`
	Category        string `yaml:"category,omitempty"`
	Popular         bool   `yaml:"popular,omitempty"`
}

// SkillEntry is a catalog entry for a skill with inline content.
type SkillEntry struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Category    string `yaml:"category,omitempty"`
	Content     string `yaml:"content"`
}

// HookEntry is a catalog entry for a hook.
type HookEntry struct {
	model.HookDef `yaml:",inline"`
	Category      string `yaml:"category,omitempty"`
}

// PermissionEntry is a catalog entry for a permission rule.
type PermissionEntry struct {
	model.PermissionRule `yaml:",inline"`
	Category             string `yaml:"category,omitempty"`
}

// Catalog holds all curated entries bundled with the binary.
type Catalog struct {
	MCPs        []MCPEntry        `yaml:"mcps"`
	Skills      []SkillEntry      `yaml:"skills"`
	Hooks       []HookEntry       `yaml:"hooks"`
	Permissions []PermissionEntry `yaml:"permissions"`
}

// Load parses the embedded catalog data.
func Load() Catalog {
	var c Catalog
	if err := yaml.Unmarshal(catalogData, &c); err != nil {
		panic("catalog: failed to parse embedded catalog.yaml: " + err.Error())
	}
	return c
}
