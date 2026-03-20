package launch

import (
	"runtime"
	"testing"

	"github.com/lcrostarosa/hystak/internal/model"
)

func TestDefaultExecutable(t *testing.T) {
	tests := []struct {
		ct      model.ClientType
		want    string
		wantErr bool
	}{
		{model.ClientClaudeCode, "claude", false},
		{model.ClientClaudeDesktop, "claude", false},
		{model.ClientCursor, "cursor", false},
		{model.ClientType("unknown"), "", true},
	}

	for _, tt := range tests {
		t.Run(string(tt.ct), func(t *testing.T) {
			got, err := DefaultExecutable(tt.ct)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DefaultExecutable(%q) error = %v, wantErr %v", tt.ct, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("DefaultExecutable(%q) = %q, want %q", tt.ct, got, tt.want)
			}
		})
	}
}

func TestResolveExecutable_Found(t *testing.T) {
	// /bin/sh should exist on all Unix systems.
	path, err := ResolveExecutable("sh")
	if err != nil {
		t.Fatalf("ResolveExecutable(\"sh\") error = %v", err)
	}
	if path == "" {
		t.Error("ResolveExecutable(\"sh\") returned empty path")
	}
}

func TestResolveExecutable_NotFound(t *testing.T) {
	_, err := ResolveExecutable("hystak-nonexistent-binary-xyz")
	if err == nil {
		t.Fatal("ResolveExecutable for nonexistent binary should error")
	}
}

func TestRunCommand_ExitZero(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh not available on Windows")
	}
	shPath, err := ResolveExecutable("sh")
	if err != nil {
		t.Fatal(err)
	}
	code, err := RunCommand(shPath, []string{"-c", "exit 0"}, t.TempDir())
	if err != nil {
		t.Fatalf("RunCommand error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestRunCommand_ExitNonZero(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh not available on Windows")
	}
	shPath, err := ResolveExecutable("sh")
	if err != nil {
		t.Fatal(err)
	}
	code, err := RunCommand(shPath, []string{"-c", "exit 42"}, t.TempDir())
	if err != nil {
		t.Fatalf("RunCommand error: %v", err)
	}
	if code != 42 {
		t.Errorf("expected exit code 42, got %d", code)
	}
}

func TestRunCommand_StdioForwarding(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh not available on Windows")
	}
	shPath, err := ResolveExecutable("sh")
	if err != nil {
		t.Fatal(err)
	}
	// Just verify it doesn't error — stdout goes to os.Stdout which is fine in tests.
	code, err := RunCommand(shPath, []string{"-c", "echo hello >/dev/null"}, t.TempDir())
	if err != nil {
		t.Fatalf("RunCommand error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestRunCommand_InvalidExecutable(t *testing.T) {
	_, err := RunCommand("/nonexistent/binary", nil, t.TempDir())
	if err == nil {
		t.Fatal("expected error for nonexistent executable")
	}
}
