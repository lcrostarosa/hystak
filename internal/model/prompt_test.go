package model

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPromptDef_ResourceName(t *testing.T) {
	p := &PromptDef{Name: "security-rules"}
	if got := p.ResourceName(); got != "security-rules" {
		t.Errorf("ResourceName() = %q, want %q", got, "security-rules")
	}
	p.SetResourceName("style-guide")
	if got := p.ResourceName(); got != "style-guide" {
		t.Errorf("after SetResourceName: ResourceName() = %q, want %q", got, "style-guide")
	}
}

func TestPromptDef_YAMLRoundTrip(t *testing.T) {
	original := PromptDef{
		Name:        "security-rules",
		Description: "Security guidelines",
		Source:      "/path/to/security.md",
		Category:    "safety",
		Order:       10,
		Tags:        []string{"security", "default"},
	}
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored PromptDef
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original, restored) {
		t.Errorf("round-trip mismatch:\n  got:  %+v\n  want: %+v", restored, original)
	}
}
