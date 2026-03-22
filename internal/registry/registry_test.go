package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lcrostarosa/hystak/internal/model"
)

func testServer(name string) model.ServerDef {
	return model.ServerDef{
		Name:        name,
		Description: name + " server",
		Transport:   model.TransportStdio,
		Command:     "npx",
		Args:        []string{"-y", "@mcp/" + name},
		Env:         map[string]string{"TOKEN": "${TOKEN}"},
	}
}

func testHTTPServer(name string) model.ServerDef {
	return model.ServerDef{
		Name:        name,
		Description: name + " HTTP server",
		Transport:   model.TransportHTTP,
		URL:         "https://example.com/" + name,
		Headers:     map[string]string{"Authorization": "Bearer ${API_KEY}"},
	}
}

func TestLoadValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.yaml")

	yaml := `servers:
  github:
    description: "GitHub API"
    transport: stdio
    command: npx
    args: ["-y", "@mcp/github"]
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN}"
  remote:
    description: "Remote API"
    transport: http
    url: "https://example.com/mcp"
    headers:
      Authorization: "Bearer ${TOKEN}"
tags:
  core: [github]
`
	_ = os.WriteFile(path, []byte(yaml), 0o644)

	r, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if r.Servers.Len() != 2 {
		t.Fatalf("expected 2 servers, got %d", r.Servers.Len())
	}

	gh, ok := r.Servers.Get("github")
	if !ok {
		t.Fatal("expected github server")
	}
	if gh.Name != "github" {
		t.Errorf("expected Name=github, got %q", gh.Name)
	}
	if gh.Transport != model.TransportStdio {
		t.Errorf("expected stdio transport, got %q", gh.Transport)
	}
	if gh.Command != "npx" {
		t.Errorf("expected command=npx, got %q", gh.Command)
	}
	if gh.Env["GITHUB_TOKEN"] != "${GITHUB_TOKEN}" {
		t.Errorf("unexpected env: %v", gh.Env)
	}

	remote, ok := r.Servers.Get("remote")
	if !ok {
		t.Fatal("expected remote server")
	}
	if remote.Transport != model.TransportHTTP {
		t.Errorf("expected http transport, got %q", remote.Transport)
	}
	if remote.URL != "https://example.com/mcp" {
		t.Errorf("unexpected URL: %q", remote.URL)
	}

	if len(r.Tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(r.Tags))
	}
	if r.Tags["core"][0] != "github" {
		t.Errorf("expected core tag to contain github")
	}
}

func TestLoadEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.yaml")
	_ = os.WriteFile(path, nil, 0o644)

	r, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if r.Servers.Len() != 0 {
		t.Errorf("expected empty servers, got %d", r.Servers.Len())
	}
	if len(r.Tags) != 0 {
		t.Errorf("expected empty tags, got %d", len(r.Tags))
	}
}

func TestLoadMissingFile(t *testing.T) {
	r, err := Load("/nonexistent/registry.yaml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if r.Servers.Len() != 0 {
		t.Errorf("expected empty servers")
	}
}

func TestSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.yaml")

	r := empty()
	_ = r.Servers.Add(testServer("github"))
	_ = r.Servers.Add(testHTTPServer("remote"))
	r.Tags["core"] = []string{"github"}

	if err := r.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	r2, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if r2.Servers.Len() != 2 {
		t.Fatalf("expected 2 servers after reload, got %d", r2.Servers.Len())
	}

	gh, ok := r2.Servers.Get("github")
	if !ok {
		t.Fatal("github not found after reload")
	}
	if gh.Command != "npx" {
		t.Errorf("expected command=npx, got %q", gh.Command)
	}

	remote, ok := r2.Servers.Get("remote")
	if !ok {
		t.Fatal("remote not found after reload")
	}
	if remote.URL != "https://example.com/remote" {
		t.Errorf("expected URL, got %q", remote.URL)
	}

	if len(r2.Tags["core"]) != 1 || r2.Tags["core"][0] != "github" {
		t.Errorf("tag core not preserved: %v", r2.Tags["core"])
	}
}

func TestAddSuccess(t *testing.T) {
	r := empty()
	srv := testServer("github")

	if err := r.Servers.Add(srv); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, ok := r.Servers.Get("github")
	if !ok {
		t.Fatal("server not found after Add")
	}
	if got.Command != "npx" {
		t.Errorf("expected command=npx, got %q", got.Command)
	}
}

