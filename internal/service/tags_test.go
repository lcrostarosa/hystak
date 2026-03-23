package service

import (
	"testing"

	"github.com/hystak/hystak/internal/model"
)

func TestService_AddTag(t *testing.T) {
	svc, _ := setupTestService(t)

	if err := svc.AddTag("core", []string{"github", "postgres"}); err != nil {
		t.Fatal(err)
	}

	members, ok := svc.GetTag("core")
	if !ok {
		t.Fatal("tag 'core' not found")
	}
	if len(members) != 2 {
		t.Errorf("members = %d, want 2", len(members))
	}
}

func TestService_AddTag_DanglingReference(t *testing.T) {
	svc, _ := setupTestService(t)

	err := svc.AddTag("bad", []string{"github", "nonexistent"})
	if err == nil {
		t.Fatal("expected error for dangling reference")
	}
}

func TestService_UpdateTag(t *testing.T) {
	svc, _ := setupTestService(t)

	if err := svc.AddTag("core", []string{"github"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.UpdateTag("core", []string{"github", "postgres"}); err != nil {
		t.Fatal(err)
	}

	members, _ := svc.GetTag("core")
	if len(members) != 2 {
		t.Errorf("members after update = %d, want 2", len(members))
	}
}

func TestService_DeleteTag(t *testing.T) {
	svc, _ := setupTestService(t)

	if err := svc.AddTag("core", []string{"github"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteTag("core"); err != nil {
		t.Fatal(err)
	}

	if _, ok := svc.GetTag("core"); ok {
		t.Error("tag should be deleted")
	}
}

func TestService_ExpandTags(t *testing.T) {
	svc, _ := setupTestService(t)

	if err := svc.AddTag("core", []string{"github", "postgres"}); err != nil {
		t.Fatal(err)
	}

	// Direct: github. Tag expands: github (deduplicated), postgres
	expanded, err := svc.ExpandTags([]string{"github"}, []string{"core"})
	if err != nil {
		t.Fatal(err)
	}
	if len(expanded) != 2 {
		t.Fatalf("expanded = %d, want 2 (deduplicated)", len(expanded))
	}
	// Should be sorted
	if expanded[0] != "github" || expanded[1] != "postgres" {
		t.Errorf("expanded = %v, want [github postgres]", expanded)
	}
}

func TestService_ExpandTags_DanglingTag(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.ExpandTags(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent tag")
	}
}

func TestService_ResolveServers_WithTags(t *testing.T) {
	svc, _ := setupTestService(t)

	if err := svc.AddTag("core", []string{"postgres"}); err != nil {
		t.Fatal(err)
	}

	prof := model.ProjectProfile{
		MCPs: []model.MCPAssignment{{Name: "github"}},
		Tags: []string{"core"},
	}

	resolved, err := svc.resolveServers(prof)
	if err != nil {
		t.Fatal(err)
	}

	if len(resolved) != 2 {
		t.Fatalf("resolved = %d, want 2 (github direct + postgres from tag)", len(resolved))
	}
	if _, ok := resolved["github"]; !ok {
		t.Error("github should be in resolved (direct)")
	}
	if _, ok := resolved["postgres"]; !ok {
		t.Error("postgres should be in resolved (from tag)")
	}
}

func TestService_ResolveServers_TagDanglingServer(t *testing.T) {
	svc, _ := setupTestService(t)

	// Add tag with a valid server, then delete the server
	if err := svc.AddTag("core", []string{"github"}); err != nil {
		t.Fatal(err)
	}
	// Directly remove from registry (bypass validation for test)
	if err := svc.registry.Servers.Delete("github"); err != nil {
		t.Fatal(err)
	}

	prof := model.ProjectProfile{
		Tags: []string{"core"},
	}

	_, err := svc.resolveServers(prof)
	if err == nil {
		t.Fatal("expected error for dangling server reference in tag")
	}
}
