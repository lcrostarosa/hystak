package model

// TemplateDef represents a CLAUDE.md template in the registry.
type TemplateDef struct {
	Name   string `yaml:"-"`
	Source string `yaml:"source"` // path to .md template file
}

func (t *TemplateDef) ResourceName() string    { return t.Name }
func (t *TemplateDef) SetResourceName(n string) { t.Name = n }
