package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMCPAssignment_UnmarshalBareString(t *testing.T) {
	input := `- github
- filesystem
`
	var assignments []MCPAssignment
	if err := yaml.Unmarshal([]byte(input), &assignments); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(assignments) != 2 {
		t.Fatalf("len = %d, want 2", len(assignments))
	}
	if assignments[0].Name != "github" {
		t.Errorf("[0].Name = %q, want %q", assignments[0].Name, "github")
	}
	if assignments[0].Overrides != nil {
		t.Errorf("[0].Overrides = %v, want nil", assignments[0].Overrides)
	}
	if assignments[1].Name != "filesystem" {
		t.Errorf("[1].Name = %q, want %q", assignments[1].Name, "filesystem")
	}
}

func TestMCPAssignment_UnmarshalMapWithOverrides(t *testing.T) {
	input := `- qdrant:
    overrides:
        env:
            COLLECTION_NAME: agent-context
`
	var assignments []MCPAssignment
	if err := yaml.Unmarshal([]byte(input), &assignments); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(assignments) != 1 {
		t.Fatalf("len = %d, want 1", len(assignments))
	}
	a := assignments[0]
	if a.Name != "qdrant" {
		t.Errorf("Name = %q, want %q", a.Name, "qdrant")
	}
	if a.Overrides == nil {
		t.Fatal("Overrides is nil, want non-nil")
	}
	if a.Overrides.Env["COLLECTION_NAME"] != "agent-context" {
		t.Errorf("Overrides.Env[COLLECTION_NAME] = %q, want %q", a.Overrides.Env["COLLECTION_NAME"], "agent-context")
	}
}

func TestMCPAssignment_MarshalBareString(t *testing.T) {
	assignments := []MCPAssignment{
		{Name: "github"},
		{Name: "filesystem"},
	}

	out, err := yaml.Marshal(assignments)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Unmarshal back to verify round-trip
	var got []MCPAssignment
	if err := yaml.Unmarshal(out, &got); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Name != "github" || got[1].Name != "filesystem" {
		t.Errorf("round-trip names: %q, %q", got[0].Name, got[1].Name)
	}
	if got[0].Overrides != nil || got[1].Overrides != nil {
		t.Errorf("bare string round-trip should have nil overrides")
	}
}

func TestMCPAssignment_MarshalMapWithOverrides(t *testing.T) {
	assignments := []MCPAssignment{
		{
			Name: "qdrant",
			Overrides: &ServerOverride{
				Env: map[string]string{"COLLECTION_NAME": "agent-context"},
			},
		},
	}

	out, err := yaml.Marshal(assignments)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got []MCPAssignment
	if err := yaml.Unmarshal(out, &got); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Name != "qdrant" {
		t.Errorf("Name = %q, want %q", got[0].Name, "qdrant")
	}
	if got[0].Overrides == nil || got[0].Overrides.Env["COLLECTION_NAME"] != "agent-context" {
		t.Errorf("Overrides round-trip failed: %+v", got[0].Overrides)
	}
}

func TestMCPAssignment_MixedFormats(t *testing.T) {
	input := `- github
- qdrant:
    overrides:
        env:
            COLLECTION_NAME: agent-context
- filesystem
`
	var assignments []MCPAssignment
	if err := yaml.Unmarshal([]byte(input), &assignments); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(assignments) != 3 {
		t.Fatalf("len = %d, want 3", len(assignments))
	}

	// github: bare
	if assignments[0].Name != "github" || assignments[0].Overrides != nil {
		t.Errorf("[0] = %+v, want bare 'github'", assignments[0])
	}
	// qdrant: with overrides
	if assignments[1].Name != "qdrant" || assignments[1].Overrides == nil {
		t.Errorf("[1] = %+v, want 'qdrant' with overrides", assignments[1])
	}
	if assignments[1].Overrides.Env["COLLECTION_NAME"] != "agent-context" {
		t.Errorf("[1].Overrides.Env = %v", assignments[1].Overrides.Env)
	}
	// filesystem: bare
	if assignments[2].Name != "filesystem" || assignments[2].Overrides != nil {
		t.Errorf("[2] = %+v, want bare 'filesystem'", assignments[2])
	}

	// Round-trip the mixed format
	out, err := yaml.Marshal(assignments)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got []MCPAssignment
	if err := yaml.Unmarshal(out, &got); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("round-trip len = %d, want 3", len(got))
	}
	if got[0].Name != "github" || got[1].Name != "qdrant" || got[2].Name != "filesystem" {
		t.Errorf("round-trip names: %q, %q, %q", got[0].Name, got[1].Name, got[2].Name)
	}
}

