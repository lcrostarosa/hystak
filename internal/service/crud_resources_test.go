package service

import (
	"reflect"
	"testing"

	"github.com/hystak/hystak/internal/model"
)

// --- Skills ---

func TestService_AddSkill(t *testing.T) {
	svc, _ := setupTestService(t)

	skill := model.SkillDef{Name: "review", Description: "Code review", Source: "/skills/review.md"}
	if err := svc.AddSkill(skill); err != nil {
		t.Fatal(err)
	}

	// Verify via Get (Item 4: round-trip content check)
	got, ok := svc.GetSkill("review")
	if !ok {
		t.Fatal("skill not found after Add")
	}
	if !reflect.DeepEqual(got, skill) {
		t.Errorf("GetSkill after Add:\n  got:  %+v\n  want: %+v", got, skill)
	}
}

func TestService_AddSkill_Duplicate(t *testing.T) {
	svc, _ := setupTestService(t)

	skill := model.SkillDef{Name: "review", Source: "/skills/review.md"}
	if err := svc.AddSkill(skill); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddSkill(skill); err == nil {
		t.Error("expected error for duplicate skill")
	}
}

func TestService_UpdateSkill(t *testing.T) {
	svc, _ := setupTestService(t)

	skill := model.SkillDef{Name: "review", Description: "Code review", Source: "/skills/review.md"}
	if err := svc.AddSkill(skill); err != nil {
		t.Fatal(err)
	}

	skill.Description = "Updated review"
	if err := svc.UpdateSkill(skill); err != nil {
		t.Fatal(err)
	}

	// Verify the updated field changed (Item 4)
	got, ok := svc.GetSkill("review")
	if !ok {
		t.Fatal("skill not found after Update")
	}
	if got.Description != "Updated review" {
		t.Errorf("Description = %q, want 'Updated review'", got.Description)
	}
}

func TestService_UpdateSkill_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	skill := model.SkillDef{Name: "nonexistent", Source: "/skills/nope.md"}
	if err := svc.UpdateSkill(skill); err == nil {
		t.Error("expected error for updating nonexistent skill")
	}
}

func TestService_DeleteSkill(t *testing.T) {
	svc, _ := setupTestService(t)

	skill := model.SkillDef{Name: "review", Source: "/skills/review.md"}
	if err := svc.AddSkill(skill); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteSkill("review"); err != nil {
		t.Fatal(err)
	}
	if len(svc.ListSkills()) != 0 {
		t.Error("skill should be deleted")
	}
}

func TestService_DeleteSkill_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	if err := svc.DeleteSkill("nonexistent"); err == nil {
		t.Error("expected error for deleting nonexistent skill")
	}
}

func TestService_DeleteSkill_CascadesToProfile(t *testing.T) {
	svc, _ := setupTestService(t)

	// Add a skill
	skill := model.SkillDef{Name: "review", Source: "/skills/review.md"}
	if err := svc.AddSkill(skill); err != nil {
		t.Fatal(err)
	}

	// Add the skill to a profile
	if _, err := svc.ToggleProfileResource("dev", "skills", "review"); err != nil {
		t.Fatal(err)
	}

	// Verify it's in the profile
	prof, err := svc.LoadProfile("dev")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range prof.Skills {
		if s == "review" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("skill 'review' should be in profile before delete")
	}

	// Delete the skill via service
	if err := svc.DeleteSkill("review"); err != nil {
		t.Fatal(err)
	}

	// Reload the profile and verify the skill reference is gone
	prof, err = svc.LoadProfile("dev")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range prof.Skills {
		if s == "review" {
			t.Error("skill 'review' should have been removed from profile after cascade delete")
		}
	}
}

// --- Hooks ---

func TestService_AddHook(t *testing.T) {
	svc, _ := setupTestService(t)

	hook := model.HookDef{Name: "lint", Event: model.HookEventPostToolUse, Matcher: "Edit", Command: "eslint", Timeout: 30}
	if err := svc.AddHook(hook); err != nil {
		t.Fatal(err)
	}

	got, ok := svc.GetHook("lint")
	if !ok {
		t.Fatal("hook not found after Add")
	}
	if !reflect.DeepEqual(got, hook) {
		t.Errorf("GetHook after Add:\n  got:  %+v\n  want: %+v", got, hook)
	}
}

func TestService_AddHook_Duplicate(t *testing.T) {
	svc, _ := setupTestService(t)

	hook := model.HookDef{Name: "lint", Event: model.HookEventPostToolUse, Command: "eslint"}
	if err := svc.AddHook(hook); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddHook(hook); err == nil {
		t.Error("expected error for duplicate hook")
	}
}

