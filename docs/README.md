# Claude Agent SDK for Go

A Go SDK for interacting with the Claude Code CLI, providing a clean, idiomatic Go interface for building AI agents.

## Features

- **Clean API**: Simple Query and Client interfaces for both one-shot and streaming interactions
- **Type Safety**: Strongly typed message and content block structures
- **Hexagonal Architecture**: Domain logic isolated from infrastructure concerns
- **Lifecycle Hooks**: Pre/post tool use, session events, and custom hooks
- **Permission System**: Fine-grained control over tool usage with custom callbacks
- **MCP Integration**: Build and integrate Model Context Protocol servers
- **Channel-Based**: Idiomatic Go patterns using channels and context

## Installation

```bash
go get github.com/conneroisu/claude
```

### Prerequisites

- Go 1.21 or higher
- Claude Code CLI installed: `npm install -g @anthropic-ai/claude-code`
- Anthropic API key configured

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/conneroisu/claude/pkg/claude"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
)

func main() {
	ctx := context.Background()

	opts := &options.AgentOptions{
		MaxTurns: intPtr(1),
	}

	msgCh, errCh := claude.Query(ctx, "What is 2 + 2?", opts, nil)

	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				return
			}

			switch m := msg.(type) {
			case *messages.AssistantMessage:
				for _, block := range m.Content {
					if textBlock, ok := block.(messages.TextBlock); ok {
						fmt.Printf("Claude: %s\n", textBlock.Text)
					}
				}

			case *messages.ResultMessageSuccess:
				fmt.Printf("Completed in %dms\n", m.DurationMs)
			}

		case err := <-errCh:
			if err != nil {
				log.Fatal(err)
			}
			return
		}
	}
}

func intPtr(i int) *int { return &i }
```

## Architecture

The SDK follows hexagonal (ports and adapters) architecture:

```
┌─────────────────────────────────────────────┐
│           Public API (pkg/claude)           │
│  Query() | NewClient() | NewMCPServer()    │
└─────────────────────────────────────────────┘
                      │
┌─────────────────────────────────────────────┐
│              Domain Services                │
│  Querying | Hooking | Permissions          │
└─────────────────────────────────────────────┘
                      │
┌─────────────────────────────────────────────┐
│              Ports (Interfaces)             │
│  Transport | ProtocolHandler | Parser      │
└─────────────────────────────────────────────┘
                      │
┌─────────────────────────────────────────────┐
│               Adapters                      │
│  CLI Transport | JSON-RPC Protocol | Parse │
└─────────────────────────────────────────────┘
```

### Benefits

- **Testability**: Mock ports for unit testing without external dependencies
- **Flexibility**: Swap implementations (e.g., HTTP transport instead of CLI)
- **Maintainability**: Clear boundaries between domain logic and infrastructure

## Usage

### Streaming Conversations

For multi-turn conversations:

```go
client := claude.NewClient(&options.AgentOptions{
	MaxTurns: intPtr(10),
})

if err := client.Connect(ctx, nil); err != nil {
	log.Fatal(err)
}
defer client.Close()

if err := client.SendMessage(ctx, "Hello"); err != nil {
	log.Fatal(err)
}

msgCh, errCh := client.ReceiveMessages(ctx)
// Handle messages...
```

See [examples/streaming](../cmd/examples/streaming) for a complete example.

### Lifecycle Hooks

Execute custom logic before/after tool use:

```go
hookCfg := &options.HookConfig{
	PreToolUse: []options.HookMatcher{
		{
			Matcher: "Bash",
			Hooks: []options.HookCallback{
				claude.BlockBashPatternHook([]string{"rm -rf", "sudo"}),
			},
		},
	},
}

opts := &options.AgentOptions{
	Hooks: hookCfg,
}
```

See [examples/hooks](../cmd/examples/hooks) for a complete example.

### Permission Callbacks

Control tool usage with custom permission logic:

```go
permCfg := &permissions.PermissionsConfig{
	Mode: options.PermissionModeDefault,
	CanUseTool: func(
		ctx context.Context,
		toolName string,
		input map[string]any,
		permCtx permissions.ToolPermissionContext,
	) (permissions.PermissionResult, error) {
		if toolName == "Bash" {
			if cmd, ok := input["command"].(string); ok {
				if strings.Contains(cmd, "rm") {
					return &permissions.PermissionResultDeny{
						Message: "rm not allowed",
					}, nil
				}
			}
		}
		return &permissions.PermissionResultAllow{}, nil
	},
}

opts := &options.AgentOptions{
	Permissions: permCfg,
}
```

See [examples/permissions](../cmd/examples/permissions) for a complete example.

### MCP Servers

Create and integrate MCP servers:

```go
server := claude.NewMCPServer("weather-tools", "1.0.0")

