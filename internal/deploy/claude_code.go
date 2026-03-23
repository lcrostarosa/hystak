package deploy

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hystak/hystak/internal/config"
	"github.com/hystak/hystak/internal/model"
)

// Compile-time interface check.
var _ Deployer = (*ClaudeCodeDeployer)(nil)

// ClaudeCodeDeployer deploys MCP servers to Claude Code's .mcp.json.
type ClaudeCodeDeployer struct{}

func (d *ClaudeCodeDeployer) ClientType() model.ClientType {
	return model.ClientClaudeCode
}

func (d *ClaudeCodeDeployer) ConfigPath(projectPath string) string {
	return filepath.Join(projectPath, ".mcp.json")
}

// Bootstrap ensures .mcp.json exists with an empty mcpServers object.
func (d *ClaudeCodeDeployer) Bootstrap(projectPath string) error {
	path := d.ConfigPath(projectPath)
	_, err := os.Stat(path)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, fs.ErrNotExist):
		// create it
	default:
		return err
	}

	raw := map[string]interface{}{
		"mcpServers": map[string]interface{}{},
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return config.AtomicWrite(path, data, 0o644)
}

// mcpServerJSON is the JSON representation of a single MCP server in .mcp.json.
type mcpServerJSON struct {
	Type    string            `json:"type"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// ReadServers reads currently deployed MCP servers from .mcp.json.
func (d *ClaudeCodeDeployer) ReadServers(projectPath string) (map[string]model.ServerDef, error) {
	path := d.ConfigPath(projectPath)
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		// file exists
	case errors.Is(err, fs.ErrNotExist):
		return make(map[string]model.ServerDef), nil
	default:
		return nil, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	serversRaw, ok := raw["mcpServers"]
	if !ok {
		return make(map[string]model.ServerDef), nil
	}

	var servers map[string]mcpServerJSON
	if err := json.Unmarshal(serversRaw, &servers); err != nil {
		return nil, err
	}

	result := make(map[string]model.ServerDef, len(servers))
	for name, s := range servers {
		result[name] = model.ServerDef{
			Name:      name,
			Transport: model.Transport(s.Type),
			Command:   s.Command,
			Args:      s.Args,
			Env:       s.Env,
			URL:       s.URL,
			Headers:   s.Headers,
		}
	}
	return result, nil
}

// WriteServers writes the full server map to .mcp.json.
// Non-mcpServers keys in the existing file are preserved (S-038).
// The caller (service layer) is responsible for merging managed and
// unmanaged servers before calling this method.
func (d *ClaudeCodeDeployer) WriteServers(projectPath string, servers map[string]model.ServerDef) error {
	path := d.ConfigPath(projectPath)

	// Read existing file to preserve non-mcpServers keys
	var existing map[string]json.RawMessage
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(data, &existing); err != nil {
			return err
		}
	case errors.Is(err, fs.ErrNotExist):
		existing = make(map[string]json.RawMessage)
	default:
		return err
	}

	// Build the mcpServers object
	mcpServers := make(map[string]mcpServerJSON, len(servers))
	for name, srv := range servers {
		mcpServers[name] = mcpServerJSON{
			Type:    string(srv.Transport),
			Command: srv.Command,
			Args:    srv.Args,
			Env:     srv.Env,
			URL:     srv.URL,
			Headers: srv.Headers,
		}
	}

	serversData, err := json.Marshal(mcpServers)
	if err != nil {
		return err
	}
	existing["mcpServers"] = serversData

	outData, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}
	outData = append(outData, '\n')
	return config.AtomicWrite(path, outData, 0o644)
}
