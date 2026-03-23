package model

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestHookEvent_Valid(t *testing.T) {
	tests := []struct {
		name  string
		value HookEvent
		want  bool
	}{
		{"PreToolUse", HookEventPreToolUse, true},
		{"PostToolUse", HookEventPostToolUse, true},
		{"Notification", HookEventNotification, true},
		{"Stop", HookEventStop, true},
		{"empty", HookEvent(""), false},
		{"unknown", HookEvent("OnStart"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.Valid(); got != tt.want {
				t.Errorf("HookEvent(%q).Valid() = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestHookDef_ResourceName(t *testing.T) {
	h := &HookDef{Name: "lint-on-edit"}
	if got := h.ResourceName(); got != "lint-on-edit" {
		t.Errorf("ResourceName() = %q, want %q", got, "lint-on-edit")
	}
	h.SetResourceName("block-rm")
	if got := h.ResourceName(); got != "block-rm" {
		t.Errorf("after SetResourceName: ResourceName() = %q, want %q", got, "block-rm")
	}
}

func TestHookDef_YAMLRoundTrip(t *testing.T) {
	original := HookDef{
		Name:    "lint-on-edit",
		Event:   HookEventPostToolUse,
		Matcher: "Edit",
		Command: "npm run lint --fix",
		Timeout: 30,
	}
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored HookDef
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original, restored) {
		t.Errorf("round-trip mismatch:\n  got:  %+v\n  want: %+v", restored, original)
	}
}
