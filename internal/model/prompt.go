package model

// PromptDef represents a reusable prompt fragment in the registry.
type PromptDef struct {
	Name        string   `yaml:"-"`
	Description string   `yaml:"description,omitempty"`
	Source      string   `yaml:"source"`              // path to .md file in hystak config
	Tags        []string `yaml:"tags,omitempty"`       // categorization labels
	Category    string   `yaml:"category,omitempty"`   // grouping (safety, tone, conventions, etc.)
	Order       int      `yaml:"order,omitempty"`      // composition precedence (lower = earlier)
}

func (p *PromptDef) ResourceName() string    { return p.Name }
func (p *PromptDef) SetResourceName(n string) { p.Name = n }
