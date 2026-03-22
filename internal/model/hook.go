package model

// HookEvent is the lifecycle event that triggers a hook.
type HookEvent string

const (
	HookEventPreToolUse   HookEvent = "PreToolUse"
	HookEventPostToolUse  HookEvent = "PostToolUse"
	HookEventNotification HookEvent = "Notification"
	HookEventStop         HookEvent = "Stop"
)

// Valid reports whether e is a known hook event.
func (e HookEvent) Valid() bool {
	switch e {
	case HookEventPreToolUse, HookEventPostToolUse, HookEventNotification, HookEventStop:
		return true
	}
	return false
}

// HookDef is a hook definition stored in the registry.
type HookDef struct {
	Name    string    `yaml:"name,omitempty"`
	Event   HookEvent `yaml:"event"`
	Matcher string    `yaml:"matcher,omitempty"`
	Command string    `yaml:"command"`
	Timeout int       `yaml:"timeout,omitempty"`
}

func (h *HookDef) ResourceName() string     { return h.Name }
func (h *HookDef) SetResourceName(n string) { h.Name = n }
