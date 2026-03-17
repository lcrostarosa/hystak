package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rbbydotdev/hystak/internal/model"
)

// ClaudeCodeDeployer reads and writes MCP server configs for Claude Code.
//
// Two scopes:
//   - Project scope: .mcp.json in the project directory
//   - Global scope: ~/.claude.json (when projectPath is empty or "~")
type ClaudeCodeDeployer struct{}

// claudeCodeServer is the JSON representation of a server in Claude Code config.
type claudeCodeServer struct {
	Type    string            `json:"type"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

func (d *ClaudeCodeDeployer) ClientType() model.ClientType {
	return model.ClientClaudeCode
}

// ConfigPath returns the config file path.
// Empty or "~" projectPath means global scope (~/.claude.json).
// Otherwise, returns <projectPath>/.mcp.json.
func (d *ClaudeCodeDeployer) ConfigPath(projectPath string) string {
	if isGlobalScope(projectPath) {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".claude.json")
	}
	return filepath.Join(projectPath, ".mcp.json")
}

// ReadServers reads MCP servers from the Claude Code config file.
func (d *ClaudeCodeDeployer) ReadServers(projectPath string) (map[string]model.ServerDef, error) {
	configPath := d.ConfigPath(projectPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]model.ServerDef{}, nil
		}
		return nil, fmt.Errorf("reading %s: %w", configPath, err)
	}

	// Parse into a generic structure to extract mcpServers.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", configPath, err)
	}

	serversRaw, ok := raw["mcpServers"]
	if !ok {
		return map[string]model.ServerDef{}, nil
	}

	var ccServers map[string]claudeCodeServer
	if err := json.Unmarshal(serversRaw, &ccServers); err != nil {
		return nil, fmt.Errorf("parsing mcpServers in %s: %w", configPath, err)
	}

	result := make(map[string]model.ServerDef, len(ccServers))
	for name, ccs := range ccServers {
		result[name] = fromClaudeCode(name, ccs)
	}
	return result, nil
}

// WriteServers writes MCP servers to the Claude Code config file,
// preserving all non-mcpServers keys.
func (d *ClaudeCodeDeployer) WriteServers(projectPath string, servers map[string]model.ServerDef) error {
	configPath := d.ConfigPath(projectPath)

	// Read existing file to preserve other keys.
	var raw map[string]json.RawMessage
	data, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("reading %s: %w", configPath, err)
		}
		raw = make(map[string]json.RawMessage)
	} else {
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("parsing %s: %w", configPath, err)
		}
	}

	// Convert servers to Claude Code format.
	ccServers := make(map[string]claudeCodeServer, len(servers))
	for name, srv := range servers {
		ccServers[name] = toClaudeCode(srv)
	}

	serversJSON, err := json.Marshal(ccServers)
	if err != nil {
		return fmt.Errorf("marshaling mcpServers: %w", err)
	}
	raw["mcpServers"] = serversJSON

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", configPath, err)
	}

	if err := os.WriteFile(configPath, append(out, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", configPath, err)
	}
	return nil
}

// Bootstrap creates the config file if it doesn't exist.
func (d *ClaudeCodeDeployer) Bootstrap(projectPath string) error {
	configPath := d.ConfigPath(projectPath)

	if _, err := os.Stat(configPath); err == nil {
		return nil // already exists
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", configPath, err)
	}

	content := []byte("{\"mcpServers\":{}}\n")
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", configPath, err)
	}
	return nil
}

// isGlobalScope returns true if the project path indicates global scope.
func isGlobalScope(projectPath string) bool {
	return projectPath == "" || projectPath == "~"
}

// toClaudeCode converts a canonical ServerDef to Claude Code JSON format.
func toClaudeCode(s model.ServerDef) claudeCodeServer {
	return claudeCodeServer{
		Type:    string(s.Transport),
		Command: s.Command,
		Args:    s.Args,
		Env:     s.Env,
		URL:     s.URL,
		Headers: s.Headers,
	}
}

// fromClaudeCode converts a Claude Code JSON server to a canonical ServerDef.
func fromClaudeCode(name string, ccs claudeCodeServer) model.ServerDef {
	return model.ServerDef{
		Name:      name,
		Transport: model.Transport(ccs.Type),
		Command:   ccs.Command,
		Args:      ccs.Args,
		Env:       ccs.Env,
		URL:       ccs.URL,
		Headers:   ccs.Headers,
	}
}
