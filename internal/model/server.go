package model

// Transport is the MCP server transport protocol.
type Transport string

const (
	TransportStdio Transport = "stdio"
	TransportSSE   Transport = "sse"
	TransportHTTP  Transport = "http"
)

// Valid reports whether t is a known transport.
func (t Transport) Valid() bool {
	switch t {
	case TransportStdio, TransportSSE, TransportHTTP:
		return true
	}
	return false
}

// ServerDef is the canonical MCP server definition stored in the registry.
type ServerDef struct {
	Name        string            `yaml:"name,omitempty"`
	Transport   Transport         `yaml:"transport"`
	Command     string            `yaml:"command,omitempty"`
	Args        []string          `yaml:"args,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	URL         string            `yaml:"url,omitempty"`
	Headers     map[string]string `yaml:"headers,omitempty"`
	Description string            `yaml:"description,omitempty"`
}

func (s *ServerDef) ResourceName() string     { return s.Name }
func (s *ServerDef) SetResourceName(n string) { s.Name = n }

// Equal compares deployment-relevant fields only. Name and Description are
// registry metadata and deliberately excluded so drift detection is semantic.
func (a ServerDef) Equal(b ServerDef) bool {
	return a.Transport == b.Transport &&
		a.Command == b.Command &&
		a.URL == b.URL &&
		slicesEqualNil(a.Args, b.Args) &&
		mapsEqualNil(a.Env, b.Env) &&
		mapsEqualNil(a.Headers, b.Headers)
}

// ServerOverride holds per-project overrides applied during sync.
// Nil/zero fields are ignored during merge.
type ServerOverride struct {
	Command *string           `yaml:"command,omitempty"`
	Args    []string          `yaml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
	URL     *string           `yaml:"url,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

// Apply merges the override into a copy of base and returns the result.
// Merge rules: command/url replace if non-nil; args replace entirely;
// env/headers map-merge with override keys winning.
func (o *ServerOverride) Apply(base ServerDef) ServerDef {
	result := base
	if o == nil {
		return result
	}
	if o.Command != nil {
		result.Command = *o.Command
	}
	if o.URL != nil {
		result.URL = *o.URL
	}
	if o.Args != nil {
		result.Args = make([]string, len(o.Args))
		copy(result.Args, o.Args)
	}
	if o.Env != nil {
		merged := make(map[string]string, len(base.Env)+len(o.Env))
		for k, v := range base.Env {
			merged[k] = v
		}
		for k, v := range o.Env {
			merged[k] = v
		}
		result.Env = merged
	}
	if o.Headers != nil {
		merged := make(map[string]string, len(base.Headers)+len(o.Headers))
		for k, v := range base.Headers {
			merged[k] = v
		}
		for k, v := range o.Headers {
			merged[k] = v
		}
		result.Headers = merged
	}
	return result
}
