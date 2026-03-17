package model

// ClientType identifies a supported MCP client.
type ClientType string

const (
	ClientClaudeCode    ClientType = "claude-code"
	ClientClaudeDesktop ClientType = "claude-desktop"
	ClientCursor        ClientType = "cursor"
)
