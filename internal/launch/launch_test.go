package launch

import (
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
