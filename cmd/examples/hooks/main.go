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

	// Configure options
	opts := &options.AgentOptions{
		Model: strPtr("claude-sonnet-4-5-20250929"),
	}

	// Set up hooks to block certain bash commands
	hooks := map[claude.HookEvent][]claude.HookMatcher{
		claude.HookEventPreToolUse: {
			{
				Matcher: "Bash", // Match all Bash tool uses
				Hooks: []claude.HookCallback{
					claude.BlockBashPatternHook([]string{"rm -rf", "sudo"}),
					customPreToolHook,
				},
			},
		},
		claude.HookEventPostToolUse: {
			{
				Matcher: "*", // Match all tools
				Hooks: []claude.HookCallback{
					postToolHook,
				},
			},
		},
	}

	// Perform query with hooks
	msgCh, errCh := claude.Query(
		ctx,
		"Can you list the files in the current directory?",
		opts,
		hooks,
		nil, // no custom permissions
	)

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
				log.Fatalf("Error: %v", err)
			}
		}
	}
}

func customPreToolHook(input map[string]any, toolUseID *string, ctx claude.HookContext) (map[string]any, error) {
	toolName, _ := input["tool_name"].(string)
	fmt.Printf("[Hook] Pre-tool use: %s\n", toolName)

	// You could inspect the input and potentially modify it or block execution
	return map[string]any{}, nil
}

func postToolHook(input map[string]any, toolUseID *string, ctx claude.HookContext) (map[string]any, error) {
	toolName, _ := input["tool_name"].(string)
	fmt.Printf("[Hook] Post-tool use: %s\n", toolName)

	return map[string]any{}, nil
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
				fmt.Printf("[Using tool: %s]\n", b.Name)
			case messages.ToolResultBlock:
				fmt.Printf("[Tool result for %s]\n", b.ToolUseID)
			}
		}

	case *messages.ResultMessage:
		fmt.Printf("\nCompleted in %dms\n", m.DurationMs)
	}
}

func strPtr(s string) *string {
	return &s
}
