package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rbbydotdev/hystak/internal/model"
	"github.com/rbbydotdev/hystak/internal/registry"
)

func TestLoadMissingFile(t *testing.T) {
	s, err := Load("/nonexistent/projects.yaml")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(s.Projects) != 0 {
		t.Fatalf("expected empty store, got %d projects", len(s.Projects))
	}
}

func TestLoadEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.yaml")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(s.Projects) != 0 {
		t.Fatalf("expected empty store, got %d projects", len(s.Projects))
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "projects.yaml")

	cmd := "custom-npx"
	s := &Store{
		Projects: map[string]model.Project{
			"agents": {
				Name:    "agents",
				Path:    "/workspace/agents",
				Clients: []model.ClientType{model.ClientClaudeCode},
				Tags:    []string{"core"},
				MCPs: []model.MCPAssignment{
					{Name: "qdrant", Overrides: &model.ServerOverride{
						Env: map[string]string{"COLLECTION_NAME": "agent-context"},
					}},
				},
			},
			"hystak": {
				Name:    "hystak",
				Path:    "/workspace/hystak",
				Clients: []model.ClientType{model.ClientClaudeCode},
				MCPs: []model.MCPAssignment{
					{Name: "github"},
					{Name: "filesystem"},
				},
			},
		},
	}

	if err := s.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Verify agents project.
	agents, ok := loaded.Get("agents")
	if !ok {
		t.Fatal("expected agents project")
	}
	if agents.Name != "agents" {
		t.Errorf("expected Name=agents, got %q", agents.Name)
	}
	if agents.Path != "/workspace/agents" {
		t.Errorf("expected Path=/workspace/agents, got %q", agents.Path)
	}
	if len(agents.Tags) != 1 || agents.Tags[0] != "core" {
		t.Errorf("expected Tags=[core], got %v", agents.Tags)
	}
	if len(agents.MCPs) != 1 {
		t.Fatalf("expected 1 MCP, got %d", len(agents.MCPs))
	}
	if agents.MCPs[0].Name != "qdrant" {
		t.Errorf("expected MCP name=qdrant, got %q", agents.MCPs[0].Name)
	}
	if agents.MCPs[0].Overrides == nil {
		t.Fatal("expected override on qdrant")
	}
	if agents.MCPs[0].Overrides.Env["COLLECTION_NAME"] != "agent-context" {
		t.Errorf("expected COLLECTION_NAME=agent-context, got %q", agents.MCPs[0].Overrides.Env["COLLECTION_NAME"])
	}

	// Verify hystak project has bare MCPs.
	hystak, ok := loaded.Get("hystak")
	if !ok {
		t.Fatal("expected hystak project")
	}
	if len(hystak.MCPs) != 2 {
		t.Fatalf("expected 2 MCPs, got %d", len(hystak.MCPs))
	}
	if hystak.MCPs[0].Overrides != nil || hystak.MCPs[1].Overrides != nil {
		t.Error("expected bare MCPs with no overrides")
	}

	// Verify override round-trip with command pointer.
	s2 := &Store{
		Projects: map[string]model.Project{
			"test": {
				Name:    "test",
				Path:    "/test",
				Clients: []model.ClientType{model.ClientClaudeCode},
				MCPs: []model.MCPAssignment{
					{Name: "srv", Overrides: &model.ServerOverride{Command: &cmd}},
				},
			},
		},
	}
	path2 := filepath.Join(dir, "projects2.yaml")
	if err := s2.Save(path2); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded2, err := Load(path2)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	testProj, _ := loaded2.Get("test")
	if testProj.MCPs[0].Overrides == nil || testProj.MCPs[0].Overrides.Command == nil {
		t.Fatal("expected command override")
	}
	if *testProj.MCPs[0].Overrides.Command != "custom-npx" {
		t.Errorf("expected command=custom-npx, got %q", *testProj.MCPs[0].Overrides.Command)
	}
}

func TestAddRemove(t *testing.T) {
	s := &Store{Projects: make(map[string]model.Project)}

	proj := model.Project{
		Name:    "myproject",
		Path:    "/my/project",
		Clients: []model.ClientType{model.ClientClaudeCode},
	}

	// Add.
	if err := s.Add(proj); err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Duplicate add.
	if err := s.Add(proj); err == nil {
		t.Fatal("expected error on duplicate add")
	}

	// Get.
	got, ok := s.Get("myproject")
	if !ok {
		t.Fatal("expected project to exist")
	}
	if got.Path != "/my/project" {
		t.Errorf("expected path=/my/project, got %q", got.Path)
	}

	// List.
	list := s.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 project, got %d", len(list))
	}

	// Remove.
	if err := s.Remove("myproject"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Remove non-existent.
	if err := s.Remove("myproject"); err == nil {
		t.Fatal("expected error on remove of non-existent project")
	}
}

