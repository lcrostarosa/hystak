package deploy

import (
	"strings"
	"testing"

	"github.com/lcrostarosa/hystak/internal/model"
)

func TestNewDeployerClaudeCode(t *testing.T) {
	d, err := NewDeployer(model.ClientClaudeCode)
	if err != nil {
		t.Fatalf("NewDeployer(ClientClaudeCode) returned unexpected error: %v", err)
	}
	if d == nil {
		t.Fatal("NewDeployer(ClientClaudeCode) returned nil deployer")
	}
	if d.ClientType() != model.ClientClaudeCode {
		t.Errorf("ClientType() = %s, want %s", d.ClientType(), model.ClientClaudeCode)
	}
	// Verify the concrete type is ClaudeCodeDeployer.
	if _, ok := d.(*ClaudeCodeDeployer); !ok {
		t.Errorf("expected *ClaudeCodeDeployer, got %T", d)
	}
}

func TestNewDeployerUnimplemented(t *testing.T) {
	unimplementedClients := []model.ClientType{
		model.ClientClaudeDesktop,
		model.ClientCursor,
	}

	for _, ct := range unimplementedClients {
		t.Run(string(ct), func(t *testing.T) {
			d, err := NewDeployer(ct)
			if err == nil {
				t.Fatalf("NewDeployer(%s) should return error for unimplemented client", ct)
			}
			if d != nil {
				t.Errorf("NewDeployer(%s) should return nil deployer, got %T", ct, d)
			}
			if !strings.Contains(err.Error(), "not yet implemented") {
				t.Errorf("error should mention 'not yet implemented', got: %v", err)
			}
		})
	}
}

func TestNewDeployerUnknown(t *testing.T) {
	d, err := NewDeployer(model.ClientType("unknown-client"))
	if err == nil {
		t.Fatal("NewDeployer(unknown-client) should return error for unknown client type")
	}
	if d != nil {
		t.Errorf("NewDeployer(unknown-client) should return nil deployer, got %T", d)
	}
	if !strings.Contains(err.Error(), "unknown client type") {
		t.Errorf("error should mention 'unknown client type', got: %v", err)
	}
}
