package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// mcpJSON is the minimal structure of a .mcp.json file.
type mcpJSON struct {
	MCPServers map[string]interface{} `json:"mcpServers"`
}

// claudeJSON is the minimal structure of ~/.claude.json.
type claudeJSON struct {
	MCPServers map[string]interface{} `json:"mcpServers"`
}

// BootstrapMCPJSON ensures a .mcp.json file exists at the given project path.
// Creates it with an empty mcpServers object if it does not exist.
func BootstrapMCPJSON(projectPath string) error {
	path := filepath.Join(projectPath, ".mcp.json")
	_, err := os.Stat(path)
	switch {
	case err == nil:
		return nil // already exists
	case errors.Is(err, fs.ErrNotExist):
		// create it
	default:
		return err
	}

	content := mcpJSON{
		MCPServers: make(map[string]interface{}),
	}
	data, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return AtomicWrite(path, data, 0o644)
}

// BootstrapClaudeJSON ensures ~/.claude.json exists.
// Creates it with an empty mcpServers object if it does not exist.
func BootstrapClaudeJSON() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, ".claude.json")
	_, statErr := os.Stat(path)
	switch {
	case statErr == nil:
		return nil // already exists
	case errors.Is(statErr, fs.ErrNotExist):
		// create it
	default:
		return statErr
	}

	content := claudeJSON{
		MCPServers: make(map[string]interface{}),
	}
	data, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return AtomicWrite(path, data, 0o644)
}