func TestAssignUnassign(t *testing.T) {
	s := &Store{Projects: map[string]model.Project{
		"proj": {Name: "proj", Path: "/proj", Clients: []model.ClientType{model.ClientClaudeCode}},
	}}

	// Assign.
	if err := s.Assign("proj", "github"); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	proj, _ := s.Get("proj")
	if len(proj.MCPs) != 1 || proj.MCPs[0].Name != "github" {
		t.Fatalf("expected github assigned, got %v", proj.MCPs)
	}

	// Duplicate assign.
	if err := s.Assign("proj", "github"); err == nil {
		t.Fatal("expected error on duplicate assign")
	}

	// Assign to non-existent project.
	if err := s.Assign("nonexistent", "github"); err == nil {
		t.Fatal("expected error on assign to non-existent project")
	}

	// Unassign.
	if err := s.Unassign("proj", "github"); err != nil {
		t.Fatalf("Unassign: %v", err)
	}
	proj, _ = s.Get("proj")
	if len(proj.MCPs) != 0 {
		t.Fatalf("expected no MCPs, got %v", proj.MCPs)
	}

	// Unassign non-existent server.
	if err := s.Unassign("proj", "github"); err == nil {
		t.Fatal("expected error on unassign of non-existent server")
	}

	// Unassign from non-existent project.
	if err := s.Unassign("nonexistent", "github"); err == nil {
		t.Fatal("expected error on unassign from non-existent project")
	}
}

func TestSetOverride(t *testing.T) {
	s := &Store{Projects: map[string]model.Project{
		"proj": {
			Name:    "proj",
			Path:    "/proj",
			Clients: []model.ClientType{model.ClientClaudeCode},
			MCPs:    []model.MCPAssignment{{Name: "github"}},
		},
	}}

	override := model.ServerOverride{
		Env: map[string]string{"GITHUB_TOKEN": "${ORG_TOKEN}"},
	}

	// Set override on existing assignment.
	if err := s.SetOverride("proj", "github", override); err != nil {
		t.Fatalf("SetOverride: %v", err)
	}
	proj, _ := s.Get("proj")
	if proj.MCPs[0].Overrides == nil {
		t.Fatal("expected override set")
	}
	if proj.MCPs[0].Overrides.Env["GITHUB_TOKEN"] != "${ORG_TOKEN}" {
		t.Errorf("expected env override, got %v", proj.MCPs[0].Overrides.Env)
	}

	// Set override on unassigned server (should add it).
	override2 := model.ServerOverride{
		Env: map[string]string{"KEY": "val"},
	}
	if err := s.SetOverride("proj", "qdrant", override2); err != nil {
		t.Fatalf("SetOverride for new server: %v", err)
	}
	proj, _ = s.Get("proj")
	if len(proj.MCPs) != 2 {
		t.Fatalf("expected 2 MCPs, got %d", len(proj.MCPs))
	}
	if proj.MCPs[1].Name != "qdrant" || proj.MCPs[1].Overrides == nil {
		t.Error("expected qdrant with override added")
	}

	// Set override on non-existent project.
	if err := s.SetOverride("nonexistent", "github", override); err == nil {
		t.Fatal("expected error on non-existent project")
	}
}

func TestSetClients(t *testing.T) {
	s := &Store{Projects: map[string]model.Project{
		"proj": {Name: "proj", Path: "/proj", Clients: []model.ClientType{model.ClientClaudeCode}},
	}}

	newClients := []model.ClientType{model.ClientClaudeCode, model.ClientClaudeDesktop}
	if err := s.SetClients("proj", newClients); err != nil {
		t.Fatalf("SetClients: %v", err)
	}
	proj, _ := s.Get("proj")
	if len(proj.Clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(proj.Clients))
	}

	if err := s.SetClients("nonexistent", newClients); err == nil {
		t.Fatal("expected error on non-existent project")
	}
}

