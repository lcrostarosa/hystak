package service

import (
	"testing"

	hysterr "github.com/lcrostarosa/hystak/internal/errors"
	"github.com/lcrostarosa/hystak/internal/model"
)

func TestAddServer(t *testing.T) {
	svc, _ := setupService(t)

	srv := model.ServerDef{
		Name:      "new-srv",
		Transport: model.TransportStdio,
		Command:   "new-cmd",
	}
	if err := svc.AddServer(srv); err != nil {
		t.Fatalf("AddServer: %v", err)
	}

	got, ok := svc.GetServer("new-srv")
	if !ok {
		t.Fatal("server not found after add")
	}
	if got.Command != "new-cmd" {
		t.Errorf("Command = %q, want %q", got.Command, "new-cmd")
	}
}

func TestAddServer_Duplicate(t *testing.T) {
	svc, _ := setupService(t)

	err := svc.AddServer(model.ServerDef{Name: "github", Transport: model.TransportStdio})
	if err == nil {
		t.Fatal("expected error for duplicate")
	}
	if !hysterr.IsAlreadyExists(err) {
		t.Errorf("expected AlreadyExistsError, got: %v", err)
	}
}

func TestUpdateServer(t *testing.T) {
	svc, _ := setupService(t)

	err := svc.UpdateServer("github", model.ServerDef{
		Transport: model.TransportStdio,
		Command:   "updated-cmd",
	})
	if err != nil {
		t.Fatalf("UpdateServer: %v", err)
	}

	got, _ := svc.GetServer("github")
	if got.Command != "updated-cmd" {
		t.Errorf("Command = %q, want %q", got.Command, "updated-cmd")
	}
}

func TestUpdateServer_NotFound(t *testing.T) {
	svc, _ := setupService(t)

	err := svc.UpdateServer("nonexistent", model.ServerDef{})
	if !hysterr.IsNotFound(err) {
		t.Errorf("expected NotFoundError, got: %v", err)
	}
}

func TestDeleteServer(t *testing.T) {
	svc, _ := setupService(t)

	if err := svc.DeleteServer("qdrant"); err != nil {
		t.Fatalf("DeleteServer: %v", err)
	}
	if _, ok := svc.GetServer("qdrant"); ok {
		t.Error("server still exists after delete")
	}
}

func TestDeleteServer_NotFound(t *testing.T) {
	svc, _ := setupService(t)

	err := svc.DeleteServer("nonexistent")
	if !hysterr.IsNotFound(err) {
		t.Errorf("expected NotFoundError, got: %v", err)
	}
}

func TestListServers(t *testing.T) {
	svc, _ := setupService(t)

	servers := svc.ListServers()
	if len(servers) != 3 {
		t.Fatalf("expected 3 servers, got %d", len(servers))
	}
	// Should be sorted.
	if servers[0].Name != "filesystem" {
		t.Errorf("first server = %q, want %q", servers[0].Name, "filesystem")
	}
}

func TestAddSkill_CRUD(t *testing.T) {
	svc, _ := setupService(t)

	skill := model.SkillDef{Name: "test-skill", Source: "/tmp/skill.md"}
	if err := svc.AddSkill(skill); err != nil {
		t.Fatalf("AddSkill: %v", err)
	}

	got, ok := svc.GetSkill("test-skill")
	if !ok {
		t.Fatal("skill not found")
	}
	if got.Source != "/tmp/skill.md" {
		t.Errorf("Source = %q, want %q", got.Source, "/tmp/skill.md")
	}

	if err := svc.UpdateSkill("test-skill", model.SkillDef{Source: "/tmp/updated.md"}); err != nil {
		t.Fatalf("UpdateSkill: %v", err)
	}

	got, _ = svc.GetSkill("test-skill")
	if got.Source != "/tmp/updated.md" {
		t.Errorf("Source = %q, want %q", got.Source, "/tmp/updated.md")
	}

	if err := svc.DeleteSkill("test-skill"); err != nil {
		t.Fatalf("DeleteSkill: %v", err)
	}
	if _, ok := svc.GetSkill("test-skill"); ok {
		t.Error("skill still exists after delete")
	}
}

func TestAddHook_CRUD(t *testing.T) {
	svc, _ := setupService(t)

	hook := model.HookDef{Name: "test-hook", Event: "PreToolUse", Command: "echo test"}
	if err := svc.AddHook(hook); err != nil {
		t.Fatalf("AddHook: %v", err)
	}

	got, ok := svc.GetHook("test-hook")
	if !ok {
		t.Fatal("hook not found")
	}
	if got.Command != "echo test" {
		t.Errorf("Command = %q, want %q", got.Command, "echo test")
	}

	if err := svc.DeleteHook("test-hook"); err != nil {
		t.Fatalf("DeleteHook: %v", err)
	}
	if _, ok := svc.GetHook("test-hook"); ok {
		t.Error("hook still exists after delete")
	}
}