weatherTool := &mcpsdk.Tool{
	Name:        "get_weather",
	Description: "Get weather for a city",
	InputSchema: mcpsdk.ToolInputSchema{
		Type: "object",
		Properties: map[string]mcpsdk.Property{
			"city": {
				Type:        "string",
				Description: "City name",
			},
		},
		Required: []string{"city"},
	},
}

claude.AddTool(server, weatherTool, handleWeather)

opts := &options.AgentOptions{
	MCPServers: map[string]options.MCPServerAdapter{
		"weather": server.HandleMessage,
	},
}
```

See [examples/mcp](../cmd/examples/mcp) for a complete example.

### Tool Filtering

Restrict which tools Claude can use:

```go
opts := &options.AgentOptions{
	AllowedTools: []string{"Read", "Glob", "Grep"},
	BlockedTools: []string{"Bash", "Write", "Edit"},
}
```

See [examples/tools](../cmd/examples/tools) for a complete example.

## Message Types

### Assistant Messages

```go
type AssistantMessage struct {
	Model   string
	Content []ContentBlock
}
```

### Content Blocks

- `TextBlock` - Text content from Claude
- `ToolUseBlock` - Tool invocation
- `ToolResultBlock` - Tool execution result
- `ThinkingBlock` - Extended thinking content

### Result Messages

```go
type ResultMessageSuccess struct {
	DurationMs int
	NumTurns   int
	SessionID  string
}

type ResultMessageError struct {
	Subtype    string
	DurationMs int
	NumTurns   int
	SessionID  string
	Error      ErrorDetails
}
```

## Configuration

### Agent Options

```go
type AgentOptions struct {
	Model              *string
	MaxTurns           *int
	MaxTokens          *int
	ThinkingEnabled    *bool
	ThinkingBudget     *int
	OutputFormat       *string
	AllowedTools       []string
	BlockedTools       []string
	Hooks              *HookConfig
	Permissions        *PermissionsConfig
	MCPServers         map[string]MCPServerAdapter
}
```

### Hook Events

- `PreToolUse` - Before tool execution
- `PostToolUse` - After tool execution
- `UserPromptSubmit` - When user submits a prompt
- `Stop` - When session stops
- `SubagentStop` - When subagent stops
- `PreCompact` - Before message compaction
- `Notification` - For notifications
- `SessionStart` - When session starts
- `SessionEnd` - When session ends

### Permission Modes

- `PermissionModeDefault` - Use callback or default to allow
- `PermissionModeBypassPermissions` - Skip all permission checks
- `PermissionModeAsk` - Prompt user for each tool use
- `PermissionModeAcceptEdits` - Auto-accept edit operations
- `PermissionModePlan` - Planning mode

## Examples

All examples are in [cmd/examples](../cmd/examples):

- **quickstart** - Basic query
- **streaming** - Multi-turn conversation
- **hooks** - Custom lifecycle hooks
- **mcp** - MCP server integration
- **permissions** - Permission callbacks
- **tools** - Tool filtering

## Testing

### Unit Tests

```bash
go test -v ./pkg/claude/...
go test -race ./pkg/claude/...
go test -cover ./pkg/claude/...
```

### Integration Tests

Requires Claude CLI to be installed:

```bash
go test -tags=integration -v ./tests/integration/...
```

## Development

### Project Structure

```
.
├── pkg/claude/
│   ├── client.go           # Public API
│   ├── query.go            # One-shot queries
│   ├── hooks.go            # Hook helpers
│   ├── mcp.go              # MCP helpers
│   ├── messages/           # Message types
│   ├── options/            # Configuration
│   ├── ports/              # Interfaces
│   ├── querying/           # Query service
│   ├── hooking/            # Hook service
│   ├── permissions/        # Permission service
│   └── adapters/           # Implementations
│       ├── cli/            # CLI transport
│       ├── jsonrpc/        # JSON-RPC protocol
│       └── parse/          # Message parser
├── cmd/examples/           # Examples
└── tests/integration/      # Integration tests
```

### Code Quality

- Files: Max 175 lines
- Functions: Max 25 lines
- Line length: Max 80 characters
- All functions have return type declarations
- Comprehensive error handling

## License

MIT

## Contributing

Contributions welcome! Please ensure:

1. All tests pass: `go test ./...`
2. Code is formatted: `go fmt ./...`
3. Linter passes: `golangci-lint run`
4. Follow existing code style and architecture

## Support

- GitHub Issues: https://github.com/conneroisu/claude/issues
- Documentation: https://pkg.go.dev/github.com/conneroisu/claude
