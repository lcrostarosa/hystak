package model

// SkillDef is a skill definition stored in the registry.
type SkillDef struct {
	Name        string `yaml:"name,omitempty"`
	Description string `yaml:"description,omitempty"`
	Source      string `yaml:"source"`
}

func (s *SkillDef) ResourceName() string     { return s.Name }
func (s *SkillDef) SetResourceName(n string) { s.Name = n }