func TestAddDuplicate(t *testing.T) {
	r := empty()
	srv := testServer("github")
	_ = r.Servers.Add(srv)

	err := r.Servers.Add(srv)
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestUpdateSuccess(t *testing.T) {
	r := empty()
	_ = r.Servers.Add(testServer("github"))

	updated := testServer("github")
	updated.Description = "Updated description"
	if err := r.Servers.Update("github", updated); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := r.Servers.Get("github")
	if got.Description != "Updated description" {
		t.Errorf("expected updated description, got %q", got.Description)
	}
}

func TestUpdateNotFound(t *testing.T) {
	r := empty()
	err := r.Servers.Update("nonexistent", testServer("nonexistent"))
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDeleteSuccess(t *testing.T) {
	r := empty()
	_ = r.Servers.Add(testServer("github"))

	if err := r.DeleteServer("github"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, ok := r.Servers.Get("github"); ok {
		t.Error("server still exists after Delete")
	}
}

func TestDeleteNotFound(t *testing.T) {
	r := empty()
	err := r.DeleteServer("nonexistent")
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDeleteReferencedByTag(t *testing.T) {
	r := empty()
	_ = r.Servers.Add(testServer("github"))
	r.Tags["core"] = []string{"github"}

	err := r.DeleteServer("github")
	if err == nil {
		t.Fatal("expected referenced-by-tag error")
	}
}

func TestList(t *testing.T) {
	r := empty()
	_ = r.Servers.Add(testServer("zzz"))
	_ = r.Servers.Add(testServer("aaa"))
	_ = r.Servers.Add(testServer("mmm"))

	list := r.Servers.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 servers, got %d", len(list))
	}
	if list[0].Name != "aaa" || list[1].Name != "mmm" || list[2].Name != "zzz" {
		t.Errorf("expected sorted order, got %v", []string{list[0].Name, list[1].Name, list[2].Name})
	}
}

func TestExpandTagSuccess(t *testing.T) {
	r := empty()
	_ = r.Servers.Add(testServer("github"))
	_ = r.Servers.Add(testServer("filesystem"))
	r.Tags["core"] = []string{"github", "filesystem"}

	names, err := r.ExpandTag("core")
	if err != nil {
		t.Fatalf("ExpandTag: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
}

func TestExpandTagUnknown(t *testing.T) {
	r := empty()
	_, err := r.ExpandTag("nonexistent")
	if err == nil {
		t.Fatal("expected unknown tag error")
	}
}

func TestExpandTagMissingServer(t *testing.T) {
	r := empty()
	r.Tags["broken"] = []string{"nonexistent"}

	_, err := r.ExpandTag("broken")
	if err == nil {
		t.Fatal("expected missing server error")
	}
}

func TestAddTagSuccess(t *testing.T) {
	r := empty()
	if err := r.AddTag("core", []string{"github"}); err != nil {
		t.Fatalf("AddTag: %v", err)
	}
	if len(r.Tags["core"]) != 1 {
		t.Errorf("expected 1 server in tag")
	}
}

func TestAddTagDuplicate(t *testing.T) {
	r := empty()
	_ = r.AddTag("core", []string{"github"})
	err := r.AddTag("core", []string{"github"})
	if err == nil {
		t.Fatal("expected duplicate tag error")
	}
}

func TestRemoveTagSuccess(t *testing.T) {
	r := empty()
	_ = r.AddTag("core", []string{"github"})

	if err := r.RemoveTag("core"); err != nil {
		t.Fatalf("RemoveTag: %v", err)
	}
	if _, ok := r.Tags["core"]; ok {
		t.Error("tag still exists after RemoveTag")
	}
}

func TestRemoveTagNotFound(t *testing.T) {
	r := empty()
	err := r.RemoveTag("nonexistent")
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestUpdateTagSuccess(t *testing.T) {
	r := empty()
	_ = r.AddTag("core", []string{"github"})

	if err := r.UpdateTag("core", []string{"github", "filesystem"}); err != nil {
		t.Fatalf("UpdateTag: %v", err)
	}
	if len(r.Tags["core"]) != 2 {
		t.Errorf("expected 2 servers in updated tag")
	}
}

func TestUpdateTagNotFound(t *testing.T) {
	r := empty()
	err := r.UpdateTag("nonexistent", []string{"github"})
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

// --- Skills CRUD tests ---

func testSkill(name string) model.SkillDef {
	return model.SkillDef{
		Name:        name,
		Description: name + " skill",
		Source:      "/path/to/" + name + ".md",
	}
}

func TestAddSkill(t *testing.T) {
	r := empty()
	skill := testSkill("code-review")

	if err := r.Skills.Add(skill); err != nil {
		t.Fatalf("AddSkill: %v", err)
	}

	got, ok := r.Skills.Get("code-review")
	if !ok {
		t.Fatal("skill not found after AddSkill")
	}
	if got.Name != "code-review" {
		t.Errorf("expected Name=code-review, got %q", got.Name)
	}
	if got.Description != "code-review skill" {
		t.Errorf("expected description, got %q", got.Description)
	}
	if got.Source != "/path/to/code-review.md" {
		t.Errorf("expected source path, got %q", got.Source)
	}
}

func TestAddSkillDuplicate(t *testing.T) {
	r := empty()
	skill := testSkill("code-review")
	_ = r.Skills.Add(skill)

	err := r.Skills.Add(skill)
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestGetSkill(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Registry)
		query     string
		wantFound bool
	}{
		{
			name: "existing skill",
			setup: func(r *Registry) {
				_ = r.Skills.Add(testSkill("code-review"))
			},
			query:     "code-review",
			wantFound: true,
		},
		{
			name:      "non-existent skill",
			setup:     func(r *Registry) {},
			query:     "nonexistent",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := empty()
			tt.setup(r)

			_, ok := r.Skills.Get(tt.query)
			if ok != tt.wantFound {
				t.Errorf("GetSkill(%q) found=%v, want %v", tt.query, ok, tt.wantFound)
			}
		})
	}
}

func TestListSkills(t *testing.T) {
	r := empty()
	_ = r.Skills.Add(testSkill("zzz-skill"))
	_ = r.Skills.Add(testSkill("aaa-skill"))
	_ = r.Skills.Add(testSkill("mmm-skill"))

	list := r.Skills.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(list))
	}
	if list[0].Name != "aaa-skill" || list[1].Name != "mmm-skill" || list[2].Name != "zzz-skill" {
		t.Errorf("expected sorted order, got %v", []string{list[0].Name, list[1].Name, list[2].Name})
	}
}

func TestUpdateSkill(t *testing.T) {
	r := empty()
	_ = r.Skills.Add(testSkill("code-review"))

	updated := model.SkillDef{
		Description: "Updated description",
		Source:      "/new/path.md",
	}
	if err := r.Skills.Update("code-review", updated); err != nil {
		t.Fatalf("UpdateSkill: %v", err)
	}

	got, _ := r.Skills.Get("code-review")
	if got.Description != "Updated description" {
		t.Errorf("expected updated description, got %q", got.Description)
	}
	if got.Source != "/new/path.md" {
		t.Errorf("expected updated source, got %q", got.Source)
	}
	if got.Name != "code-review" {
		t.Errorf("expected name preserved, got %q", got.Name)
	}
}

func TestUpdateSkillNotFound(t *testing.T) {
	r := empty()
	err := r.Skills.Update("nonexistent", model.SkillDef{})
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDeleteSkill(t *testing.T) {
	r := empty()
	_ = r.Skills.Add(testSkill("code-review"))

	if err := r.Skills.Delete("code-review"); err != nil {
		t.Fatalf("DeleteSkill: %v", err)
	}

	if _, ok := r.Skills.Get("code-review"); ok {
		t.Error("skill still exists after DeleteSkill")
	}
}

func TestDeleteSkillNotFound(t *testing.T) {
	r := empty()
	err := r.Skills.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

// --- Hooks CRUD tests ---

func testHook(name string) model.HookDef {
	return model.HookDef{
		Name:    name,
		Event:   "PreToolUse",
		Matcher: "Bash",
		Command: "/usr/bin/" + name,
		Timeout: 5000,
	}
}

func TestAddHook(t *testing.T) {
	r := empty()
	hook := testHook("lint-check")

	if err := r.Hooks.Add(hook); err != nil {
		t.Fatalf("AddHook: %v", err)
	}

	got, ok := r.Hooks.Get("lint-check")
	if !ok {
		t.Fatal("hook not found after AddHook")
	}
	if got.Name != "lint-check" {
		t.Errorf("expected Name=lint-check, got %q", got.Name)
	}
	if got.Event != "PreToolUse" {
		t.Errorf("expected Event=PreToolUse, got %q", got.Event)
	}
	if got.Command != "/usr/bin/lint-check" {
		t.Errorf("expected command, got %q", got.Command)
	}
	if got.Timeout != 5000 {
		t.Errorf("expected timeout=5000, got %d", got.Timeout)
	}
}

func TestAddHookDuplicate(t *testing.T) {
	r := empty()
	hook := testHook("lint-check")
	_ = r.Hooks.Add(hook)

	err := r.Hooks.Add(hook)
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestGetHook(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Registry)
		query     string
		wantFound bool
	}{
		{
			name: "existing hook",
			setup: func(r *Registry) {
				_ = r.Hooks.Add(testHook("lint-check"))
			},
			query:     "lint-check",
			wantFound: true,
		},
		{
			name:      "non-existent hook",
			setup:     func(r *Registry) {},
			query:     "nonexistent",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := empty()
			tt.setup(r)

			_, ok := r.Hooks.Get(tt.query)
			if ok != tt.wantFound {
				t.Errorf("GetHook(%q) found=%v, want %v", tt.query, ok, tt.wantFound)
			}
		})
	}
}

func TestListHooks(t *testing.T) {
	r := empty()
	_ = r.Hooks.Add(testHook("zzz-hook"))
	_ = r.Hooks.Add(testHook("aaa-hook"))
	_ = r.Hooks.Add(testHook("mmm-hook"))

	list := r.Hooks.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 hooks, got %d", len(list))
	}
	if list[0].Name != "aaa-hook" || list[1].Name != "mmm-hook" || list[2].Name != "zzz-hook" {
		t.Errorf("expected sorted order, got %v", []string{list[0].Name, list[1].Name, list[2].Name})
	}
}

func TestUpdateHook(t *testing.T) {
	r := empty()
	_ = r.Hooks.Add(testHook("lint-check"))

	updated := model.HookDef{
		Event:   "PostToolUse",
		Matcher: "WebFetch",
		Command: "/usr/bin/updated-hook",
		Timeout: 10000,
	}
	if err := r.Hooks.Update("lint-check", updated); err != nil {
		t.Fatalf("UpdateHook: %v", err)
	}

	got, _ := r.Hooks.Get("lint-check")
	if got.Event != "PostToolUse" {
		t.Errorf("expected updated event, got %q", got.Event)
	}
	if got.Matcher != "WebFetch" {
		t.Errorf("expected updated matcher, got %q", got.Matcher)
	}
	if got.Command != "/usr/bin/updated-hook" {
		t.Errorf("expected updated command, got %q", got.Command)
	}
	if got.Timeout != 10000 {
		t.Errorf("expected updated timeout, got %d", got.Timeout)
	}
	if got.Name != "lint-check" {
		t.Errorf("expected name preserved, got %q", got.Name)
	}
}

func TestUpdateHookNotFound(t *testing.T) {
	r := empty()
	err := r.Hooks.Update("nonexistent", model.HookDef{})
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDeleteHook(t *testing.T) {
	r := empty()
	_ = r.Hooks.Add(testHook("lint-check"))

	if err := r.Hooks.Delete("lint-check"); err != nil {
		t.Fatalf("DeleteHook: %v", err)
	}

	if _, ok := r.Hooks.Get("lint-check"); ok {
		t.Error("hook still exists after DeleteHook")
	}
}

func TestDeleteHookNotFound(t *testing.T) {
	r := empty()
	err := r.Hooks.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

// --- Permissions CRUD tests ---

func testPermission(name string) model.PermissionRule {
	return model.PermissionRule{
		Name: name,
		Rule: "Bash(" + name + ")",
		Type: "allow",
	}
}

func TestAddPermission(t *testing.T) {
	r := empty()
	perm := testPermission("bash-all")

	if err := r.Permissions.Add(perm); err != nil {
		t.Fatalf("AddPermission: %v", err)
	}

	got, ok := r.Permissions.Get("bash-all")
	if !ok {
		t.Fatal("permission not found after AddPermission")
	}
	if got.Name != "bash-all" {
		t.Errorf("expected Name=bash-all, got %q", got.Name)
	}
	if got.Rule != "Bash(bash-all)" {
		t.Errorf("expected rule, got %q", got.Rule)
	}
	if got.Type != "allow" {
		t.Errorf("expected type=allow, got %q", got.Type)
	}
}

func TestAddPermissionDuplicate(t *testing.T) {
	r := empty()
	perm := testPermission("bash-all")
	_ = r.Permissions.Add(perm)

	err := r.Permissions.Add(perm)
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestGetPermission(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Registry)
		query     string
		wantFound bool
	}{
		{
			name: "existing permission",
			setup: func(r *Registry) {
				_ = r.Permissions.Add(testPermission("bash-all"))
			},
			query:     "bash-all",
			wantFound: true,
		},
		{
			name:      "non-existent permission",
			setup:     func(r *Registry) {},
			query:     "nonexistent",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := empty()
			tt.setup(r)

			_, ok := r.Permissions.Get(tt.query)
			if ok != tt.wantFound {
				t.Errorf("GetPermission(%q) found=%v, want %v", tt.query, ok, tt.wantFound)
			}
		})
	}
}

func TestListPermissions(t *testing.T) {
	r := empty()
	_ = r.Permissions.Add(testPermission("zzz-perm"))
	_ = r.Permissions.Add(testPermission("aaa-perm"))
	_ = r.Permissions.Add(testPermission("mmm-perm"))

	list := r.Permissions.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 permissions, got %d", len(list))
	}
	if list[0].Name != "aaa-perm" || list[1].Name != "mmm-perm" || list[2].Name != "zzz-perm" {
		t.Errorf("expected sorted order, got %v", []string{list[0].Name, list[1].Name, list[2].Name})
	}
}

func TestUpdatePermission(t *testing.T) {
	r := empty()
	_ = r.Permissions.Add(testPermission("bash-all"))

	updated := model.PermissionRule{
		Rule: "WebFetch(domain:example.com)",
		Type: "deny",
	}
	if err := r.Permissions.Update("bash-all", updated); err != nil {
		t.Fatalf("UpdatePermission: %v", err)
	}

	got, _ := r.Permissions.Get("bash-all")
	if got.Rule != "WebFetch(domain:example.com)" {
		t.Errorf("expected updated rule, got %q", got.Rule)
	}
	if got.Type != "deny" {
		t.Errorf("expected updated type, got %q", got.Type)
	}
	if got.Name != "bash-all" {
		t.Errorf("expected name preserved, got %q", got.Name)
	}
}

func TestUpdatePermissionNotFound(t *testing.T) {
	r := empty()
	err := r.Permissions.Update("nonexistent", model.PermissionRule{})
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDeletePermission(t *testing.T) {
	r := empty()
	_ = r.Permissions.Add(testPermission("bash-all"))

	if err := r.Permissions.Delete("bash-all"); err != nil {
		t.Fatalf("DeletePermission: %v", err)
	}

	if _, ok := r.Permissions.Get("bash-all"); ok {
		t.Error("permission still exists after DeletePermission")
	}
}

func TestDeletePermissionNotFound(t *testing.T) {
	r := empty()
	err := r.Permissions.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

// --- Templates CRUD tests ---

func testTemplate(name string) model.TemplateDef {
	return model.TemplateDef{
		Name:   name,
		Source: "/templates/" + name + ".md",
	}
}

func TestAddTemplate(t *testing.T) {
	r := empty()
	tmpl := testTemplate("golang-project")

	if err := r.Templates.Add(tmpl); err != nil {
		t.Fatalf("AddTemplate: %v", err)
	}

	got, ok := r.Templates.Get("golang-project")
	if !ok {
		t.Fatal("template not found after AddTemplate")
	}
	if got.Name != "golang-project" {
		t.Errorf("expected Name=golang-project, got %q", got.Name)
	}
	if got.Source != "/templates/golang-project.md" {
		t.Errorf("expected source path, got %q", got.Source)
	}
}

func TestAddTemplateDuplicate(t *testing.T) {
	r := empty()
	tmpl := testTemplate("golang-project")
	_ = r.Templates.Add(tmpl)

	err := r.Templates.Add(tmpl)
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestGetTemplate(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Registry)
		query     string
		wantFound bool
	}{
		{
			name: "existing template",
			setup: func(r *Registry) {
				_ = r.Templates.Add(testTemplate("golang-project"))
			},
			query:     "golang-project",
			wantFound: true,
		},
		{
			name:      "non-existent template",
			setup:     func(r *Registry) {},
			query:     "nonexistent",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := empty()
			tt.setup(r)

			_, ok := r.Templates.Get(tt.query)
			if ok != tt.wantFound {
				t.Errorf("GetTemplate(%q) found=%v, want %v", tt.query, ok, tt.wantFound)
			}
		})
	}
}

func TestListTemplates(t *testing.T) {
	r := empty()
	_ = r.Templates.Add(testTemplate("zzz-template"))
	_ = r.Templates.Add(testTemplate("aaa-template"))
	_ = r.Templates.Add(testTemplate("mmm-template"))

	list := r.Templates.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 templates, got %d", len(list))
	}
	if list[0].Name != "aaa-template" || list[1].Name != "mmm-template" || list[2].Name != "zzz-template" {
		t.Errorf("expected sorted order, got %v", []string{list[0].Name, list[1].Name, list[2].Name})
	}
}