func TestProjectYAMLRoundTrip(t *testing.T) {
	// Test Project serialization as a value within a map (simulating projects.yaml structure)
	type projectsFile struct {
		Projects map[string]Project `yaml:"projects"`
	}

	input := `projects:
    agents:
        path: /workspace/agents
        clients:
            - claude-code
        tags:
            - core
        mcps:
            - qdrant:
                overrides:
                    env:
                        COLLECTION_NAME: agent-context
    hystak:
        path: /workspace/hystak
        clients:
            - claude-code
        mcps:
            - github
            - filesystem
`
	var pf projectsFile
	if err := yaml.Unmarshal([]byte(input), &pf); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Verify agents project
	agents, ok := pf.Projects["agents"]
	if !ok {
		t.Fatal("missing 'agents' project")
	}
	if agents.Path != "/workspace/agents" {
		t.Errorf("agents.Path = %q", agents.Path)
	}
	if len(agents.Clients) != 1 || agents.Clients[0] != ClientClaudeCode {
		t.Errorf("agents.Clients = %v", agents.Clients)
	}
	if len(agents.Tags) != 1 || agents.Tags[0] != "core" {
		t.Errorf("agents.Tags = %v", agents.Tags)
	}
	if len(agents.MCPs) != 1 || agents.MCPs[0].Name != "qdrant" {
		t.Errorf("agents.MCPs = %+v", agents.MCPs)
	}
	if agents.MCPs[0].Overrides == nil || agents.MCPs[0].Overrides.Env["COLLECTION_NAME"] != "agent-context" {
		t.Errorf("agents.MCPs[0].Overrides = %+v", agents.MCPs[0].Overrides)
	}

	// Verify hystak project
	hystak, ok := pf.Projects["hystak"]
	if !ok {
		t.Fatal("missing 'hystak' project")
	}
	if len(hystak.MCPs) != 2 {
		t.Fatalf("hystak.MCPs len = %d, want 2", len(hystak.MCPs))
	}
	if hystak.MCPs[0].Name != "github" || hystak.MCPs[1].Name != "filesystem" {
		t.Errorf("hystak.MCPs = %+v", hystak.MCPs)
	}

	// Round-trip
	out, err := yaml.Marshal(&pf)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var pf2 projectsFile
	if err := yaml.Unmarshal(out, &pf2); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	agents2 := pf2.Projects["agents"]
	if len(agents2.MCPs) != 1 || agents2.MCPs[0].Name != "qdrant" {
		t.Errorf("round-trip agents.MCPs = %+v", agents2.MCPs)
	}
	hystak2 := pf2.Projects["hystak"]
	if len(hystak2.MCPs) != 2 || hystak2.MCPs[0].Name != "github" {
		t.Errorf("round-trip hystak.MCPs = %+v", hystak2.MCPs)
	}
}

func TestDriftStatusConstants(t *testing.T) {
	// Verify the constants have expected values
	tests := []struct {
		status DriftStatus
		want   string
	}{
		{DriftSynced, "synced"},
		{DriftDrifted, "drifted"},
		{DriftMissing, "missing"},
		{DriftUnmanaged, "unmanaged"},
	}
	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("DriftStatus = %q, want %q", tt.status, tt.want)
		}
	}
}
