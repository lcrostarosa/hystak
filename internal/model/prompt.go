package model

// PromptDef is a prompt fragment definition stored in the registry.
type PromptDef struct {
	Name        string   `yaml:"name,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Source      string   `yaml:"source"`
	Category    string   `yaml:"category,omitempty"`
	Order       int      `yaml:"order,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
}

func (p *PromptDef) ResourceName() string     { return p.Name }
func (p *PromptDef) SetResourceName(n string) { p.Name = n }
