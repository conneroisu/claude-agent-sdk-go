//nolint:revive // Example file with acceptable complexity
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

	// Define hooks
	hooks := createHooks()

	// Configure options
	maxTurns := 1
	opts := &options.AgentOptions{
		MaxTurns: &maxTurns,
	}

	// Execute query with hooks
	msgCh, errCh := claude.Query(
		ctx,
		"List files in the current directory",
		opts,
		hooks,
	)

	// Process responses
	if err := processResponses(msgCh, errCh); err != nil {
		log.Fatal(err)
	}
}

// createHooks builds the hook configuration.
func createHooks() map[claude.HookEvent][]claude.HookMatcher {
	return map[claude.HookEvent][]claude.HookMatcher{
		claude.HookEventPreToolUse: {
			{
				Matcher: "Bash",
				Hooks:   []claude.HookCallback{preToolHook},
			},
		},
		claude.HookEventPostToolUse: {
			{
				Matcher: "Bash",
				Hooks:   []claude.HookCallback{postToolHook},
			},
		},
	}
}

// preToolHook executes before tool use.
func preToolHook(
	input map[string]any,
	_ *string,
	_ claude.HookContext,
) (map[string]any, error) {
	fmt.Println("=== Pre-Tool Hook ===")
	fmt.Println("Tool: Bash")
	fmt.Printf("Input: %v\n", input)
	fmt.Println("====================")

	return make(map[string]any), nil
}

// postToolHook executes after tool use.
func postToolHook(
	input map[string]any,
	_ *string,
	_ claude.HookContext,
) (map[string]any, error) {
	fmt.Println("=== Post-Tool Hook ===")
	fmt.Println("Tool: Bash")
	fmt.Printf("Result: %v\n", input)
	fmt.Println("======================")

	return make(map[string]any), nil
}

// processResponses handles incoming messages and errors.
func processResponses(
	msgCh <-chan messages.Message,
	errCh <-chan error,
) error {
	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				return nil
			}

			handleMessage(msg)

		case err := <-errCh:
			return err
		}
	}
}

// handleMessage processes a single message.
func handleMessage(msg messages.Message) {
	assistantMsg, ok := msg.(*messages.AssistantMessage)
	if !ok {
		return
	}

	printTextBlocks(assistantMsg.Content)
}

// printTextBlocks prints all text blocks from content.
func printTextBlocks(content []messages.ContentBlock) {
	for _, block := range content {
		textBlock, ok := block.(messages.TextBlock)
		if !ok {
			continue
		}

		fmt.Printf("Claude: %s\n", textBlock.Text)
	}
}