func TestUpdateTemplate(t *testing.T) {
	r := empty()
	_ = r.Templates.Add(testTemplate("golang-project"))

	updated := model.TemplateDef{
		Source: "/new/templates/updated.md",
	}
	if err := r.Templates.Update("golang-project", updated); err != nil {
		t.Fatalf("UpdateTemplate: %v", err)
	}

	got, _ := r.Templates.Get("golang-project")
	if got.Source != "/new/templates/updated.md" {
		t.Errorf("expected updated source, got %q", got.Source)
	}
	if got.Name != "golang-project" {
		t.Errorf("expected name preserved, got %q", got.Name)
	}
}

func TestUpdateTemplateNotFound(t *testing.T) {
	r := empty()
	err := r.Templates.Update("nonexistent", model.TemplateDef{})
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDeleteTemplate(t *testing.T) {
	r := empty()
	_ = r.Templates.Add(testTemplate("golang-project"))

	if err := r.Templates.Delete("golang-project"); err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}

	if _, ok := r.Templates.Get("golang-project"); ok {
		t.Error("template still exists after DeleteTemplate")
	}
}

func TestDeleteTemplateNotFound(t *testing.T) {
	r := empty()
	err := r.Templates.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

// --- Load/Save round-trip test for all entities ---

func TestLoadSaveRoundTripAllEntities(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.yaml")

	// Build a registry with all entity types.
	r := empty()

	// Servers
	_ = r.Servers.Add(testServer("github"))
	_ = r.Servers.Add(testHTTPServer("remote-api"))

	// Skills
	_ = r.Skills.Add(model.SkillDef{
		Name:        "code-review",
		Description: "Reviews code changes",
		Source:      "/skills/code-review.md",
	})
	_ = r.Skills.Add(model.SkillDef{
		Name:        "refactor",
		Description: "Refactors code",
		Source:      "/skills/refactor.md",
	})

	// Hooks
	_ = r.Hooks.Add(model.HookDef{
		Name:    "pre-lint",
		Event:   "PreToolUse",
		Matcher: "Bash",
		Command: "/usr/bin/lint",
		Timeout: 3000,
	})
	_ = r.Hooks.Add(model.HookDef{
		Name:    "post-test",
		Event:   "PostToolUse",
		Command: "/usr/bin/report",
	})

	// Permissions
	_ = r.Permissions.Add(model.PermissionRule{
		Name: "allow-bash",
		Rule: "Bash(*)",
		Type: "allow",
	})
	_ = r.Permissions.Add(model.PermissionRule{
		Name: "deny-web",
		Rule: "WebFetch(domain:evil.com)",
		Type: "deny",
	})

	// Templates
	_ = r.Templates.Add(model.TemplateDef{
		Name:   "go-project",
		Source: "/templates/go-project.md",
	})

	// Prompts
	_ = r.Prompts.Add(model.PromptDef{
		Name:        "defensive-security",
		Description: "Defensive-only security guardrails",
		Source:      "prompts/defensive-security.md",
		Tags:        []string{"security", "guardrail"},
		Category:    "safety",
		Order:       10,
	})

	// Tags
	_ = r.AddTag("core", []string{"github"})
	_ = r.AddTag("all", []string{"github", "remote-api"})

	// Save
	if err := r.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load
	r2, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Verify servers
	if r2.Servers.Len() != 2 {
		t.Fatalf("expected 2 servers, got %d", r2.Servers.Len())
	}
	gh, ok := r2.Servers.Get("github")
	if !ok {
		t.Fatal("github server not found after round-trip")
	}
	if gh.Command != "npx" {
		t.Errorf("expected command=npx, got %q", gh.Command)
	}
	if gh.Transport != model.TransportStdio {
		t.Errorf("expected stdio transport, got %q", gh.Transport)
	}

	remote, ok := r2.Servers.Get("remote-api")
	if !ok {
		t.Fatal("remote-api server not found after round-trip")
	}
	if remote.Transport != model.TransportHTTP {
		t.Errorf("expected http transport, got %q", remote.Transport)
	}
	if remote.URL != "https://example.com/remote-api" {
		t.Errorf("expected URL, got %q", remote.URL)
	}

	// Verify skills
	if r2.Skills.Len() != 2 {
		t.Fatalf("expected 2 skills, got %d", r2.Skills.Len())
	}
	cr, ok := r2.Skills.Get("code-review")
	if !ok {
		t.Fatal("code-review skill not found after round-trip")
	}
	if cr.Name != "code-review" {
		t.Errorf("expected skill Name=code-review, got %q", cr.Name)
	}
	if cr.Description != "Reviews code changes" {
		t.Errorf("expected skill description, got %q", cr.Description)
	}
	if cr.Source != "/skills/code-review.md" {
		t.Errorf("expected skill source, got %q", cr.Source)
	}

	rf, ok := r2.Skills.Get("refactor")
	if !ok {
		t.Fatal("refactor skill not found after round-trip")
	}
	if rf.Source != "/skills/refactor.md" {
		t.Errorf("expected refactor source, got %q", rf.Source)
	}

	// Verify hooks
	if r2.Hooks.Len() != 2 {
		t.Fatalf("expected 2 hooks, got %d", r2.Hooks.Len())
	}
	pl, ok := r2.Hooks.Get("pre-lint")
	if !ok {
		t.Fatal("pre-lint hook not found after round-trip")
	}
	if pl.Name != "pre-lint" {
		t.Errorf("expected hook Name=pre-lint, got %q", pl.Name)
	}
	if pl.Event != "PreToolUse" {
		t.Errorf("expected hook event, got %q", pl.Event)
	}
	if pl.Matcher != "Bash" {
		t.Errorf("expected hook matcher, got %q", pl.Matcher)
	}
	if pl.Command != "/usr/bin/lint" {
		t.Errorf("expected hook command, got %q", pl.Command)
	}
	if pl.Timeout != 3000 {
		t.Errorf("expected hook timeout=3000, got %d", pl.Timeout)
	}

	pt, ok := r2.Hooks.Get("post-test")
	if !ok {
		t.Fatal("post-test hook not found after round-trip")
	}
	if pt.Event != "PostToolUse" {
		t.Errorf("expected post-test event, got %q", pt.Event)
	}

	// Verify permissions
	if r2.Permissions.Len() != 2 {
		t.Fatalf("expected 2 permissions, got %d", r2.Permissions.Len())
	}
	ab, ok := r2.Permissions.Get("allow-bash")
	if !ok {
		t.Fatal("allow-bash permission not found after round-trip")
	}
	if ab.Name != "allow-bash" {
		t.Errorf("expected permission Name=allow-bash, got %q", ab.Name)
	}
	if ab.Rule != "Bash(*)" {
		t.Errorf("expected permission rule, got %q", ab.Rule)
	}
	if ab.Type != "allow" {
		t.Errorf("expected permission type=allow, got %q", ab.Type)
	}

	dw, ok := r2.Permissions.Get("deny-web")
	if !ok {
		t.Fatal("deny-web permission not found after round-trip")
	}
	if dw.Type != "deny" {
		t.Errorf("expected deny type, got %q", dw.Type)
	}

	// Verify templates
	if r2.Templates.Len() != 1 {
		t.Fatalf("expected 1 template, got %d", r2.Templates.Len())
	}
	gp, ok := r2.Templates.Get("go-project")
	if !ok {
		t.Fatal("go-project template not found after round-trip")
	}
	if gp.Name != "go-project" {
		t.Errorf("expected template Name=go-project, got %q", gp.Name)
	}
	if gp.Source != "/templates/go-project.md" {
		t.Errorf("expected template source, got %q", gp.Source)
	}

	// Verify prompts
	if r2.Prompts.Len() != 1 {
		t.Fatalf("expected 1 prompt, got %d", r2.Prompts.Len())
	}
	dp, ok := r2.Prompts.Get("defensive-security")
	if !ok {
		t.Fatal("defensive-security prompt not found after round-trip")
	}
	if dp.Name != "defensive-security" {
		t.Errorf("expected prompt Name=defensive-security, got %q", dp.Name)
	}
	if dp.Description != "Defensive-only security guardrails" {
		t.Errorf("expected prompt description, got %q", dp.Description)
	}
	if dp.Category != "safety" {
		t.Errorf("expected prompt category=safety, got %q", dp.Category)
	}
	if dp.Order != 10 {
		t.Errorf("expected prompt order=10, got %d", dp.Order)
	}
	if len(dp.Tags) != 2 || dp.Tags[0] != "security" {
		t.Errorf("expected prompt tags, got %v", dp.Tags)
	}

	// Verify tags
	if len(r2.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(r2.Tags))
	}
	if len(r2.Tags["core"]) != 1 || r2.Tags["core"][0] != "github" {
		t.Errorf("tag core not preserved: %v", r2.Tags["core"])
	}
	if len(r2.Tags["all"]) != 2 {
		t.Errorf("tag all not preserved: %v", r2.Tags["all"])
	}
}

