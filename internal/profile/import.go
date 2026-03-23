package profile

import "github.com/hystak/hystak/internal/model"

// ImportedProfile is the YAML structure for profile import/export.
// It mirrors ProjectProfile but is used for the CLI import path.
type ImportedProfile struct {
	Name        string                  `yaml:"name"`
	Description string                  `yaml:"description,omitempty"`
	Scope       string                  `yaml:"scope,omitempty"`
	MCPs        []model.MCPAssignment   `yaml:"mcps,omitempty"`
	Skills      []string                `yaml:"skills,omitempty"`
	Hooks       []string                `yaml:"hooks,omitempty"`
	Permissions []string                `yaml:"permissions,omitempty"`
	Template    string                  `yaml:"template,omitempty"`
	Prompts     []string                `yaml:"prompts,omitempty"`
	Env         map[string]string       `yaml:"env,omitempty"`
	Tags        []string                `yaml:"tags,omitempty"`
	Isolation   model.IsolationStrategy `yaml:"isolation,omitempty"`
}

// ToProjectProfile converts to the internal model type.
func (p ImportedProfile) ToProjectProfile() model.ProjectProfile {
	return model.ProjectProfile{
		Name:        p.Name,
		Description: p.Description,
		Scope:       p.Scope,
		MCPs:        p.MCPs,
		Skills:      p.Skills,
		Hooks:       p.Hooks,
		Permissions: p.Permissions,
		Template:    p.Template,
		Prompts:     p.Prompts,
		Env:         p.Env,
		Tags:        p.Tags,
		Isolation:   p.Isolation,
	}
}
