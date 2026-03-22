package launch

import (
	"os/exec"
	"testing"
)

func TestRunCommand_Success(t *testing.T) {
	err := RunCommand("true", nil, t.TempDir())
	if err != nil {
		t.Errorf("RunCommand(true) = %v, want nil", err)
	}
}

func TestRunCommand_NonZeroExit(t *testing.T) {
	err := RunCommand("false", nil, t.TempDir())
	if err == nil {
		t.Fatal("RunCommand(false) = nil, want error")
	}
	if _, ok := err.(*exec.ExitError); !ok {
		t.Errorf("expected *exec.ExitError, got %T", err)
	}
}

func TestRunCommand_NotFound(t *testing.T) {
	err := RunCommand("nonexistent-binary-hystak-test", nil, t.TempDir())
	if err == nil {
		t.Error("expected error for nonexistent binary")
	}
}

func TestRunCommand_WithArgs(t *testing.T) {
	err := RunCommand("echo", []string{"hello", "world"}, t.TempDir())
	if err != nil {
		t.Errorf("RunCommand(echo hello world) = %v, want nil", err)
	}
}

func TestFindClient_Exists(t *testing.T) {
	path, err := FindClient("echo")
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
}

func TestFindClient_NotFound(t *testing.T) {
	_, err := FindClient("nonexistent-binary-hystak-test")
	if err == nil {
		t.Error("expected error for nonexistent binary")
	}
}

func TestDefaultClientCommand(t *testing.T) {
	cmd := DefaultClientCommand()
	if cmd != "claude" {
		t.Errorf("DefaultClientCommand() = %q, want claude", cmd)
	}
}
