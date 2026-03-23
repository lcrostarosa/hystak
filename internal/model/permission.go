package model

// PermissionType classifies a permission rule as allow or deny.
type PermissionType string

const (
	PermissionAllow PermissionType = "allow"
	PermissionDeny  PermissionType = "deny"
)

// Valid reports whether t is a known permission type.
func (t PermissionType) Valid() bool {
	switch t {
	case PermissionAllow, PermissionDeny:
		return true
	}
	return false
}

// PermissionRule is a permission rule stored in the registry.
type PermissionRule struct {
	Name string         `yaml:"name,omitempty"`
	Rule string         `yaml:"rule"`
	Type PermissionType `yaml:"type"`
}

func (p *PermissionRule) ResourceName() string     { return p.Name }
func (p *PermissionRule) SetResourceName(n string) { p.Name = n }
