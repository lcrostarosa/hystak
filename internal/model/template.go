package model

// TemplateDef is a CLAUDE.md template definition stored in the registry.
type TemplateDef struct {
	Name   string `yaml:"name,omitempty"`
	Source string `yaml:"source"`
}

func (t *TemplateDef) ResourceName() string     { return t.Name }
func (t *TemplateDef) SetResourceName(n string) { t.Name = n }
