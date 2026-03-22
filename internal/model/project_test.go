package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestClientType_Valid(t *testing.T) {
	tests := []struct {
		name  string
		value ClientType
		want  bool
	}{
		{"claude-code", ClientClaudeCode, true},
		{"claude-desktop", ClientClaudeDesktop, true},
		{"cursor", ClientCursor, true},
		{"empty", ClientType(""), false},
		{"unknown", ClientType("vscode"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.Valid(); got != tt.want {
				t.Errorf("ClientType(%q).Valid() = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestIsolationStrategy_Valid(t *testing.T) {
	tests := []struct {
		name  string
		value IsolationStrategy
		want  bool
	}{
		{"none", IsolationNone, true},
		{"worktree", IsolationWorktree, true},
		{"lock", IsolationLock, true},
		{"empty", IsolationStrategy(""), false},
		{"unknown", IsolationStrategy("docker"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.Valid(); got != tt.want {
				t.Errorf("IsolationStrategy(%q).Valid() = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestProject_ResourceName(t *testing.T) {
	p := &Project{Name: "myproject"}
	if got := p.ResourceName(); got != "myproject" {
		t.Errorf("ResourceName() = %q, want %q", got, "myproject")
	}
	p.SetResourceName("other")
	if got := p.ResourceName(); got != "other" {
		t.Errorf("after SetResourceName: ResourceName() = %q, want %q", got, "other")
	}
}

func TestMCPAssignment_YAMLRoundTrip_BareString(t *testing.T) {
	original := MCPAssignment{Name: "github"}
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored MCPAssignment
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Name != original.Name {
		t.Errorf("Name: got %q, want %q", restored.Name, original.Name)
	}
	if restored.Overrides != nil {
		t.Errorf("Overrides should be nil for bare string, got %+v", restored.Overrides)
	}
}

func TestMCPAssignment_YAMLRoundTrip_WithOverrides(t *testing.T) {
	cmd := "node"
	original := MCPAssignment{
		Name: "github",
		Overrides: &ServerOverride{
			Command: &cmd,
			Env:     map[string]string{"KEY": "val"},
		},
	}
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored MCPAssignment
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Name != original.Name {
		t.Errorf("Name: got %q, want %q", restored.Name, original.Name)
	}
	if restored.Overrides == nil {
		t.Fatal("Overrides should not be nil")
	}
	if restored.Overrides.Command == nil || *restored.Overrides.Command != cmd {
		t.Errorf("Overrides.Command: got %v, want %q", restored.Overrides.Command, cmd)
	}
	if restored.Overrides.Env["KEY"] != "val" {
		t.Errorf("Overrides.Env[KEY]: got %q, want %q", restored.Overrides.Env["KEY"], "val")
	}
}

func TestMCPAssignment_UnmarshalYAML_InvalidKind(t *testing.T) {
	// A YAML sequence should fail
	input := "- a\n- b\n"
	var a MCPAssignment
	err := yaml.Unmarshal([]byte(input), &a)
	if err == nil {
		t.Error("expected error for sequence node, got nil")
	}
}

func TestMCPAssignment_MarshalYAML_ListRoundTrip(t *testing.T) {
	// A list of MCPAssignments should round-trip correctly
	original := []MCPAssignment{
		{Name: "github"},
		{Name: "postgres", Overrides: &ServerOverride{Env: map[string]string{"DB": "test"}}},
	}
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored []MCPAssignment
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if len(restored) != 2 {
		t.Fatalf("got %d assignments, want 2", len(restored))
	}
	if restored[0].Name != "github" {
		t.Errorf("[0].Name = %q, want %q", restored[0].Name, "github")
	}
	if restored[0].Overrides != nil {
		t.Errorf("[0].Overrides should be nil")
	}
	if restored[1].Name != "postgres" {
		t.Errorf("[1].Name = %q, want %q", restored[1].Name, "postgres")
	}
	if restored[1].Overrides == nil || restored[1].Overrides.Env["DB"] != "test" {
		t.Errorf("[1].Overrides.Env[DB] mismatch")
	}
}
