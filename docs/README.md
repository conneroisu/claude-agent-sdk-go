# Claude Agent SDK for Go

A comprehensive Go SDK for building AI agents with Claude Code CLI, supporting both simple one-shot queries and complex bidirectional streaming conversations.

## Features

- **One-Shot Queries**: Simple, single-turn interactions with Claude
- **Bidirectional Streaming**: Interactive, multi-turn conversations
- **Lifecycle Hooks**: Intercept and modify agent behavior at key points
- **Permission System**: Control tool usage with custom callbacks
- **MCP Support**: Connect to Model Context Protocol servers (planned)
- **Type-Safe**: Strongly-typed messages and configurations
- **Hexagonal Architecture**: Clean separation between domain logic and infrastructure

## Installation

```bash
go get github.com/conneroisu/claude/pkg/claude
```

Requires the [Claude Code CLI](https://github.com/anthropics/claude-code) to be installed:

```bash
npm install -g @anthropics/claude-code
```

## Quick Start

### One-Shot Query

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
    opts := &options.AgentOptions{
        Model: stringPtr("claude-sonnet-4-5-20250929"),
    }

    ctx := context.Background()
    msgCh, errCh := claude.Query(ctx, "What is 2+2?", opts, nil)

    for {
        select {
        case msg, ok := <-msgCh:
            if !ok {
                return
            }
            if assistantMsg, ok := msg.(*messages.AssistantMessage); ok {
                for _, block := range assistantMsg.Content {
                    if textBlock, ok := block.(messages.TextBlock); ok {
                        fmt.Println(textBlock.Text)
                    }
                }
            }

        case err := <-errCh:
            if err != nil {
                log.Fatal(err)
            }
        }
    }
}

func stringPtr(s string) *string { return &s }
```

### Streaming Conversation

```go
package main

import (
    "context"
    "fmt"

    "github.com/conneroisu/claude/pkg/claude"
    "github.com/conneroisu/claude/pkg/claude/options"
)

func main() {
    opts := &options.AgentOptions{
        Model: stringPtr("claude-sonnet-4-5-20250929"),
    }

    client := claude.NewClient(opts, nil, nil)
    ctx := context.Background()

    prompt := "Hello, Claude!"
    if err := client.Connect(ctx, &prompt); err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Start receiving messages
    msgCh, errCh := client.ReceiveMessages(ctx)
    go handleMessages(msgCh, errCh)

    // Send additional messages
    client.SendMessage(ctx, "Can you help me with Go?")
}
```

### Using Hooks

```go
package main

import (
    "context"
    "fmt"

    "github.com/conneroisu/claude/pkg/claude"
    "github.com/conneroisu/claude/pkg/claude/options"
)

func main() {
    // Block dangerous bash commands
    hooks := map[claude.HookEvent][]claude.HookMatcher{
        claude.HookEventPreToolUse: {
            {
                Matcher: "Bash",
                Hooks: []claude.HookCallback{
                    claude.BlockBashPatternHook([]string{"rm -rf", "dd if="}),
                },
            },
        },
    }

    opts := &options.AgentOptions{
        Model: stringPtr("claude-sonnet-4-5-20250929"),
    }

    ctx := context.Background()
    config := &claude.QueryConfig{
        Hooks: hooks,
    }
    msgCh, errCh := claude.Query(ctx, "List files", opts, config)

    // Process messages...
}
```

## Architecture

This SDK follows hexagonal (ports and adapters) architecture:

```
┌─────────────────────────────────────────────────────────┐
│                     PUBLIC API                          │
│              (client.go, query.go)                      │
└─────────────────────────────────────────────────────────┘
                          │
┌─────────────────────────────────────────────────────────┐
│                   DOMAIN SERVICES                       │
│   • querying/    - One-shot query execution            │
│   • streaming/   - Bidirectional conversations         │
│   • hooking/     - Lifecycle hooks                     │
│   • permissions/ - Tool permission checking            │
└─────────────────────────────────────────────────────────┘
                          │
┌─────────────────────────────────────────────────────────┐
│                   PORTS (Interfaces)                    │
│   • Transport    - CLI communication                   │
│   • Protocol     - Control message handling            │
│   • Parser       - Message parsing                     │
│   • MCPServer    - MCP server integration              │
└─────────────────────────────────────────────────────────┘
                          │
┌─────────────────────────────────────────────────────────┐
│            ADAPTERS (Infrastructure)                    │
│   • cli/         - CLI subprocess transport            │
│   • jsonrpc/     - Control protocol handler            │
│   • parse/       - Message parser                      │
│   • mcp/         - MCP server wrapper                  │
└─────────────────────────────────────────────────────────┘
```

## Configuration

### Agent Options

```go
opts := &options.AgentOptions{
    // Model selection
    Model: stringPtr("claude-sonnet-4-5-20250929"),

    // Tool control
    AllowedTools:    []string{"Read", "Write"},
    DisallowedTools: []string{"Bash"},

    // Execution limits
    MaxTurns: intPtr(10),

    // Permissions
    PermissionMode: &options.PermissionModeDefault,

    // Session management
    ContinueConversation: true,
    Resume: stringPtr("session-id"),

    // Infrastructure
    Cwd: stringPtr("/path/to/workspace"),
    Env: map[string]string{"VAR": "value"},
}
```

## Message Types

The SDK provides strongly-typed messages:

- `UserMessage` - User input
- `AssistantMessage` - Assistant responses with content blocks
- `SystemMessage` - System-level messages (init, compact boundary)
- `ResultMessageSuccess` - Successful completion with usage stats
- `ResultMessageError` - Error information
- `StreamEvent` - Raw API streaming events

### Content Blocks

- `TextBlock` - Plain text content
- `ThinkingBlock` - Extended thinking (Claude 3.7+)
- `ToolUseBlock` - Tool execution requests
- `ToolResultBlock` - Tool execution results

## Examples

See the [`cmd/examples/`](../cmd/examples/) directory for complete examples:

- [`quickstart/`](../cmd/examples/quickstart/) - Simple one-shot query
- [`streaming/`](../cmd/examples/streaming/) - Interactive conversation
- [`hooks/`](../cmd/examples/hooks/) - Using lifecycle hooks
- [`mcp/`](../cmd/examples/mcp/) - MCP server integration (coming soon)

## Development

### Build

```bash
go build ./pkg/claude/...
```

### Run Examples

```bash
go run cmd/examples/quickstart/main.go "What is the weather?"
go run cmd/examples/streaming/main.go
go run cmd/examples/hooks/main.go
```

### Test

```bash
go test ./pkg/claude/...
```

## Architecture Principles

1. **Domain Independence**: Core domain never imports adapters or infrastructure code
2. **Ports Define Contracts**: Interfaces defined by domain needs, not external systems
3. **Adapters Implement Ports**: Infrastructure code implements domain-defined interfaces
4. **Dependency Direction**: Always flows inward (adapters → domain), never outward
5. **Context-Based Naming**: Packages named for what they provide, not what they contain

## Contributing

Contributions are welcome! Please ensure:

- All code follows Go conventions and passes `go vet`
- Tests are written for new functionality
- Examples are updated for new features
- The CHANGELOG is updated

## License

MIT License - see LICENSE file for details

## Credits

Based on the [Python SDK reference implementation](https://github.com/anthropics/claude-agent-sdk-python).