// --- Prompt CRUD tests ---

func testPrompt(name string, order int) model.PromptDef {
	return model.PromptDef{
		Name:        name,
		Description: name + " prompt",
		Source:      "prompts/" + name + ".md",
		Category:    "test",
		Order:       order,
	}
}

func TestAddPrompt(t *testing.T) {
	r := empty()
	p := testPrompt("defensive-security", 10)

	if err := r.Prompts.Add(p); err != nil {
		t.Fatalf("AddPrompt: %v", err)
	}

	got, ok := r.Prompts.Get("defensive-security")
	if !ok {
		t.Fatal("prompt not found after AddPrompt")
	}
	if got.Name != "defensive-security" {
		t.Errorf("expected Name=defensive-security, got %q", got.Name)
	}
	if got.Order != 10 {
		t.Errorf("expected Order=10, got %d", got.Order)
	}
}

func TestAddPromptDuplicate(t *testing.T) {
	r := empty()
	_ = r.Prompts.Add(testPrompt("my-prompt", 0))

	err := r.Prompts.Add(testPrompt("my-prompt", 0))
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestListPrompts_SortedByOrderThenName(t *testing.T) {
	r := empty()
	_ = r.Prompts.Add(testPrompt("zzz-prompt", 10))
	_ = r.Prompts.Add(testPrompt("aaa-prompt", 20))
	_ = r.Prompts.Add(testPrompt("bbb-prompt", 10))

	list := r.Prompts.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 prompts, got %d", len(list))
	}

	// Order 10: bbb, zzz (alphabetical within same order)
	// Order 20: aaa
	expected := []string{"bbb-prompt", "zzz-prompt", "aaa-prompt"}
	for i, name := range expected {
		if list[i].Name != name {
			t.Errorf("list[%d].Name = %q, want %q", i, list[i].Name, name)
		}
	}
}

