package errors

import (
	"fmt"
	"testing"
)

func TestProjectNotFound_Error(t *testing.T) {
	e := &ProjectNotFound{Name: "myproject"}
	want := `project "myproject" not found`
	if got := e.Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestServerNotFound_Error(t *testing.T) {
	e := &ServerNotFound{Name: "github"}
	want := `server "github" not found`
	if got := e.Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResourceNotFound_Error(t *testing.T) {
	e := &ResourceNotFound{Kind: "skill", Name: "code-review"}
	want := `skill "code-review" not found`
	if got := e.Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAlreadyExists_Error(t *testing.T) {
	e := &AlreadyExists{Kind: "server", Name: "github"}
	want := `server "github" already exists`
	if got := e.Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *ValidationError
		want string
	}{
		{
			name: "with field",
			err:  &ValidationError{Field: "transport", Message: "must be stdio, sse, or http"},
			want: "validation error: transport: must be stdio, sse, or http",
		},
		{
			name: "without field",
			err:  &ValidationError{Message: "name is required"},
			want: "validation error: name is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfigParseError_Error(t *testing.T) {
	inner := fmt.Errorf("yaml: line 5: mapping values are not allowed here")
	e := &ConfigParseError{Path: "/home/user/.hystak/registry.yaml", Err: inner}
	want := `failed to parse config "/home/user/.hystak/registry.yaml": yaml: line 5: mapping values are not allowed here`
	if got := e.Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConfigParseError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("parse failure")
	e := &ConfigParseError{Path: "/test", Err: inner}
	if e.Unwrap() != inner {
		t.Error("Unwrap() did not return the wrapped error")
	}
}
