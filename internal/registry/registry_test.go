package registry

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/hystak/hystak/internal/model"
)

func TestNew_IsEmpty(t *testing.T) {
	reg := New()
	if !reg.IsEmpty() {
		t.Error("New() registry should be empty")
	}
}

func TestRegistry_IsEmpty_WithServer(t *testing.T) {
	reg := New()
	if err := reg.Servers.Add(model.ServerDef{Name: "github", Transport: model.TransportStdio, Command: "npx"}); err != nil {
		t.Fatal(err)
	}
	if reg.IsEmpty() {
		t.Error("registry with server should not be empty")
	}
}

func TestRegistry_IsEmpty_WithTag(t *testing.T) {
	reg := New()
	if err := reg.AddTag("core", []string{"github"}); err != nil {
		t.Fatal(err)
	}
	if reg.IsEmpty() {
		t.Error("registry with tag should not be empty")
	}
}

func TestRegistry_AddTag(t *testing.T) {
	reg := New()
	if err := reg.AddTag("core", []string{"github", "postgres"}); err != nil {
		t.Fatal(err)
	}
	if !reg.HasTags() {
		t.Error("HasTags() should be true")
	}
}

func TestRegistry_AddTag_Duplicate(t *testing.T) {
	reg := New()
	if err := reg.AddTag("core", []string{"github"}); err != nil {
		t.Fatal(err)
	}
	err := reg.AddTag("core", []string{"postgres"})
	if err == nil {
		t.Fatal("expected error for duplicate tag")
	}
}

func TestRegistry_AddTag_EmptyName(t *testing.T) {
	reg := New()
	err := reg.AddTag("", []string{"github"})
	if err == nil {
		t.Fatal("expected error for empty tag name")
	}
}

func TestRegistry_GetTag(t *testing.T) {
	reg := New()
	if err := reg.AddTag("core", []string{"github", "postgres"}); err != nil {
		t.Fatal(err)
	}
	members, ok := reg.GetTag("core")
	if !ok {
		t.Fatal("GetTag returned false")
	}
	if !slices.Equal(members, []string{"github", "postgres"}) {
		t.Errorf("members = %v, want [github postgres]", members)
	}
}

func TestRegistry_GetTag_ReturnsCopy(t *testing.T) {
	reg := New()
	if err := reg.AddTag("core", []string{"github"}); err != nil {
		t.Fatal(err)
	}
	members, _ := reg.GetTag("core")
	members[0] = "mutated"

	original, _ := reg.GetTag("core")
	if original[0] != "github" {
		t.Error("GetTag returned live slice, not a copy")
	}
}

func TestRegistry_GetTag_NotFound(t *testing.T) {
	reg := New()
	_, ok := reg.GetTag("nonexistent")
	if ok {
		t.Error("GetTag returned true for nonexistent tag")
	}
}

func TestRegistry_DeleteTag(t *testing.T) {
	reg := New()
	if err := reg.AddTag("core", []string{"github"}); err != nil {
		t.Fatal(err)
	}
	if err := reg.DeleteTag("core"); err != nil {
		t.Fatal(err)
	}
	if reg.HasTags() {
		t.Error("HasTags() should be false after delete")
	}
}

func TestRegistry_DeleteTag_NotFound(t *testing.T) {
	reg := New()
	err := reg.DeleteTag("nonexistent")
	if err == nil {
		t.Fatal("expected error for deleting nonexistent tag")
	}
}

func TestRegistry_ListTags_ReturnsCopy(t *testing.T) {
	reg := New()
	if err := reg.AddTag("core", []string{"github"}); err != nil {
		t.Fatal(err)
	}
	tags := reg.ListTags()
	delete(tags, "core")

	if !reg.HasTags() {
		t.Error("ListTags returned live map, not a copy")
	}
}

