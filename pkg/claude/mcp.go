package claude

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewMCPServer creates a new in-process MCP server using the go-sdk.
// This server can be configured with tools and then passed to the Claude client
// via AgentOptions.
func NewMCPServer(name, version string) *mcpsdk.Server {
	return mcpsdk.NewServer(&mcpsdk.Implementation{Name: name, Version: version}, nil)
}

// AddTool is a convenience wrapper around the go-sdk's generic AddTool function.
// It allows users to add a tool with a typed handler to an mcp.Server instance,
// benefiting from automatic schema inference and validation provided by the go-sdk.
//
// Example:
//
//	server := NewMCPServer("my-server", "1.0")
//	type myArgs struct { Arg1 string `json:"arg1"` }
//	type myResult struct { Res string `json:"res"` }
//	myHandler := func(ctx context.Context, req *mcpsdk.CallToolRequest, args myArgs) (*mcpsdk.CallToolResult, myResult, error) {
//	    return nil, myResult{Res: "Result for " + args.Arg1}, nil
//	}
//	AddTool(server, &mcpsdk.Tool{Name: "my_tool", Description: "My test tool"}, myHandler)
func AddTool[In, Out any](server *mcpsdk.Server, tool *mcpsdk.Tool, handler mcpsdk.ToolHandlerFor[In, Out]) {
	mcpsdk.AddTool(server, tool, handler)
}
