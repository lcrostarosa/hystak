package model

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSkillDef_ResourceName(t *testing.T) {
	s := &SkillDef{Name: "code-review"}
	if got := s.ResourceName(); got != "code-review" {
		t.Errorf("ResourceName() = %q, want %q", got, "code-review")
	}
	s.SetResourceName("commit")
	if got := s.ResourceName(); got != "commit" {
		t.Errorf("after SetResourceName: ResourceName() = %q, want %q", got, "commit")
	}
}

func TestSkillDef_YAMLRoundTrip(t *testing.T) {
	original := SkillDef{
		Name:        "code-review",
		Description: "Structured code review skill",
		Source:      "/path/to/SKILL.md",
	}
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored SkillDef
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original, restored) {
		t.Errorf("round-trip mismatch:\n  got:  %+v\n  want: %+v", restored, original)
	}
}