func TestAddPermission_CRUD(t *testing.T) {
	svc, _ := setupService(t)

	perm := model.PermissionRule{Name: "test-perm", Rule: "Bash(*)"}
	if err := svc.AddPermission(perm); err != nil {
		t.Fatalf("AddPermission: %v", err)
	}

	got, ok := svc.GetPermission("test-perm")
	if !ok {
		t.Fatal("permission not found")
	}
	if got.Rule != "Bash(*)" {
		t.Errorf("Rule = %q, want %q", got.Rule, "Bash(*)")
	}

	if err := svc.DeletePermission("test-perm"); err != nil {
		t.Fatalf("DeletePermission: %v", err)
	}
	if _, ok := svc.GetPermission("test-perm"); ok {
		t.Error("permission still exists after delete")
	}
}

func TestAddTemplate_CRUD(t *testing.T) {
	svc, _ := setupService(t)

	tmpl := model.TemplateDef{Name: "test-tmpl", Source: "/tmp/tmpl.md"}
	if err := svc.AddTemplate(tmpl); err != nil {
		t.Fatalf("AddTemplate: %v", err)
	}

	got, ok := svc.GetTemplate("test-tmpl")
	if !ok {
		t.Fatal("template not found")
	}
	if got.Source != "/tmp/tmpl.md" {
		t.Errorf("Source = %q, want %q", got.Source, "/tmp/tmpl.md")
	}

	if err := svc.DeleteTemplate("test-tmpl"); err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}
	if _, ok := svc.GetTemplate("test-tmpl"); ok {
		t.Error("template still exists after delete")
	}
}

func TestProjectCRUD(t *testing.T) {
	svc, _ := setupService(t)

	proj := model.Project{Name: "new-proj", Path: "/tmp/new"}
	if err := svc.AddProject(proj); err != nil {
		t.Fatalf("AddProject: %v", err)
	}

	got, ok := svc.GetProject("new-proj")
	if !ok {
		t.Fatal("project not found")
	}
	if got.Path != "/tmp/new" {
		t.Errorf("Path = %q, want %q", got.Path, "/tmp/new")
	}

	projects := svc.ListProjects()
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}

	if err := svc.DeleteProject("new-proj"); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	if _, ok := svc.GetProject("new-proj"); ok {
		t.Error("project still exists after delete")
	}
}

func TestAssignServer(t *testing.T) {
	svc, _ := setupService(t)

	// github is already assigned via tag, not direct MCP.
	// Let's add a new project first.
	svc.AddProject(model.Project{Name: "test-proj", Path: "/tmp/test"})

	if err := svc.AssignServer("test-proj", "github"); err != nil {
		t.Fatalf("AssignServer: %v", err)
	}

	proj, _ := svc.GetProject("test-proj")
	found := false
	for _, mcp := range proj.MCPs {
		if mcp.Name == "github" {
			found = true
		}
	}
	if !found {
		t.Error("github not assigned to test-proj")
	}
}

func TestUnassignServer(t *testing.T) {
	svc, _ := setupService(t)

	// qdrant is directly assigned via MCPs in myproject.
	if err := svc.UnassignServer("myproject", "qdrant"); err != nil {
		t.Fatalf("UnassignServer: %v", err)
	}

	proj, _ := svc.GetProject("myproject")
	for _, mcp := range proj.MCPs {
		if mcp.Name == "qdrant" {
			t.Error("qdrant should have been unassigned")
		}
	}
}

func TestAssignSkill(t *testing.T) {
	svc, _ := setupService(t)

	if err := svc.AssignSkill("myproject", "some-skill"); err != nil {
		t.Fatalf("AssignSkill: %v", err)
	}

	proj, _ := svc.GetProject("myproject")
	found := false
	for _, sk := range proj.Skills {
		if sk == "some-skill" {
			found = true
		}
	}
	if !found {
		t.Error("skill not assigned")
	}
}

func TestCountServerProfileRefs(t *testing.T) {
	svc, _ := setupService(t)

	counts := svc.CountServerProfileRefs()
	// github and filesystem come from core tag, qdrant from MCPs.
	if counts["github"] != 1 {
		t.Errorf("github count = %d, want 1", counts["github"])
	}
	if counts["qdrant"] != 1 {
		t.Errorf("qdrant count = %d, want 1", counts["qdrant"])
	}
}

func TestIsServerAssigned(t *testing.T) {
	svc, _ := setupService(t)

	proj, _ := svc.GetProject("myproject")

	if !svc.IsServerAssigned(proj, "github") {
		t.Error("github should be assigned via tag")
	}
	if !svc.IsServerAssigned(proj, "qdrant") {
		t.Error("qdrant should be assigned via MCPs")
	}
}

func TestIsServerFromTag(t *testing.T) {
	svc, _ := setupService(t)

	proj, _ := svc.GetProject("myproject")

	if !svc.IsServerFromTag(proj, "github") {
		t.Error("github should be from tag")
	}
	if svc.IsServerFromTag(proj, "qdrant") {
		t.Error("qdrant should NOT be from tag")
	}
}

func TestExpandTag(t *testing.T) {
	svc, _ := setupService(t)

	names, err := svc.ExpandTag("core")
	if err != nil {
		t.Fatalf("ExpandTag: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
}

func TestExpandTag_NotFound(t *testing.T) {
	svc, _ := setupService(t)

	_, err := svc.ExpandTag("nonexistent")
	if !hysterr.IsNotFound(err) {
		t.Errorf("expected NotFoundError, got: %v", err)
	}
}
