package claude

import (
	"context"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/ports"
)

// MCPServer interface for in-process MCP servers
type MCPServer interface {
	Name() string
	Version() string
	ListTools(ctx context.Context) ([]MCPTool, error)
	CallTool(ctx context.Context, name string, args map[string]any) (MCPToolResult, error)
}

// MCPTool represents an MCP tool
type MCPTool struct {
	Name        string
	Description string
	InputSchema map[string]any
}

// MCPToolResult represents the result of calling an MCP tool
type MCPToolResult struct {
	Content []MCPContent
	IsError bool
}

// MCPContent represents MCP content
type MCPContent struct {
	Type     string
	Text     string
	Data     string
	MimeType string
}

// SDKMCPTool represents a tool decorated for SDK MCP servers
type SDKMCPTool struct {
	Name        string
	Description string
	InputSchema any
	Handler     func(context.Context, map[string]any) (map[string]any, error)
}

// Tool creates a new SDK MCP tool
func Tool(name, description string, inputSchema any, handler func(context.Context, map[string]any) (map[string]any, error)) *SDKMCPTool {
	return &SDKMCPTool{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
		Handler:     handler,
	}
}

type sdkMCPServer struct {
	name    string
	version string
	tools   map[string]*SDKMCPTool
}

// CreateSDKMCPServer creates both config and server instance
func CreateSDKMCPServer(name, version string, tools []*SDKMCPTool) (ports.MCPServer, error) {
	toolMap := make(map[string]*SDKMCPTool)
	for _, tool := range tools {
		toolMap[tool.Name] = tool
	}

	server := &sdkMCPServer{
		name:    name,
		version: version,
		tools:   toolMap,
	}

	return server, nil
}

func (s *sdkMCPServer) Name() string {
	return s.name
}

func (s *sdkMCPServer) Initialize(ctx context.Context, params any) (any, error) {
	return map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"serverInfo": map[string]any{
			"name":    s.name,
			"version": s.version,
		},
	}, nil
}

func (s *sdkMCPServer) ListTools(ctx context.Context) ([]ports.MCPTool, error) {
	var tools []ports.MCPTool
	for _, tool := range s.tools {
		schema := convertInputSchema(tool.InputSchema)
		tools = append(tools, ports.MCPTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: schema,
		})
	}

	return tools, nil
}

func (s *sdkMCPServer) CallTool(ctx context.Context, name string, args map[string]any) (ports.MCPToolResult, error) {
	tool, exists := s.tools[name]
	if !exists {
		return ports.MCPToolResult{}, fmt.Errorf("tool not found: %s", name)
	}

	result, err := tool.Handler(ctx, args)
	if err != nil {
		return ports.MCPToolResult{
			Content: []map[string]any{{"type": "text", "text": err.Error()}},
			IsError: true,
		}, nil
	}

	// Convert result to MCP format
	var content []map[string]any
	if contentList, ok := result["content"].([]any); ok {
		for _, item := range contentList {
			if itemMap, ok := item.(map[string]any); ok {
				content = append(content, itemMap)
			}
		}
	}

	isError, _ := result["is_error"].(bool)

	return ports.MCPToolResult{
		Content: content,
		IsError: isError,
	}, nil
}

func (s *sdkMCPServer) HandleNotification(ctx context.Context, method string, params any) error {
	// Handle notifications (currently no-op)
	return nil
}

func convertInputSchema(schema any) map[string]any {
	switch s := schema.(type) {
	case map[string]any:
		// Check if already JSON schema
		if _, hasType := s["type"]; hasType {
			return s
		}

		// Convert simple map to JSON schema
		properties := make(map[string]any)
		required := []string{}

		for name, typ := range s {
			required = append(required, name)
			switch typ {
			case "string":
				properties[name] = map[string]any{"type": "string"}
			case "int", "integer":
				properties[name] = map[string]any{"type": "integer"}
			case "number":
				properties[name] = map[string]any{"type": "number"}
			case "boolean":
				properties[name] = map[string]any{"type": "boolean"}
			default:
				properties[name] = map[string]any{"type": "string"}
			}
		}

		return map[string]any{
			"type":       "object",
			"properties": properties,
			"required":   required,
		}

	default:
		// Return empty object schema
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}
}
