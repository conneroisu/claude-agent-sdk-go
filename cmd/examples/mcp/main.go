package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/conneroisu/claude/pkg/claude"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
)

func main() {
	ctx := context.Background()

	// Create an MCP server with custom tools
	weatherTool := claude.Tool(
		"get_weather",
		"Get the current weather for a location",
		map[string]any{
			"location": "string",
		},
		func(ctx context.Context, args map[string]any) (map[string]any, error) {
			location, _ := args["location"].(string)
			// Simulate weather API call
			return map[string]any{
				"content": []any{
					map[string]any{
						"type": "text",
						"text": fmt.Sprintf("The weather in %s is sunny, 72°F", location),
					},
				},
			}, nil
		},
	)

	timeTool := claude.Tool(
		"get_time",
		"Get the current time",
		map[string]any{},
		func(ctx context.Context, args map[string]any) (map[string]any, error) {
			now := time.Now().Format("3:04 PM MST")

			return map[string]any{
				"content": []any{
					map[string]any{
						"type": "text",
						"text": fmt.Sprintf("The current time is %s", now),
					},
				},
			}, nil
		},
	)

	// Create SDK MCP server
	mcpServer, err := claude.CreateSDKMCPServer("my-tools", "1.0.0", []*claude.SDKMCPTool{
		weatherTool,
		timeTool,
	})
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	// Configure options
	opts := &options.AgentOptions{
		Model: strPtr("claude-sonnet-4-5-20250929"),
		MCPServers: map[string]options.MCPServerConfig{
			"my-tools": options.SDKServerConfig{
				Type: "sdk",
				Name: "my-tools",
			},
		},
	}

	// Create client and register MCP server
	client := claude.NewClient(opts, nil, nil)
	client.RegisterMCPServer("my-tools", mcpServer)

	// Connect with prompt
	if err := client.Connect(ctx, "What's the weather in San Francisco and what time is it?"); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer func() {
		_ = client.Close()
	}()

	// Receive messages
	msgCh, errCh := client.ReceiveMessages(ctx)

	// Process messages
	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				return
			}
			handleMessage(msg)

		case err, ok := <-errCh:
			if !ok {
				return
			}
			if err != nil {
				log.Printf("Error: %v", err)

				return
			}
		}
	}
}

func handleMessage(msg messages.Message) {
	switch m := msg.(type) {
	case *messages.AssistantMessage:
		fmt.Println("\nAssistant:")
		for _, block := range m.Content {
			switch b := block.(type) {
			case messages.TextBlock:
				fmt.Println(b.Text)
			case messages.ToolUseBlock:
				fmt.Printf("[Calling tool: %s with args %v]\n", b.Name, b.Input)
			case messages.ToolResultBlock:
				fmt.Printf("[Tool result: %v]\n", b.Content)
			}
		}

	case *messages.ResultMessage:
		fmt.Printf("\nCompleted in %dms\n", m.DurationMs)
	}
}

func strPtr(s string) *string {
	return &s
}
