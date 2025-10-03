## Phase 5b: MCP Server Support

### Priority: Critical

### Overview
To support in-process user-defined tools, the SDK provides a public API that wraps the official `github.com/modelcontextprotocol/go-sdk`. Instead of re-implementing the MCP server, the SDK offers convenience functions to create and configure an `mcp.Server`.

### Public API (mcp.go)

```go
package claude

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewMCPServer creates a new in-process MCP server using the go-sdk.
// This server can be configured with tools and then passed to the Claude client
// via AgentOptions.
func NewMCPServer(name, version string) *mcp.Server {
	return mcp.NewServer(&mcp.Implementation{Name: name, Version: version}, nil)
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
//	myHandler := func(ctx context.Context, req *mcp.CallToolRequest, args myArgs) (*mcp.CallToolResult, myResult, error) {
//	    return nil, myResult{Res: "Result for " + args.Arg1}, nil
//	}
//	AddTool(server, &mcp.Tool{Name: "my_tool", Description: "My test tool"}, myHandler)
func AddTool[In, Out any](server *mcp.Server, tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) {
	mcp.AddTool(server, tool, handler)
}
```

### Integration Architecture

SDK-managed servers are registered separately to avoid storing server instances in configuration (preventing circular dependencies as established in Phase 1). The agent SDK maintains an internal registry that maps server names to adapters implementing `ports.MCPServer`.

**Key components:**
- User creates `*mcp.Server` using `NewMCPServer()` and `AddTool()`
- User registers server with SDK using a registration function (not via `options.SDKServerConfig`)
- SDK creates internal adapter implementing `ports.MCPServer`
- Adapter manages in-memory transports to communicate with user's server
- Adapter proxies messages from Claude CLI to user's MCP server

**Note:** The `options.SDKServerConfig` type is only used to mark that a server is SDK-managed. The actual server instances are managed separately by the MCP adapter layer to avoid circular dependencies.

---

## Implementation Notes

### File Size Requirements

**MCP integration in adapters/mcp/:**
- ✅ Likely 1-2 files (under 175 lines total)
- Adapter implementation should be straightforward

### Complexity Hotspots

**MCP message routing:**
- JSON-RPC handling → Reuse existing adapter patterns
- Message proxying → Extract proxy helper function
- Transport management → Use standard in-memory transport patterns

**Recommended patterns:**
- Leverage existing message routing from other adapters
- Create dedicated proxy helper for request/response forwarding
- Use go-sdk's built-in transport mechanisms

### Checklist

- [ ] MCP routing uses extracted helpers
- [ ] Adapter file(s) under 175 lines
- [ ] Transport setup follows standard patterns
- [ ] Error handling properly propagates from user's server

---

## Related Files
- [Phase 5a: Hooks Support](./07a_phase_5_hooks.md)
- [Phase 5c: Permission Callbacks](./07c_phase_5_permissions.md)
- [Phase 5: Integration Summary](./07d_phase_5_integration_summary.md)