func TestService_UpdateHook(t *testing.T) {
	svc, _ := setupTestService(t)

	hook := model.HookDef{Name: "lint", Event: model.HookEventPostToolUse, Matcher: "Edit", Command: "eslint", Timeout: 30}
	if err := svc.AddHook(hook); err != nil {
		t.Fatal(err)
	}

	hook.Timeout = 60
	if err := svc.UpdateHook(hook); err != nil {
		t.Fatal(err)
	}

	got, ok := svc.GetHook("lint")
	if !ok {
		t.Fatal("hook not found after Update")
	}
	if got.Timeout != 60 {
		t.Errorf("Timeout = %d, want 60", got.Timeout)
	}
}

func TestService_UpdateHook_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	hook := model.HookDef{Name: "nonexistent", Command: "nope"}
	if err := svc.UpdateHook(hook); err == nil {
		t.Error("expected error for updating nonexistent hook")
	}
}

func TestService_DeleteHook(t *testing.T) {
	svc, _ := setupTestService(t)

	hook := model.HookDef{Name: "lint", Event: model.HookEventPostToolUse, Command: "eslint", Timeout: 30}
	if err := svc.AddHook(hook); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteHook("lint"); err != nil {
		t.Fatal(err)
	}
	if len(svc.ListHooks()) != 0 {
		t.Error("hook should be deleted")
	}
}

func TestService_DeleteHook_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	if err := svc.DeleteHook("nonexistent"); err == nil {
		t.Error("expected error for deleting nonexistent hook")
	}
}

// --- Permissions ---

func TestService_AddPermission(t *testing.T) {
	svc, _ := setupTestService(t)

	perm := model.PermissionRule{Name: "allow-bash", Rule: "Bash(*)", Type: model.PermissionAllow}
	if err := svc.AddPermission(perm); err != nil {
		t.Fatal(err)
	}

	got, ok := svc.GetPermission("allow-bash")
	if !ok {
		t.Fatal("permission not found after Add")
	}
	if !reflect.DeepEqual(got, perm) {
		t.Errorf("GetPermission after Add:\n  got:  %+v\n  want: %+v", got, perm)
	}
}

func TestService_AddPermission_Duplicate(t *testing.T) {
	svc, _ := setupTestService(t)

	perm := model.PermissionRule{Name: "allow-bash", Rule: "Bash(*)", Type: model.PermissionAllow}
	if err := svc.AddPermission(perm); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddPermission(perm); err == nil {
		t.Error("expected error for duplicate permission")
	}
}

func TestService_UpdatePermission(t *testing.T) {
	svc, _ := setupTestService(t)

	perm := model.PermissionRule{Name: "allow-bash", Rule: "Bash(*)", Type: model.PermissionAllow}
	if err := svc.AddPermission(perm); err != nil {
		t.Fatal(err)
	}

	perm.Rule = "Bash(ls)"
	if err := svc.UpdatePermission(perm); err != nil {
		t.Fatal(err)
	}

	got, ok := svc.GetPermission("allow-bash")
	if !ok {
		t.Fatal("permission not found after Update")
	}
	if got.Rule != "Bash(ls)" {
		t.Errorf("Rule = %q, want Bash(ls)", got.Rule)
	}
}

func TestService_UpdatePermission_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	perm := model.PermissionRule{Name: "nonexistent", Rule: "Bash(*)", Type: model.PermissionAllow}
	if err := svc.UpdatePermission(perm); err == nil {
		t.Error("expected error for updating nonexistent permission")
	}
}

func TestService_DeletePermission(t *testing.T) {
	svc, _ := setupTestService(t)

	perm := model.PermissionRule{Name: "allow-bash", Rule: "Bash(*)", Type: model.PermissionAllow}
	if err := svc.AddPermission(perm); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeletePermission("allow-bash"); err != nil {
		t.Fatal(err)
	}
	if len(svc.ListPermissions()) != 0 {
		t.Error("permission should be deleted")
	}
}

func TestService_DeletePermission_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	if err := svc.DeletePermission("nonexistent"); err == nil {
		t.Error("expected error for deleting nonexistent permission")
	}
}

// --- Templates ---

func TestService_AddTemplate(t *testing.T) {
	svc, _ := setupTestService(t)

	tmpl := model.TemplateDef{Name: "standard", Source: "/templates/standard.md"}
	if err := svc.AddTemplate(tmpl); err != nil {
		t.Fatal(err)
	}

	got, ok := svc.GetTemplate("standard")
	if !ok {
		t.Fatal("template not found after Add")
	}
	if !reflect.DeepEqual(got, tmpl) {
		t.Errorf("GetTemplate after Add:\n  got:  %+v\n  want: %+v", got, tmpl)
	}
}

