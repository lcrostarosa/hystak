package service

import (
	"testing"

	"github.com/hystak/hystak/internal/model"
)

func TestService_AddServer(t *testing.T) {
	svc, _ := setupTestService(t)

	err := svc.AddServer(model.ServerDef{
		Name:      "slack",
		Transport: model.TransportStdio,
		Command:   "npx",
	})
	if err != nil {
		t.Fatal(err)
	}

	srv, ok := svc.GetServer("slack")
	if !ok {
		t.Fatal("server 'slack' not found after add")
	}
	if srv.Command != "npx" {
		t.Errorf("Command = %q, want npx", srv.Command)
	}
}

func TestService_AddServer_Duplicate(t *testing.T) {
	svc, _ := setupTestService(t)

	err := svc.AddServer(model.ServerDef{
		Name:      "github",
		Transport: model.TransportStdio,
		Command:   "npx",
	})
	if err == nil {
		t.Fatal("expected error for duplicate server")
	}
}

func TestService_UpdateServer(t *testing.T) {
	svc, _ := setupTestService(t)

	err := svc.UpdateServer(model.ServerDef{
		Name:      "github",
		Transport: model.TransportStdio,
		Command:   "node",
	})
	if err != nil {
		t.Fatal(err)
	}

	srv, ok := svc.GetServer("github")
	if !ok {
		t.Fatal("server 'github' not found after update")
	}
	if srv.Command != "node" {
		t.Errorf("Command = %q, want node", srv.Command)
	}
}

func TestService_UpdateServer_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	err := svc.UpdateServer(model.ServerDef{
		Name:      "nonexistent",
		Transport: model.TransportStdio,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent server")
	}
}

func TestService_DeleteServer(t *testing.T) {
	svc, _ := setupTestService(t)

	if err := svc.DeleteServer("github"); err != nil {
		t.Fatal(err)
	}

	if _, ok := svc.GetServer("github"); ok {
		t.Error("server 'github' should not exist after delete")
	}
}

func TestService_DeleteServer_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	err := svc.DeleteServer("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent server")
	}
}

func TestService_GetServer(t *testing.T) {
	svc, _ := setupTestService(t)

	srv, ok := svc.GetServer("github")
	if !ok {
		t.Fatal("server 'github' not found")
	}
	if srv.Transport != model.TransportStdio {
		t.Errorf("Transport = %q, want stdio", srv.Transport)
	}
}

func TestService_GetServer_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	_, ok := svc.GetServer("nonexistent")
	if ok {
		t.Error("expected not found for nonexistent server")
	}
}
