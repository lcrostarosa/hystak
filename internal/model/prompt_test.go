package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPromptDef_YAMLRoundTrip(t *testing.T) {
	original := PromptDef{
		Name:        "defensive-security",
		Description: "Defensive-only security guardrails",
		Source:      "prompts/defensive-security.md",
		Tags:        []string{"security", "guardrail"},
		Category:    "safety",
		Order:       10,
	}

	// Marshal (Name should be excluded via yaml:"-").
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// Verify Name is not in YAML output.
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal raw: %v", err)
	}
	if _, ok := raw["name"]; ok {
		t.Error("Name should not appear in YAML output (has yaml:\"-\" tag)")
	}

	// Unmarshal back.
	var decoded PromptDef
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// Name is yaml:"-", so it won't survive round-trip.
	if decoded.Description != original.Description {
		t.Errorf("Description = %q, want %q", decoded.Description, original.Description)
	}
	if decoded.Source != original.Source {
		t.Errorf("Source = %q, want %q", decoded.Source, original.Source)
	}
	if decoded.Category != original.Category {
		t.Errorf("Category = %q, want %q", decoded.Category, original.Category)
	}
	if decoded.Order != original.Order {
		t.Errorf("Order = %d, want %d", decoded.Order, original.Order)
	}
	if len(decoded.Tags) != len(original.Tags) {
		t.Fatalf("Tags len = %d, want %d", len(decoded.Tags), len(original.Tags))
	}
	for i, tag := range decoded.Tags {
		if tag != original.Tags[i] {
			t.Errorf("Tags[%d] = %q, want %q", i, tag, original.Tags[i])
		}
	}
}

func TestPromptDef_YAMLOmitsEmpty(t *testing.T) {
	minimal := PromptDef{
		Source: "prompts/basic.md",
	}

	data, err := yaml.Marshal(minimal)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal raw: %v", err)
	}

	// Only source should be present.
	if _, ok := raw["source"]; !ok {
		t.Error("source should be present")
	}
	for _, key := range []string{"description", "tags", "category", "order"} {
		if _, ok := raw[key]; ok {
			t.Errorf("%s should be omitted when empty/zero", key)
		}
	}
}
