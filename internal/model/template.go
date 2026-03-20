package model

// TemplateDef represents a CLAUDE.md template in the registry.
type TemplateDef struct {
	Name   string `yaml:"-"`
	Source string `yaml:"source"` // path to .md template file
}