func TestUpdatePrompt(t *testing.T) {
	r := empty()
	_ = r.Prompts.Add(testPrompt("my-prompt", 10))

	updated := model.PromptDef{
		Description: "Updated description",
		Source:      "prompts/updated.md",
		Order:       20,
	}
	if err := r.Prompts.Update("my-prompt", updated); err != nil {
		t.Fatalf("UpdatePrompt: %v", err)
	}

	got, _ := r.Prompts.Get("my-prompt")
	if got.Description != "Updated description" {
		t.Errorf("expected updated description, got %q", got.Description)
	}
	if got.Name != "my-prompt" {
		t.Errorf("expected name preserved, got %q", got.Name)
	}
	if got.Order != 20 {
		t.Errorf("expected updated order=20, got %d", got.Order)
	}
}

func TestUpdatePromptNotFound(t *testing.T) {
	r := empty()
	err := r.Prompts.Update("nonexistent", model.PromptDef{})
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDeletePrompt(t *testing.T) {
	r := empty()
	_ = r.Prompts.Add(testPrompt("my-prompt", 0))

	if err := r.Prompts.Delete("my-prompt"); err != nil {
		t.Fatalf("DeletePrompt: %v", err)
	}

	if _, ok := r.Prompts.Get("my-prompt"); ok {
		t.Error("prompt still exists after DeletePrompt")
	}
}

func TestDeletePromptNotFound(t *testing.T) {
	r := empty()
	err := r.Prompts.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected not-found error")
	}
}
