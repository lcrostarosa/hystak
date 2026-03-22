package model

// HookDef represents a hook definition in the registry.
type HookDef struct {
	Name    string `yaml:"-"`
	Event   string `yaml:"event"`             // PreToolUse, PostToolUse, UserPromptSubmit, etc.
	Matcher string `yaml:"matcher,omitempty"`  // e.g., "Bash", optional
	Command string `yaml:"command"`
	Timeout int    `yaml:"timeout,omitempty"` // ms
}

func (h *HookDef) ResourceName() string    { return h.Name }
func (h *HookDef) SetResourceName(n string) { h.Name = n }
