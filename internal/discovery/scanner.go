package discovery

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/hystak/hystak/internal/model"
)

// Candidate is an MCP server discovered from an existing config file.
// Callers must mask sensitive env values (TOKEN, SECRET, KEY, PASSWORD, CREDENTIAL)
// before displaying candidates to the user. The raw values are preserved for import.
type Candidate struct {
	Name   string
	Server model.ServerDef
	Source string // file path where it was discovered
}

// mcpServerJSON matches the JSON structure of MCP servers in .mcp.json / .claude.json.
type mcpServerJSON struct {
	Type    string            `json:"type"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// ScanFile reads a .mcp.json or .claude.json file and returns discovered
// MCP server candidates. Returns nil (not error) if the file doesn't exist.
// Returns an error only for permission or parse failures.
func ScanFile(path string) ([]Candidate, error) {
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		// file exists
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil
	default:
		return nil, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	serversRaw, ok := raw["mcpServers"]
	if !ok {
		return nil, nil
	}

	var servers map[string]mcpServerJSON
	if err := json.Unmarshal(serversRaw, &servers); err != nil {
		return nil, err
	}

	candidates := make([]Candidate, 0, len(servers))
	for name, s := range servers {
		candidates = append(candidates, Candidate{
			Name: name,
			Server: model.ServerDef{
				Name:      name,
				Transport: model.Transport(s.Type),
				Command:   s.Command,
				Args:      s.Args,
				Env:       s.Env,
				URL:       s.URL,
				Headers:   s.Headers,
			},
			Source: path,
		})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Name < candidates[j].Name })
	return candidates, nil
}

// ScanProject scans the project directory's .mcp.json for MCP servers.
func ScanProject(projectPath string) ([]Candidate, error) {
	return ScanFile(filepath.Join(projectPath, ".mcp.json"))
}

// ScanGlobal scans ~/.claude.json for globally configured MCP servers.
func ScanGlobal() ([]Candidate, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return ScanFile(filepath.Join(home, ".claude.json"))
}

// ScanAll scans both the project .mcp.json and ~/.claude.json,
// returning all discovered candidates. Duplicates (same name) from
// different sources are both included — the caller resolves conflicts.
func ScanAll(projectPath string) ([]Candidate, error) {
	var all []Candidate

	projectCandidates, err := ScanProject(projectPath)
	if err != nil {
		return nil, err
	}
	all = append(all, projectCandidates...)

	globalCandidates, err := ScanGlobal()
	if err != nil {
		return nil, err
	}
	all = append(all, globalCandidates...)

	return all, nil
}
