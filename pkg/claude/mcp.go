// Package claude provides a high-level API for interacting with Claude.
package claude

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewMCPServer creates a new in-process MCP server using the go-sdk.
// This server can be configured with tools and then passed to the Claude
// client via AgentOptions.
//
// Example:
//
//	server := claude.NewMCPServer("calculator", "1.0")
//	claude.AddTool(server, &mcp.Tool{
//	    Name: "add",
//	    Description: "Add numbers",
//	}, addHandler)
//
//	opts := &options.AgentOptions{
//	    MCPServers: map[string]options.MCPServerConfig{
//	        "calc": options.SDKServerConfig{
//	            Type: "sdk",
//	            Name: "calc",
//	            Instance: server,
//	        },
//	    },
//	}
func NewMCPServer(name, version string) *mcp.Server {
	return mcp.NewServer(
		&mcp.Implementation{
			Name:    name,
			Version: version,
		},
		nil,
	)
}

// AddTool is a convenience wrapper around the go-sdk's generic AddTool
// function. It allows users to add a tool with a typed handler to an
// mcp.Server instance, benefiting from automatic schema inference and
// validation provided by the go-sdk.
//
// The Go MCP SDK uses generics to automatically infer JSON schema from
// the Args and Result type parameters. Struct tags (json, jsonschema)
// control schema generation.
//
// Example:
//
//	type AddArgs struct {
//	    A float64 `json:"a" jsonschema:"description=First number"`
//	    B float64 `json:"b" jsonschema:"description=Second number"`
//	}
//
//	type AddResult struct {
//	    Sum float64 `json:"sum"`
//	}
//
//	addHandler := func(
//	    ctx context.Context,
//	    req *mcp.CallToolRequest,
//	    args AddArgs,
//	) (*mcp.CallToolResult, AddResult, error) {
//	    return nil, AddResult{Sum: args.A + args.B}, nil
//	}
//
//	server := NewMCPServer("calculator", "1.0")
//	AddTool(server, &mcp.Tool{
//	    Name: "add",
//	    Description: "Add two numbers",
//	}, addHandler)
func AddTool[In, Out any](
	server *mcp.Server,
	tool *mcp.Tool,
	handler mcp.ToolHandlerFor[In, Out],
) {
	mcp.AddTool(server, tool, handler)
}
