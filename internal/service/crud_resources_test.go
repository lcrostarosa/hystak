package service

import (
	"testing"

	"github.com/hystak/hystak/internal/model"
)

// --- Skills ---

func TestService_SkillCRUD(t *testing.T) {
	svc, _ := setupTestService(t)

	skill := model.SkillDef{Name: "review", Description: "Code review", Source: "/skills/review.md"}
	if err := svc.AddSkill(skill); err != nil {
		t.Fatal(err)
	}

	got := svc.ListSkills()
	if len(got) != 1 || got[0].Name != "review" {
		t.Fatalf("ListSkills = %v, want [review]", got)
	}

	skill.Description = "Updated"
	if err := svc.UpdateSkill(skill); err != nil {
		t.Fatal(err)
	}

	if err := svc.DeleteSkill("review"); err != nil {
		t.Fatal(err)
	}
	if len(svc.ListSkills()) != 0 {
		t.Error("skill should be deleted")
	}
}

// --- Hooks ---

func TestService_HookCRUD(t *testing.T) {
	svc, _ := setupTestService(t)

	hook := model.HookDef{Name: "lint", Event: model.HookEventPostToolUse, Matcher: "Edit", Command: "eslint", Timeout: 30}
	if err := svc.AddHook(hook); err != nil {
		t.Fatal(err)
	}

	got := svc.ListHooks()
	if len(got) != 1 || got[0].Name != "lint" {
		t.Fatalf("ListHooks = %v, want [lint]", got)
	}

	hook.Timeout = 60
	if err := svc.UpdateHook(hook); err != nil {
		t.Fatal(err)
	}

	if err := svc.DeleteHook("lint"); err != nil {
		t.Fatal(err)
	}
	if len(svc.ListHooks()) != 0 {
		t.Error("hook should be deleted")
	}
}

// --- Permissions ---

func TestService_PermissionCRUD(t *testing.T) {
	svc, _ := setupTestService(t)

	perm := model.PermissionRule{Name: "allow-bash", Rule: "Bash(*)", Type: model.PermissionAllow}
	if err := svc.AddPermission(perm); err != nil {
		t.Fatal(err)
	}

	got := svc.ListPermissions()
	if len(got) != 1 || got[0].Name != "allow-bash" {
		t.Fatalf("ListPermissions = %v, want [allow-bash]", got)
	}

	perm.Rule = "Bash(ls)"
	if err := svc.UpdatePermission(perm); err != nil {
		t.Fatal(err)
	}

	if err := svc.DeletePermission("allow-bash"); err != nil {
		t.Fatal(err)
	}
	if len(svc.ListPermissions()) != 0 {
		t.Error("permission should be deleted")
	}
}

// --- Templates ---

func TestService_TemplateCRUD(t *testing.T) {
	svc, _ := setupTestService(t)

	tmpl := model.TemplateDef{Name: "standard", Source: "/templates/standard.md"}
	if err := svc.AddTemplate(tmpl); err != nil {
		t.Fatal(err)
	}

	got := svc.ListTemplates()
	if len(got) != 1 || got[0].Name != "standard" {
		t.Fatalf("ListTemplates = %v, want [standard]", got)
	}

	tmpl.Source = "/templates/v2.md"
	if err := svc.UpdateTemplate(tmpl); err != nil {
		t.Fatal(err)
	}

	if err := svc.DeleteTemplate("standard"); err != nil {
		t.Fatal(err)
	}
	if len(svc.ListTemplates()) != 0 {
		t.Error("template should be deleted")
	}
}

// --- Prompts ---

func TestService_PromptCRUD(t *testing.T) {
	svc, _ := setupTestService(t)

	prompt := model.PromptDef{Name: "safety", Description: "Safety rules", Source: "/prompts/safety.md", Category: "safety", Order: 10, Tags: []string{"default"}}
	if err := svc.AddPrompt(prompt); err != nil {
		t.Fatal(err)
	}

	got := svc.ListPrompts()
	if len(got) != 1 || got[0].Name != "safety" {
		t.Fatalf("ListPrompts = %v, want [safety]", got)
	}

	prompt.Order = 5
	if err := svc.UpdatePrompt(prompt); err != nil {
		t.Fatal(err)
	}

	if err := svc.DeletePrompt("safety"); err != nil {
		t.Fatal(err)
	}
	if len(svc.ListPrompts()) != 0 {
		t.Error("prompt should be deleted")
	}
}
