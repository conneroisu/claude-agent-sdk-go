package ports

import "context"

// MCPServer defines what the domain needs from MCP server implementations
type MCPServer interface {
	Name() string
	Initialize(ctx context.Context, params any) (any, error)
	ListTools(ctx context.Context) ([]MCPTool, error)
	CallTool(ctx context.Context, name string, args map[string]any) (MCPToolResult, error)
	HandleNotification(ctx context.Context, method string, params any) error
}

// MCPTool represents an MCP tool definition
type MCPTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// MCPToolResult represents the result of calling an MCP tool
type MCPToolResult struct {
	Content []map[string]any `json:"content"`
	IsError bool             `json:"isError,omitempty"`
}
