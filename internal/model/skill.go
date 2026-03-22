package model

// SkillDef represents a skill definition in the registry.
type SkillDef struct {
	Name        string `yaml:"-"`
	Description string `yaml:"description,omitempty"`
	Source      string `yaml:"source"` // path to .md file in hystak config
}

func (s *SkillDef) ResourceName() string    { return s.Name }
func (s *SkillDef) SetResourceName(n string) { s.Name = n }
