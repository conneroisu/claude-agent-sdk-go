## Phase 5: Advanced Features
### 5.1 Hooks Support (hooks.go)
Priority: Medium
The facade re-exports domain hook types from the `hooking` package for public API convenience:
```go
package claude

import (
	"fmt"
	"strings"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/permissions"
)

// Re-export domain hook types for public API
type HookEvent = hooking.HookEvent
type HookContext = hooking.HookContext
type HookCallback = hooking.HookCallback
type HookMatcher = hooking.HookMatcher

// Re-export hook event constants
const (
	HookEventPreToolUse       = hooking.HookEventPreToolUse
	HookEventPostToolUse      = hooking.HookEventPostToolUse
	HookEventUserPromptSubmit = hooking.HookEventUserPromptSubmit
	HookEventStop             = hooking.HookEventStop
	HookEventSubagentStop     = hooking.HookEventSubagentStop
	HookEventPreCompact       = hooking.HookEventPreCompact
)

// Re-export permissions types for public API
type PermissionsConfig = permissions.PermissionsConfig
type PermissionResult = permissions.PermissionResult
type CanUseToolFunc = permissions.CanUseToolFunc

type HookJSONOutput struct {
	Decision           *string        `json:"decision,omitempty"`           // "block"
	SystemMessage      *string        `json:"systemMessage,omitempty"`
	HookSpecificOutput map[string]any `json:"hookSpecificOutput,omitempty"`
}

// BlockBashPatternHook returns a hook callback that blocks bash commands containing forbidden patterns
func BlockBashPatternHook(patterns []string) HookCallback {
	return func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error) {
		toolName, _ := input["tool_name"].(string)
		if toolName != "Bash" {
			return map[string]any{}, nil
		}
		toolInput, _ := input["tool_input"].(map[string]any)
		command, _ := toolInput["command"].(string)
		for _, pattern := range patterns {
			if strings.Contains(command, pattern) {
				return map[string]any{
					"hookSpecificOutput": map[string]any{
						"hookEventName":            "PreToolUse",
						"permissionDecision":       "deny",
						"permissionDecisionReason": fmt.Sprintf("Command contains forbidden pattern: %s", pattern),
					},
				}, nil
			}
		}
		return map[string]any{}, nil
	}
}
```
### 5.2 MCP Server Support (mcp.go)
Priority: Medium
To support in-process user-defined tools, the SDK will provide a public API that wraps the official `github.com/modelcontextprotocol/go-sdk`. Instead of re-implementing the MCP server, the SDK will offer convenience functions to create and configure an `mcp.Server`.
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
The `options.SDKServerConfig` will be updated to hold the `*mcp.Server` instance. The agent SDK will then be responsible for internally creating an adapter that implements the `ports.MCPServer` interface. This adapter will manage a pair of in-memory transports to communicate with the user's `mcp.Server` instance, proxying messages received from the Claude CLI.
### 5.3 Permission Callbacks (permissions.go)
Priority: Medium
```go
package claude

import (
	"context"
)

type ToolPermissionContext struct {
	Suggestions []PermissionUpdate
}

type PermissionResult interface {
	permissionResult()
}

type PermissionResultAllow struct {
	UpdatedInput       map[string]any
	UpdatedPermissions []PermissionUpdate
}

type PermissionResultDeny struct {
	Message   string
	Interrupt bool
}

func (PermissionResultAllow) permissionResult() {}
func (PermissionResultDeny) permissionResult()  {}

type CanUseToolFunc func(ctx context.Context, toolName string, input map[string]any, permCtx ToolPermissionContext) (PermissionResult, error)

type PermissionUpdate struct {
	Type        string
	Rules       []PermissionRuleValue
	Behavior    *PermissionBehavior
	Mode        *PermissionMode
	Directories []string
	Destination *PermissionUpdateDestination
}

type PermissionRuleValue struct {
	ToolName    string
	RuleContent *string
}

type PermissionBehavior string

const (
	PermissionBehaviorAllow PermissionBehavior = "allow"
	PermissionBehaviorDeny  PermissionBehavior = "deny"
	PermissionBehaviorAsk   PermissionBehavior = "ask"
)

type PermissionMode string

type PermissionUpdateDestination string

const (
	PermissionDestinationUserSettings    PermissionUpdateDestination = "userSettings"
	PermissionDestinationProjectSettings PermissionUpdateDestination = "projectSettings"
	PermissionDestinationLocalSettings   PermissionUpdateDestination = "localSettings"
	PermissionDestinationSession         PermissionUpdateDestination = "session"
)
```