func TestService_AddTemplate_Duplicate(t *testing.T) {
	svc, _ := setupTestService(t)

	tmpl := model.TemplateDef{Name: "standard", Source: "/templates/standard.md"}
	if err := svc.AddTemplate(tmpl); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddTemplate(tmpl); err == nil {
		t.Error("expected error for duplicate template")
	}
}

func TestService_UpdateTemplate(t *testing.T) {
	svc, _ := setupTestService(t)

	tmpl := model.TemplateDef{Name: "standard", Source: "/templates/standard.md"}
	if err := svc.AddTemplate(tmpl); err != nil {
		t.Fatal(err)
	}

	tmpl.Source = "/templates/v2.md"
	if err := svc.UpdateTemplate(tmpl); err != nil {
		t.Fatal(err)
	}

	got, ok := svc.GetTemplate("standard")
	if !ok {
		t.Fatal("template not found after Update")
	}
	if got.Source != "/templates/v2.md" {
		t.Errorf("Source = %q, want /templates/v2.md", got.Source)
	}
}

func TestService_UpdateTemplate_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	tmpl := model.TemplateDef{Name: "nonexistent", Source: "/nope"}
	if err := svc.UpdateTemplate(tmpl); err == nil {
		t.Error("expected error for updating nonexistent template")
	}
}

func TestService_DeleteTemplate(t *testing.T) {
	svc, _ := setupTestService(t)

	tmpl := model.TemplateDef{Name: "standard", Source: "/templates/standard.md"}
	if err := svc.AddTemplate(tmpl); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteTemplate("standard"); err != nil {
		t.Fatal(err)
	}
	if len(svc.ListTemplates()) != 0 {
		t.Error("template should be deleted")
	}
}

func TestService_DeleteTemplate_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	if err := svc.DeleteTemplate("nonexistent"); err == nil {
		t.Error("expected error for deleting nonexistent template")
	}
}

// --- Prompts ---

func TestService_AddPrompt(t *testing.T) {
	svc, _ := setupTestService(t)

	prompt := model.PromptDef{Name: "safety", Description: "Safety rules", Source: "/prompts/safety.md", Category: "safety", Order: 10, Tags: []string{"default"}}
	if err := svc.AddPrompt(prompt); err != nil {
		t.Fatal(err)
	}

	got, ok := svc.GetPrompt("safety")
	if !ok {
		t.Fatal("prompt not found after Add")
	}
	if !reflect.DeepEqual(got, prompt) {
		t.Errorf("GetPrompt after Add:\n  got:  %+v\n  want: %+v", got, prompt)
	}
}

func TestService_AddPrompt_Duplicate(t *testing.T) {
	svc, _ := setupTestService(t)

	prompt := model.PromptDef{Name: "safety", Source: "/prompts/safety.md"}
	if err := svc.AddPrompt(prompt); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddPrompt(prompt); err == nil {
		t.Error("expected error for duplicate prompt")
	}
}

func TestService_UpdatePrompt(t *testing.T) {
	svc, _ := setupTestService(t)

	prompt := model.PromptDef{Name: "safety", Description: "Safety rules", Source: "/prompts/safety.md", Category: "safety", Order: 10, Tags: []string{"default"}}
	if err := svc.AddPrompt(prompt); err != nil {
		t.Fatal(err)
	}

	prompt.Order = 5
	if err := svc.UpdatePrompt(prompt); err != nil {
		t.Fatal(err)
	}

	got, ok := svc.GetPrompt("safety")
	if !ok {
		t.Fatal("prompt not found after Update")
	}
	if got.Order != 5 {
		t.Errorf("Order = %d, want 5", got.Order)
	}
}

func TestService_UpdatePrompt_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	prompt := model.PromptDef{Name: "nonexistent", Source: "/nope"}
	if err := svc.UpdatePrompt(prompt); err == nil {
		t.Error("expected error for updating nonexistent prompt")
	}
}

func TestService_DeletePrompt(t *testing.T) {
	svc, _ := setupTestService(t)

	prompt := model.PromptDef{Name: "safety", Description: "Safety rules", Source: "/prompts/safety.md", Category: "safety", Order: 10, Tags: []string{"default"}}
	if err := svc.AddPrompt(prompt); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeletePrompt("safety"); err != nil {
		t.Fatal(err)
	}
	if len(svc.ListPrompts()) != 0 {
		t.Error("prompt should be deleted")
	}
}

func TestService_DeletePrompt_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	if err := svc.DeletePrompt("nonexistent"); err == nil {
		t.Error("expected error for deleting nonexistent prompt")
	}
}
