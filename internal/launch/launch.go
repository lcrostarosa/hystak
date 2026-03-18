package launch

import (
	"fmt"
	"os/exec"

	"github.com/lcrostarosa/hystak/internal/model"
)

// defaultExecutables maps known client types to their executable names.
var defaultExecutables = map[model.ClientType]string{
	model.ClientClaudeCode:    "claude",
	model.ClientClaudeDesktop: "claude",
	model.ClientCursor:        "cursor",
}

// DefaultExecutable returns the executable name for a known ClientType.
func DefaultExecutable(ct model.ClientType) (string, error) {
	name, ok := defaultExecutables[ct]
	if !ok {
		return "", fmt.Errorf("no default executable for client type %q", ct)
	}
	return name, nil
}

// ResolveExecutable verifies that an executable exists on $PATH and returns its full path.
func ResolveExecutable(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("executable %q not found on $PATH", name)
	}
	return path, nil
}

