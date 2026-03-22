package model

// PermissionRule represents a permission rule in the registry.
type PermissionRule struct {
	Name string `yaml:"-"`
	Rule string `yaml:"rule"` // e.g., "Bash(*)", "WebFetch(domain:github.com)"
	Type string `yaml:"type,omitempty"` // "allow" (default) or "deny"
}

func (p *PermissionRule) ResourceName() string    { return p.Name }
func (p *PermissionRule) SetResourceName(n string) { p.Name = n }

// EffectiveType returns the permission type, defaulting to "allow" if empty.
func (p PermissionRule) EffectiveType() string {
	if p.Type == "" {
		return "allow"
	}
	return p.Type
}
