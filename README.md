# Claude Agent SDK for Go

Official Go SDK for Claude Agent, providing programmatic access to Claude's agent capabilities.

## Installation

```bash
go get github.com/conneroisu/claude-agent-sdk-go
```

## Prerequisites

- Go 1.23 or later
- Claude Code CLI installed and available in PATH
- ANTHROPIC_API_KEY environment variable set

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/conneroisu/claude-agent-sdk-go/pkg/claude"
)

const (
	// maxTurns is the maximum number of conversation turns allowed.
	maxTurns = 5
)

func main() {
	ctx := context.Background()

	// Create client with basic options
	opts := &claude.Options{
		Model:    "claude-sonnet-4-5",
		MaxTurns: maxTurns,
	}

	client, err := claude.NewClient(opts)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			log.Printf("Failed to close client: %v", closeErr)
		}
	}()

	// Send a simple query
	fmt.Println("Sending query: What is 2+2?")
	query := "What is 2+2? Just respond with the number."
	if err = client.Query(ctx, query); err != nil {
		log.Printf("Failed to send query: %v", err)

		return
	}

	// Receive and process responses
	msgChan, errChan := client.ReceiveMessages(ctx)

	for {
		select {
		case msg := <-msgChan:
			if msg == nil {
				fmt.Println("Query completed")

				return
			}

			handleMessage(msg)

		case err := <-errChan:
			if err != nil {
				log.Printf("Error: %v", err)

				return
			}
		}
	}
}

// handleMessage processes different types of SDK messages.
func handleMessage(msg claude.SDKMessage) {
	switch m := msg.(type) {
	case *claude.SDKSystemMessage:
		if m.Subtype == "init" {
			fmt.Printf("Initialized with model: %v\n", m.Data["model"])
		}

	case *claude.SDKAssistantMessage:
		fmt.Println("\nAssistant response:")
		for _, block := range m.Message.Content {
			switch b := block.(type) {
			case claude.TextBlock:
				fmt.Printf("  %s\n", b.Text)
			case claude.TextContentBlock:
				fmt.Printf("  %s\n", b.Text)
			}
		}

	case *claude.SDKResultMessage:
		fmt.Printf("\nResult: %s\n", m.Subtype)
		fmt.Printf("Duration: %dms\n", m.DurationMS)
		fmt.Printf("Cost: $%.4f\n", m.TotalCostUSD)
		fmt.Printf("Turns: %d\n", m.NumTurns)
	}
}
```

## Features

- **Type-safe**: Fully typed Go interfaces matching the TypeScript SDK
- **Streaming**: Real-time message streaming support
- **MCP Support**: Integration with MCP servers
- **Custom Tools**: Define your own SDK MCP tools
- **Permission Control**: Fine-grained permission management
- **Context Support**: Full context.Context integration
- **Idiomatic Go**: Follows Go best practices and conventions

## Configuration Options

```go
opts := &claude.Options{
	// Model configuration
	Model:             "claude-sonnet-4-5",
	FallbackModel:     "claude-3-5-haiku-20241022",
	MaxTurns:          10,
	MaxThinkingTokens: 8000,

	// Directory and tool configuration
	Cwd:                   "/path/to/working/dir",
	AdditionalDirectories: []string{"/path/to/extra/dir"},
	AllowedTools:          []string{"Read", "Write", "Bash"},
	DisallowedTools:       []string{"WebSearch"},

	// Permission handling
	PermissionMode:           claudeagent.PermissionModeDefault,
	PermissionPromptToolName: "PermissionPrompt",

	// Message handling
	IncludePartialMessages: true,

	// Environment variables
	Env: map[string]string{
		"MAX_THINKING_TOKENS": "8000",
	},
}
```

## Examples

See the [examples/](./examples/) directory for complete examples:

- [examples/basic/](./examples/basic/) - **Working**: Basic query execution with the claude binary

## Architecture

## Testing

```bash
# Run unit tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests
go test -tags=integration ./test/integration/...

# Run e2e tests (requires ANTHROPIC_API_KEY)
go test -tags=e2e ./test/e2e/...
```

## Development Status

The SDK is **fully functional and tested** with the `claude` CLI binary.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT

## Related Projects

- [Claude Code TypeScript SDK](https://github.com/anthropics/anthropic-sdk-typescript)
- [Claude Code Python SDK](https://github.com/anthropics/anthropic-sdk-python)

## Support

For issues and questions:
- Open an issue on GitHub
- Check the [examples/](./examples/) directory
- Refer to the Claude Code documentation
