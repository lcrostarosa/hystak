package model

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestTemplateDef_ResourceName(t *testing.T) {
	tmpl := &TemplateDef{Name: "standard"}
	if got := tmpl.ResourceName(); got != "standard" {
		t.Errorf("ResourceName() = %q, want %q", got, "standard")
	}
	tmpl.SetResourceName("minimal")
	if got := tmpl.ResourceName(); got != "minimal" {
		t.Errorf("after SetResourceName: ResourceName() = %q, want %q", got, "minimal")
	}
}

func TestTemplateDef_YAMLRoundTrip(t *testing.T) {
	original := TemplateDef{
		Name:   "standard",
		Source: "/path/to/template.md",
	}
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored TemplateDef
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original, restored) {
		t.Errorf("round-trip mismatch:\n  got:  %+v\n  want: %+v", restored, original)
	}
}
