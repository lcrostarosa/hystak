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
	os.WriteFile(path, []byte(yaml), 0o644)

	r, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(r.Servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(r.Servers))
	}

	gh, ok := r.Get("github")
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

	remote, ok := r.Get("remote")
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
	os.WriteFile(path, nil, 0o644)

	r, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(r.Servers) != 0 {
		t.Errorf("expected empty servers, got %d", len(r.Servers))
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
	if len(r.Servers) != 0 {
		t.Errorf("expected empty servers")
	}
}

func TestSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.yaml")

	r := empty()
	r.Servers["github"] = testServer("github")
	r.Servers["remote"] = testHTTPServer("remote")
	r.Tags["core"] = []string{"github"}

	if err := r.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	r2, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(r2.Servers) != 2 {
		t.Fatalf("expected 2 servers after reload, got %d", len(r2.Servers))
	}

	gh, ok := r2.Get("github")
	if !ok {
		t.Fatal("github not found after reload")
	}
	if gh.Command != "npx" {
		t.Errorf("expected command=npx, got %q", gh.Command)
	}

	remote, ok := r2.Get("remote")
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

	if err := r.Add(srv); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, ok := r.Get("github")
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
	r.Add(srv)

	err := r.Add(srv)
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestUpdateSuccess(t *testing.T) {
	r := empty()
	r.Add(testServer("github"))

	updated := testServer("github")
	updated.Description = "Updated description"
	if err := r.Update("github", updated); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := r.Get("github")
	if got.Description != "Updated description" {
		t.Errorf("expected updated description, got %q", got.Description)
	}
}

func TestUpdateNotFound(t *testing.T) {
	r := empty()
	err := r.Update("nonexistent", testServer("nonexistent"))
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDeleteSuccess(t *testing.T) {
	r := empty()
	r.Add(testServer("github"))

	if err := r.Delete("github"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, ok := r.Get("github"); ok {
		t.Error("server still exists after Delete")
	}
}

func TestDeleteNotFound(t *testing.T) {
	r := empty()
	err := r.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestDeleteReferencedByTag(t *testing.T) {
	r := empty()
	r.Add(testServer("github"))
	r.Tags["core"] = []string{"github"}

	err := r.Delete("github")
	if err == nil {
		t.Fatal("expected referenced-by-tag error")
	}
}

func TestList(t *testing.T) {
	r := empty()
	r.Add(testServer("zzz"))
	r.Add(testServer("aaa"))
	r.Add(testServer("mmm"))

	list := r.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 servers, got %d", len(list))
	}
	if list[0].Name != "aaa" || list[1].Name != "mmm" || list[2].Name != "zzz" {
		t.Errorf("expected sorted order, got %v", []string{list[0].Name, list[1].Name, list[2].Name})
	}
}

func TestExpandTagSuccess(t *testing.T) {
	r := empty()
	r.Add(testServer("github"))
	r.Add(testServer("filesystem"))
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
	r.AddTag("core", []string{"github"})
	err := r.AddTag("core", []string{"github"})
	if err == nil {
		t.Fatal("expected duplicate tag error")
	}
}

func TestRemoveTagSuccess(t *testing.T) {
	r := empty()
	r.AddTag("core", []string{"github"})

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
	r.AddTag("core", []string{"github"})

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
