package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSkillDef_YAMLRoundTrip(t *testing.T) {
	original := SkillDef{
		Name:        "code-review",
		Description: "Reviews pull requests",
		Source:      "skills/code-review.md",
	}

	out, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got SkillDef
	if err := yaml.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Name has yaml:"-" so it should NOT survive the round-trip
	if got.Name != "" {
		t.Errorf("Name = %q, want empty (yaml:\"-\" tag should exclude it)", got.Name)
	}
	if got.Description != original.Description {
		t.Errorf("Description = %q, want %q", got.Description, original.Description)
	}
	if got.Source != original.Source {
		t.Errorf("Source = %q, want %q", got.Source, original.Source)
	}
}

func TestHookDef_YAMLRoundTrip(t *testing.T) {
	original := HookDef{
		Name:    "lint-check",
		Event:   "PreToolUse",
		Matcher: "Bash",
		Command: "/usr/local/bin/lint.sh",
		Timeout: 5000,
	}

	out, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got HookDef
	if err := yaml.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Name has yaml:"-" so it should NOT survive the round-trip
	if got.Name != "" {
		t.Errorf("Name = %q, want empty (yaml:\"-\" tag should exclude it)", got.Name)
	}
	if got.Event != original.Event {
		t.Errorf("Event = %q, want %q", got.Event, original.Event)
	}
	if got.Matcher != original.Matcher {
		t.Errorf("Matcher = %q, want %q", got.Matcher, original.Matcher)
	}
	if got.Command != original.Command {
		t.Errorf("Command = %q, want %q", got.Command, original.Command)
	}
	if got.Timeout != original.Timeout {
		t.Errorf("Timeout = %d, want %d", got.Timeout, original.Timeout)
	}
}

func TestTemplateDef_YAMLRoundTrip(t *testing.T) {
	original := TemplateDef{
		Name:   "go-service",
		Source: "templates/go-service.md",
	}

	out, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got TemplateDef
	if err := yaml.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Name has yaml:"-" so it should NOT survive the round-trip
	if got.Name != "" {
		t.Errorf("Name = %q, want empty (yaml:\"-\" tag should exclude it)", got.Name)
	}
	if got.Source != original.Source {
		t.Errorf("Source = %q, want %q", got.Source, original.Source)
	}
}
