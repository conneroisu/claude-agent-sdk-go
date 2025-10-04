# Claude Agent SDK for Go

[![Go Version](https://img.shields.io/github/go-mod/go-version/conneroisu/claude-agent-sdk-go)](go.mod)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Go SDK for building AI agents with Claude. This SDK provides a clean, type-safe interface to Claude's agent capabilities including tool use, hooks, MCP servers, and streaming conversations.

## Features

- 🎯 **Type-safe API** - Full Go type safety with discriminated unions
- 🔌 **Hexagonal Architecture** - Clean separation of concerns
- 🪝 **Lifecycle Hooks** - Intercept and modify tool calls
- 🔧 **Tool Support** - Built-in and custom tools
- 🌐 **MCP Integration** - Connect to Model Context Protocol servers
- 🔐 **Permission System** - Fine-grained control over tool usage
- 📡 **Streaming** - Bidirectional streaming conversations
- ⚡ **Context-aware** - Full context.Context support

## Installation

```bash
go get github.com/conneroisu/claude-agent-sdk-go
```

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

    // Simple one-shot query
    maxTurns := 1
    msgCh, errCh := claude.Query(
        ctx,
        "What is 2 + 2?",
        &options.AgentOptions{MaxTurns: &maxTurns},
        nil,
    )

    // Process responses
    for {
        select {
        case msg := <-msgCh:
            if msg == nil {
                return
            }
            if assistantMsg, ok := msg.(*messages.AssistantMessage); ok {
                for _, block := range assistantMsg.Content {
                    if textBlock, ok := block.(messages.TextBlock); ok {
                        fmt.Printf("Claude: %s\n", textBlock.Text)
                    }
                }
            }
        case err := <-errCh:
            if err != nil {
                log.Fatal(err)
            }
            return
        }
    }
}
```

## Usage Examples

### Streaming Conversation

```go
client, _ := claude.NewClient(&options.AgentOptions{})
client.Connect(ctx, nil)
defer client.Close()

// Send messages
client.SendMessage(ctx, "Hello!")

// Receive responses
msgCh, errCh := client.ReceiveMessages(ctx)
for msg := range msgCh {
    // Process messages
}
```

### Hooks

Intercept tool calls before and after execution:

```go
hooks := map[claude.HookEvent][]claude.HookMatcher{
    claude.HookEventPreToolUse: {
        {
            Matcher: "Bash",
            Hooks: []claude.HookCallback{
                func(input map[string]any, toolUseID *string, ctx claude.HookContext) (map[string]any, error) {
                    fmt.Printf("About to run: %v\n", input)
                    return map[string]any{}, nil
                },
            },
        },
    },
}

msgCh, errCh := claude.Query(ctx, "List files", opts, hooks)
```

### Tool Filtering

Control which tools Claude can use:

```go
opts := &options.AgentOptions{
    AllowedTools: []options.BuiltinTool{
        options.ToolBash,
        options.ToolRead,
    },
}
```

### Custom Permissions

Implement custom permission logic:

```go
permsConfig := &permissions.Config{
    Mode: options.PermissionModeAsk,
    CanUseTool: func(ctx context.Context, toolName string, input map[string]any, permCtx permissions.ToolPermissionContext) (permissions.PermissionResult, error) {
        // Custom permission logic
        return &permissions.PermissionResultAllow{}, nil
    },
}
```

## Architecture

This SDK follows hexagonal (ports and adapters) architecture:

```
pkg/claude/
├── messages/      # Domain models (messages, content blocks)
├── options/       # Configuration and options
├── ports/         # Interface definitions
├── adapters/      # Infrastructure implementations
│   ├── cli/       # Claude CLI transport
│   ├── jsonrpc/   # JSON-RPC protocol
│   ├── mcp/       # MCP client/server
│   └── parse/     # Message parsing
├── hooking/       # Hook management service
├── permissions/   # Permission checking service
├── querying/      # One-shot query service
└── streaming/     # Streaming conversation service
```

### Key Concepts

- **Ports**: Interfaces defining contracts (Transport, ProtocolHandler, MessageParser)
- **Adapters**: Concrete implementations of ports
- **Domain Services**: Business logic (querying, streaming, hooking, permissions)
- **Public API**: Facade layer in `pkg/claude/client.go`

## Code Quality

This codebase enforces strict quality standards:

- Maximum 175 lines per file
- Maximum 25 lines per function
- Maximum 80 characters per line
- Maximum cognitive complexity 20
- Maximum nesting depth 3
- Minimum 15% comment density
- 100% golangci-lint compliance

## Testing

```bash
# Run all tests
go test ./pkg/claude/...

# Run with coverage
go test -cover ./pkg/claude/...

# Run integration tests (requires Claude CLI)
go test -tags=integration ./tests/integration/...
```

## Examples

See [cmd/examples/](../cmd/examples/) for complete examples:

- [quickstart](../cmd/examples/quickstart/) - Basic query
- [streaming](../cmd/examples/streaming/) - Bidirectional conversation
- [hooks](../cmd/examples/hooks/) - Tool use hooks
- [tools](../cmd/examples/tools/) - Tool filtering

## Contributing

Contributions welcome! Please ensure:

1. All tests pass (`go test ./...`)
2. Code is formatted (`nix fmt` or `gofmt`)
3. Linting passes (`golangci-lint run`)
4. Files respect size limits (175 lines)
5. Functions respect complexity limits (25 lines, complexity 20)

## License

MIT License - see [LICENSE](../LICENSE) for details

## Acknowledgments

Built with reference to the [TypeScript SDK](https://github.com/anthropics/anthropic-sdk-typescript)
