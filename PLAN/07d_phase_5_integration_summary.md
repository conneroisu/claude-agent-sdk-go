## Phase 5: Integrations Summary

### Overview
Phase 5 adds three critical integration capabilities to the Claude Agent SDK, enabling users to customize agent behavior and add custom tools.

### Integration Components

#### **5a. Hooks Support** (Critical Priority)
[📄 Full Documentation](./07a_phase_5_hooks.md)

Lifecycle hooks that execute at key points in agent execution:
- `PreToolUse` - Before tool execution (can block/modify)
- `PostToolUse` - After tool execution
- `UserPromptSubmit` - When user submits a prompt
- `Stop` - When agent stops
- `SubagentStop` - When subagent stops
- `PreCompact` - Before context compaction

**Key file:** `hooks.go` in public API

#### **5b. MCP Server Support** (Critical Priority)
[📄 Full Documentation](./07b_phase_5_mcp_servers.md)

In-process user-defined tools via Model Context Protocol:
- **Two integration modes:**
  - External servers: Connect TO external MCP servers via stdio/HTTP/SSE
  - SDK servers: User creates in-process servers with direct state access
- Wraps official `github.com/modelcontextprotocol/go-sdk`
- Provides `NewMCPServer()` and `AddTool()` convenience functions
- Type-safe tool definitions using Go generics
- Automatic JSON schema inference from struct types
- In-memory transport for zero IPC overhead (SDK servers)
- Unified `ports.MCPServer` interface for both modes

**Key files:**
- `mcp.go` - Public API (NewMCPServer, AddTool)
- `adapters/mcp/client.go` - External server adapter
- `adapters/mcp/sdk_server.go` - SDK server adapter

#### **5c. Permission Callbacks** (Medium Priority)
[📄 Full Documentation](./07c_phase_5_permissions.md)

Custom authorization logic for tool usage:
- `CanUseToolFunc` callback interface
- Can allow, deny, or modify tool requests
- Supports permission updates and suggestions

**Key file:** `permissions.go` in public API

---

## Cross-Cutting Implementation Guidance

### Integration Points
- **Hooks** integrate with tool execution pipeline in agent core
- **MCP servers** integrate via adapter pattern in `adapters/mcp/`
- **Permissions** integrate with tool authorization in agent core

### Shared Patterns
- All three use callback/interface patterns for user extensibility
- All expose simple public APIs that wrap complex internal machinery
- All follow the SDK's adapter pattern for protocol translation

### Dependencies
- Hooks may trigger permission checks
- Permission callbacks may generate hook events
- MCP tools participate in both hooks and permissions

---

## End-to-End Integration Example

Example showing all three integrations working together:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/conneroisu/claude/pkg/claude"
	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	ctx := context.Background()

	// 1. Create SDK MCP server with custom tools
	server := claude.NewMCPServer("api-client", "1.0")

	type APICallArgs struct {
		Endpoint string `json:"endpoint" jsonschema:"description=API endpoint to call"`
		Method   string `json:"method" jsonschema:"description=HTTP method"`
	}

	apiHandler := func(ctx context.Context, req *mcpsdk.CallToolRequest, args APICallArgs) (*mcpsdk.CallToolResult, struct{ Response string }, error) {
		// Make actual API call here
		return nil, struct{ Response string }{Response: "API response"}, nil
	}

	claude.AddTool(server, &mcpsdk.Tool{
		Name:        "call_api",
		Description: "Make API calls to external services",
	}, apiHandler)

	// 2. Set up hooks for logging and monitoring
	hooks := map[hooking.HookEvent][]hooking.HookMatcher{
		hooking.HookEventPreToolUse: {
			{
				Matcher: "call_api",
				Hooks: []hooking.HookCallback{
					func(input hooking.HookInput) (hooking.HookOutput, error) {
						log.Printf("About to call API: %v", input)
						// Optionally block or modify the call
						return hooking.HookOutput{Continue: true}, nil
					},
				},
			},
		},
		hooking.HookEventPostToolUse: {
			{
				Hooks: []hooking.HookCallback{
					func(input hooking.HookInput) (hooking.HookOutput, error) {
						log.Printf("Tool executed: %v", input)
						return hooking.HookOutput{}, nil
					},
				},
			},
		},
	}

	// 3. Set up permission callback for authorization
	canUseTool := func(toolName string, input map[string]any, ctx permissions.Context) (permissions.Result, error) {
		// Custom authorization logic
		if toolName == "call_api" {
			endpoint, _ := input["endpoint"].(string)
			if isAllowedEndpoint(endpoint) {
				return permissions.ResultAllow{}, nil
			}
			return permissions.ResultDeny{
				Message: fmt.Sprintf("Endpoint %s not allowed", endpoint),
			}, nil
		}
		return permissions.ResultAllow{}, nil
	}

	// 4. Configure Claude with all three integrations
	opts := &options.AgentOptions{
		MCPServers: map[string]options.MCPServerConfig{
			"api": options.SDKServerConfig{
				Type:     "sdk",
				Name:     "api",
				Instance: server,
			},
		},
		AllowedTools: []options.BuiltinTool{
			options.ToolMcp,
		},
	}

	permsConfig := &permissions.PermissionsConfig{
		CanUseTool: canUseTool,
	}

	// 5. Execute query with all integrations active
	client := claude.NewClient(opts, hooks, permsConfig)
	if err := client.Connect(ctx, nil); err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Send message and receive responses
	if err := client.SendMessage(ctx, "Use the API to fetch user data"); err != nil {
		log.Fatal(err)
	}

	msgCh, errCh := client.ReceiveMessages(ctx)
	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				return
			}
			fmt.Printf("Response: %v\n", msg)

		case err, ok := <-errCh:
			if !ok {
				return
			}
			if err != nil {
				log.Printf("Error: %v", err)
			}
		}
	}
}

func isAllowedEndpoint(endpoint string) bool {
	// Authorization logic
	allowedEndpoints := []string{"/api/users", "/api/data"}
	for _, allowed := range allowedEndpoints {
		if endpoint == allowed {
			return true
		}
	}
	return false
}
```

**Integration Flow:**
1. User defines custom tools via SDK MCP server
2. Hooks log and monitor tool usage
3. Permission callback authorizes tool calls
4. All three work together seamlessly via hexagonal architecture

---

## Overall Checklist

### Code Organization
- [ ] All files under 175 lines (linting requirement)
- [ ] Functions under 25 lines where possible
- [ ] Complex logic extracted to helper functions
- [ ] Type switching replaced with handler maps

### Testing
- [ ] Unit tests for all hook types
- [ ] Integration tests for MCP server communication
- [ ] Permission callback authorization tests
- [ ] Cross-feature integration tests (hooks + permissions)

### Documentation
- [ ] Public API fully documented with examples
- [ ] Integration patterns documented
- [ ] Error handling documented
- [ ] Migration guide for users (if applicable)

### Performance
- [ ] Hook execution optimized (minimal overhead)
- [ ] MCP message routing efficient
- [ ] Permission checks don't block unnecessarily

---

## Implementation Order

**Recommended sequence:**
1. **Permissions** (simplest, foundational)
2. **Hooks** (depends on permissions)
3. **MCP** (can leverage hook/permission infrastructure)

This order minimizes rework and allows testing of each integration independently before combining them.