func TestRegistry_SaveLoad_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "registry.yaml")

	origGH := model.ServerDef{
		Name:      "github",
		Transport: model.TransportStdio,
		Command:   "npx",
		Args:      []string{"-y", "@anthropic/mcp-github"},
		Env:       map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
	}
	origRemote := model.ServerDef{
		Name:      "remote",
		Transport: model.TransportSSE,
		URL:       "https://mcp.example.com/sse",
		Headers:   map[string]string{"Auth": "Bearer ${TOKEN}"},
	}
	origSkill := model.SkillDef{
		Name:        "code-review",
		Description: "Structured review",
		Source:      "/path/to/SKILL.md",
	}
	origHook := model.HookDef{
		Name:    "lint",
		Event:   model.HookEventPostToolUse,
		Matcher: "Edit",
		Command: "npm run lint",
		Timeout: 30,
	}
	origPerm := model.PermissionRule{
		Name: "allow-bash",
		Rule: "Bash(*)",
		Type: model.PermissionAllow,
	}
	origTemplate := model.TemplateDef{
		Name:   "standard",
		Source: "/path/to/template.md",
	}
	origPrompt := model.PromptDef{
		Name:        "security",
		Description: "Security rules",
		Source:      "/path/to/security.md",
		Category:    "safety",
		Order:       10,
		Tags:        []string{"security"},
	}
	origTagMembers := []string{"github", "remote"}

	original := New()
	if err := original.Servers.Add(origGH); err != nil {
		t.Fatal(err)
	}
	if err := original.Servers.Add(origRemote); err != nil {
		t.Fatal(err)
	}
	if err := original.Skills.Add(origSkill); err != nil {
		t.Fatal(err)
	}
	if err := original.Hooks.Add(origHook); err != nil {
		t.Fatal(err)
	}
	if err := original.Permissions.Add(origPerm); err != nil {
		t.Fatal(err)
	}
	if err := original.Templates.Add(origTemplate); err != nil {
		t.Fatal(err)
	}
	if err := original.Prompts.Add(origPrompt); err != nil {
		t.Fatal(err)
	}
	if err := original.AddTag("core", origTagMembers); err != nil {
		t.Fatal(err)
	}

	if err := original.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("servers", func(t *testing.T) {
		if loaded.Servers.Len() != 2 {
			t.Fatalf("Servers.Len() = %d, want 2", loaded.Servers.Len())
		}
		gh, ok := loaded.Servers.Get("github")
		if !ok {
			t.Fatal("missing server 'github'")
		}
		if !reflect.DeepEqual(gh, origGH) {
			t.Errorf("github mismatch:\n  got:  %+v\n  want: %+v", gh, origGH)
		}
		remote, ok := loaded.Servers.Get("remote")
		if !ok {
			t.Fatal("missing server 'remote'")
		}
		if !reflect.DeepEqual(remote, origRemote) {
			t.Errorf("remote mismatch:\n  got:  %+v\n  want: %+v", remote, origRemote)
		}
	})

	t.Run("skills", func(t *testing.T) {
		skill, ok := loaded.Skills.Get("code-review")
		if !ok {
			t.Fatal("missing skill 'code-review'")
		}
		if !reflect.DeepEqual(skill, origSkill) {
			t.Errorf("mismatch:\n  got:  %+v\n  want: %+v", skill, origSkill)
		}
	})

	t.Run("hooks", func(t *testing.T) {
		hook, ok := loaded.Hooks.Get("lint")
		if !ok {
			t.Fatal("missing hook 'lint'")
		}
		if !reflect.DeepEqual(hook, origHook) {
			t.Errorf("mismatch:\n  got:  %+v\n  want: %+v", hook, origHook)
		}
	})

	t.Run("permissions", func(t *testing.T) {
		perm, ok := loaded.Permissions.Get("allow-bash")
		if !ok {
			t.Fatal("missing permission 'allow-bash'")
		}
		if !reflect.DeepEqual(perm, origPerm) {
			t.Errorf("mismatch:\n  got:  %+v\n  want: %+v", perm, origPerm)
		}
	})

	t.Run("templates", func(t *testing.T) {
		tmpl, ok := loaded.Templates.Get("standard")
		if !ok {
			t.Fatal("missing template 'standard'")
		}
		if !reflect.DeepEqual(tmpl, origTemplate) {
			t.Errorf("mismatch:\n  got:  %+v\n  want: %+v", tmpl, origTemplate)
		}
	})

	t.Run("prompts", func(t *testing.T) {
		prompt, ok := loaded.Prompts.Get("security")
		if !ok {
			t.Fatal("missing prompt 'security'")
		}
		if !reflect.DeepEqual(prompt, origPrompt) {
			t.Errorf("mismatch:\n  got:  %+v\n  want: %+v", prompt, origPrompt)
		}
	})

	t.Run("tags", func(t *testing.T) {
		members, ok := loaded.GetTag("core")
		if !ok {
			t.Fatal("missing tag 'core'")
		}
		if !slices.Equal(members, origTagMembers) {
			t.Errorf("tag members mismatch:\n  got:  %v\n  want: %v", members, origTagMembers)
		}
	})
}

func TestRegistry_Load_NonexistentFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "nonexistent.yaml")

	reg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reg.IsEmpty() {
		t.Error("loading nonexistent file should return empty registry")
	}
}

func TestRegistry_Load_MalformedYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "registry.yaml")

	if err := os.WriteFile(path, []byte("mcps: [invalid yaml"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
}

func TestRegistry_Load_EmptyFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "registry.yaml")

	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	reg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reg.IsEmpty() {
		t.Error("loading empty file should return empty registry")
	}
}

func TestRegistry_Save_CreatesFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "registry.yaml")

	reg := New()
	if err := reg.Save(path); err != nil {
		t.Fatal(err)
	}

	_, err := os.Stat(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
}

func TestRegistry_PromptsSortedByOrder(t *testing.T) {
	reg := New()
	if err := reg.Prompts.Add(model.PromptDef{Name: "style", Order: 20, Source: "/s"}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Prompts.Add(model.PromptDef{Name: "security", Order: 10, Source: "/s"}); err != nil {
		t.Fatal(err)
	}

	list := reg.Prompts.List()
	if len(list) != 2 {
		t.Fatalf("Prompts.List() = %d items, want 2", len(list))
	}
	if list[0].Name != "security" {
		t.Errorf("Prompts.List()[0] = %q, want 'security' (order 10)", list[0].Name)
	}
	if list[1].Name != "style" {
		t.Errorf("Prompts.List()[1] = %q, want 'style' (order 20)", list[1].Name)
	}
}

func TestRegistry_SaveLoad_SetItemsSetsNames(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "registry.yaml")

	reg := New()
	if err := reg.Servers.Add(model.ServerDef{
		Name:      "github",
		Transport: model.TransportStdio,
		Command:   "npx",
	}); err != nil {
		t.Fatal(err)
	}

	if err := reg.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	srv, ok := loaded.Servers.Get("github")
	if !ok {
		t.Fatal("server 'github' not found after load")
	}
	if srv.Name != "github" {
		t.Errorf("Name = %q, want %q (SetItems should set name from key)", srv.Name, "github")
	}
}