func TestSetTags(t *testing.T) {
	s := &Store{Projects: map[string]model.Project{
		"proj": {Name: "proj", Path: "/proj", Clients: []model.ClientType{model.ClientClaudeCode}},
	}}

	if err := s.SetTags("proj", []string{"core", "data"}); err != nil {
		t.Fatalf("SetTags: %v", err)
	}
	proj, _ := s.Get("proj")
	if len(proj.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(proj.Tags))
	}

	if err := s.SetTags("nonexistent", []string{"core"}); err == nil {
		t.Fatal("expected error on non-existent project")
	}
}

// Helper to create a test registry with known servers and tags.
func testRegistry(t *testing.T) *registry.Registry {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.yaml")

	yaml := `servers:
  github:
    description: "GitHub API"
    transport: stdio
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN}"
  filesystem:
    description: "Filesystem"
    transport: stdio
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/"]
  qdrant:
    description: "Qdrant"
    transport: stdio
    command: uvx
    args: ["mcp-server-qdrant"]
    env:
      QDRANT_URL: "${QDRANT_URL}"
      COLLECTION_NAME: "${COLLECTION_NAME}"
  remote-api:
    description: "Remote API"
    transport: http
    url: "https://mcp.example.com/mcp"
    headers:
      Authorization: "Bearer ${API_TOKEN}"
tags:
  core: [github, filesystem]
  data: [qdrant]
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	reg, err := registry.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func TestResolveServersTagsOnly(t *testing.T) {
	reg := testRegistry(t)
	s := &Store{Projects: map[string]model.Project{
		"proj": {
			Name:    "proj",
			Path:    "/proj",
			Clients: []model.ClientType{model.ClientClaudeCode},
			Tags:    []string{"core"},
		},
	}}

	resolved, err := s.ResolveServers("proj", reg)
	if err != nil {
		t.Fatalf("ResolveServers: %v", err)
	}
	if len(resolved) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(resolved))
	}

	names := map[string]bool{}
	for _, srv := range resolved {
		names[srv.Name] = true
	}
	if !names["github"] || !names["filesystem"] {
		t.Errorf("expected github and filesystem, got %v", names)
	}
}

func TestResolveServersMCPsOnly(t *testing.T) {
	reg := testRegistry(t)
	s := &Store{Projects: map[string]model.Project{
		"proj": {
			Name:    "proj",
			Path:    "/proj",
			Clients: []model.ClientType{model.ClientClaudeCode},
			MCPs: []model.MCPAssignment{
				{Name: "github"},
				{Name: "qdrant"},
			},
		},
	}}

	resolved, err := s.ResolveServers("proj", reg)
	if err != nil {
		t.Fatalf("ResolveServers: %v", err)
	}
	if len(resolved) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(resolved))
	}
	if resolved[0].Name != "github" || resolved[1].Name != "qdrant" {
		t.Errorf("unexpected order: %v, %v", resolved[0].Name, resolved[1].Name)
	}
}

func TestResolveServersTagsPlusMCPsUnion(t *testing.T) {
	reg := testRegistry(t)
	s := &Store{Projects: map[string]model.Project{
		"proj": {
			Name:    "proj",
			Path:    "/proj",
			Clients: []model.ClientType{model.ClientClaudeCode},
			Tags:    []string{"core"},
			MCPs: []model.MCPAssignment{
				{Name: "qdrant"},
				{Name: "github"}, // duplicate from tag — should be deduped
			},
		},
	}}

	resolved, err := s.ResolveServers("proj", reg)
	if err != nil {
		t.Fatalf("ResolveServers: %v", err)
	}
	if len(resolved) != 3 {
		t.Fatalf("expected 3 servers (github, filesystem, qdrant), got %d", len(resolved))
	}

	names := make([]string, len(resolved))
	for i, srv := range resolved {
		names[i] = srv.Name
	}
	// Tags expand first (github, filesystem), then mcps add qdrant.
	// github from mcps is deduped.
	expected := []string{"github", "filesystem", "qdrant"}
	for i, exp := range expected {
		if names[i] != exp {
			t.Errorf("position %d: expected %q, got %q", i, exp, names[i])
		}
	}
}

func TestResolveServersOverrideOnTagExpanded(t *testing.T) {
	reg := testRegistry(t)
	s := &Store{Projects: map[string]model.Project{
		"proj": {
			Name:    "proj",
			Path:    "/proj",
			Clients: []model.ClientType{model.ClientClaudeCode},
			Tags:    []string{"core"},
			MCPs: []model.MCPAssignment{
				{Name: "github", Overrides: &model.ServerOverride{
					Env: map[string]string{"GITHUB_TOKEN": "${ORG_TOKEN}"},
				}},
			},
		},
	}}

	resolved, err := s.ResolveServers("proj", reg)
	if err != nil {
		t.Fatalf("ResolveServers: %v", err)
	}

	// Find github in resolved.
	var github *model.ServerDef
	for i := range resolved {
		if resolved[i].Name == "github" {
			github = &resolved[i]
			break
		}
	}
	if github == nil {
		t.Fatal("expected github in resolved servers")
	}
	if github.Env["GITHUB_TOKEN"] != "${ORG_TOKEN}" {
		t.Errorf("expected GITHUB_TOKEN=${ORG_TOKEN}, got %q", github.Env["GITHUB_TOKEN"])
	}
}

func TestResolveServersOverrideEnvMerge(t *testing.T) {
	reg := testRegistry(t)
	s := &Store{Projects: map[string]model.Project{
		"proj": {
			Name:    "proj",
			Path:    "/proj",
			Clients: []model.ClientType{model.ClientClaudeCode},
			MCPs: []model.MCPAssignment{
				{Name: "qdrant", Overrides: &model.ServerOverride{
					Env: map[string]string{"COLLECTION_NAME": "agent-context"},
				}},
			},
		},
	}}

	resolved, err := s.ResolveServers("proj", reg)
	if err != nil {
		t.Fatalf("ResolveServers: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 server, got %d", len(resolved))
	}

	qdrant := resolved[0]
	// Env should be merged: QDRANT_URL from registry, COLLECTION_NAME overridden.
	if qdrant.Env["QDRANT_URL"] != "${QDRANT_URL}" {
		t.Errorf("expected QDRANT_URL preserved, got %q", qdrant.Env["QDRANT_URL"])
	}
	if qdrant.Env["COLLECTION_NAME"] != "agent-context" {
		t.Errorf("expected COLLECTION_NAME=agent-context, got %q", qdrant.Env["COLLECTION_NAME"])
	}
}

func TestResolveServersOverrideArgsReplace(t *testing.T) {
	reg := testRegistry(t)
	s := &Store{Projects: map[string]model.Project{
		"proj": {
			Name:    "proj",
			Path:    "/proj",
			Clients: []model.ClientType{model.ClientClaudeCode},
			MCPs: []model.MCPAssignment{
				{Name: "github", Overrides: &model.ServerOverride{
					Args: []string{"--custom", "flag"},
				}},
			},
		},
	}}

	resolved, err := s.ResolveServers("proj", reg)
	if err != nil {
		t.Fatalf("ResolveServers: %v", err)
	}

	github := resolved[0]
	if len(github.Args) != 2 || github.Args[0] != "--custom" || github.Args[1] != "flag" {
		t.Errorf("expected args replaced to [--custom flag], got %v", github.Args)
	}
}

func TestResolveServersOverrideCommand(t *testing.T) {
	reg := testRegistry(t)
	cmd := "custom-npx"
	s := &Store{Projects: map[string]model.Project{
		"proj": {
			Name:    "proj",
			Path:    "/proj",
			Clients: []model.ClientType{model.ClientClaudeCode},
			MCPs: []model.MCPAssignment{
				{Name: "github", Overrides: &model.ServerOverride{
					Command: &cmd,
				}},
			},
		},
	}}

	resolved, err := s.ResolveServers("proj", reg)
	if err != nil {
		t.Fatalf("ResolveServers: %v", err)
	}
	if resolved[0].Command != "custom-npx" {
		t.Errorf("expected command=custom-npx, got %q", resolved[0].Command)
	}
}

func TestResolveServersOverrideURL(t *testing.T) {
	reg := testRegistry(t)
	url := "https://new.example.com/mcp"
	s := &Store{Projects: map[string]model.Project{
		"proj": {
			Name:    "proj",
			Path:    "/proj",
			Clients: []model.ClientType{model.ClientClaudeCode},
			MCPs: []model.MCPAssignment{
				{Name: "remote-api", Overrides: &model.ServerOverride{
					URL: &url,
				}},
			},
		},
	}}

	resolved, err := s.ResolveServers("proj", reg)
	if err != nil {
		t.Fatalf("ResolveServers: %v", err)
	}
	if resolved[0].URL != "https://new.example.com/mcp" {
		t.Errorf("expected URL override, got %q", resolved[0].URL)
	}
}

func TestResolveServersOverrideHeaders(t *testing.T) {
	reg := testRegistry(t)
	s := &Store{Projects: map[string]model.Project{
		"proj": {
			Name:    "proj",
			Path:    "/proj",
			Clients: []model.ClientType{model.ClientClaudeCode},
			MCPs: []model.MCPAssignment{
				{Name: "remote-api", Overrides: &model.ServerOverride{
					Headers: map[string]string{"X-Custom": "value"},
				}},
			},
		},
	}}

	resolved, err := s.ResolveServers("proj", reg)
	if err != nil {
		t.Fatalf("ResolveServers: %v", err)
	}

	api := resolved[0]
	// Headers should be merged: Authorization from registry, X-Custom from override.
	if api.Headers["Authorization"] != "Bearer ${API_TOKEN}" {
		t.Errorf("expected Authorization preserved, got %q", api.Headers["Authorization"])
	}
	if api.Headers["X-Custom"] != "value" {
		t.Errorf("expected X-Custom=value, got %q", api.Headers["X-Custom"])
	}
}

func TestResolveServersDanglingReference(t *testing.T) {
	reg := testRegistry(t)
	s := &Store{Projects: map[string]model.Project{
		"proj": {
			Name:    "proj",
			Path:    "/proj",
			Clients: []model.ClientType{model.ClientClaudeCode},
			MCPs: []model.MCPAssignment{
				{Name: "nonexistent-server"},
			},
		},
	}}

	_, err := s.ResolveServers("proj", reg)
	if err == nil {
		t.Fatal("expected error for dangling server reference")
	}
}

func TestResolveServersProjectNotFound(t *testing.T) {
	reg := testRegistry(t)
	s := &Store{Projects: make(map[string]model.Project)}

	_, err := s.ResolveServers("nonexistent", reg)
	if err == nil {
		t.Fatal("expected error for non-existent project")
	}
}

func TestResolveServersDanglingTag(t *testing.T) {
	reg := testRegistry(t)
	s := &Store{Projects: map[string]model.Project{
		"proj": {
			Name:    "proj",
			Path:    "/proj",
			Clients: []model.ClientType{model.ClientClaudeCode},
			Tags:    []string{"nonexistent-tag"},
		},
	}}

	_, err := s.ResolveServers("proj", reg)
	if err == nil {
		t.Fatal("expected error for dangling tag reference")
	}
}

func TestResolveServersServerInBothTagAndMCPsWithOverride(t *testing.T) {
	reg := testRegistry(t)
	// github is in tag "core" AND in mcps with an override.
	s := &Store{Projects: map[string]model.Project{
		"proj": {
			Name:    "proj",
			Path:    "/proj",
			Clients: []model.ClientType{model.ClientClaudeCode},
			Tags:    []string{"core"},
			MCPs: []model.MCPAssignment{
				{Name: "github", Overrides: &model.ServerOverride{
					Env: map[string]string{"GITHUB_TOKEN": "${ORG_TOKEN}"},
				}},
			},
		},
	}}

	resolved, err := s.ResolveServers("proj", reg)
	if err != nil {
		t.Fatalf("ResolveServers: %v", err)
	}

	// Should have 2: github (from tag, deduped with mcps) and filesystem.
	if len(resolved) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(resolved))
	}

	var github *model.ServerDef
	for i := range resolved {
		if resolved[i].Name == "github" {
			github = &resolved[i]
		}
	}
	if github == nil {
		t.Fatal("github not found in resolved servers")
	}
	// Override should be applied even though github came from tag expansion.
	if github.Env["GITHUB_TOKEN"] != "${ORG_TOKEN}" {
		t.Errorf("expected override applied, got %q", github.Env["GITHUB_TOKEN"])
	}
}

func TestListSorted(t *testing.T) {
	s := &Store{Projects: map[string]model.Project{
		"zebra": {Name: "zebra", Path: "/z"},
		"alpha": {Name: "alpha", Path: "/a"},
		"mid":   {Name: "mid", Path: "/m"},
	}}

	list := s.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 projects, got %d", len(list))
	}
	if list[0].Name != "alpha" || list[1].Name != "mid" || list[2].Name != "zebra" {
		t.Errorf("expected sorted order, got %v, %v, %v", list[0].Name, list[1].Name, list[2].Name)
	}
}
