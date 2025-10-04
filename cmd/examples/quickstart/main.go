// Package main demonstrates basic Query() usage with the Claude Agent SDK.
//
// This example shows:
//   - Simple one-shot query to Claude
//   - Basic response handling
//   - Error handling patterns
//
// Prerequisites: Claude CLI must be installed and configured
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/conneroisu/claude/pkg/claude"
	"github.com/conneroisu/claude/pkg/claude/messages"
)

func main() {
	// Create a context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	if err := run(ctx); err != nil {
		cancel()
		log.Fatalf("Error: %v", err)
	}

	cancel()
}

func run(ctx context.Context) error {
	// Prepare a simple prompt
	prompt := "What are the three Great Lakes that border Michigan? Please be concise."

	// Execute the query
	// Query returns immediately with channels that populate as the query executes
	msgCh, errCh := claude.Query(
		ctx,
		prompt,
		nil, // Use default agent options
		nil, // No hooks needed for this simple example
	)

	fmt.Println("Sending query to Claude...")
	fmt.Printf("Prompt: %s\n\n", prompt)

	// Process messages as they arrive
	// The message channel closes when Claude finishes responding
	for msg := range msgCh {
		// Type assert to get the specific message type
		switch m := msg.(type) {
		case messages.AssistantMessage:
			// Assistant messages contain Claude's actual response
			fmt.Println("Claude's response:")
			for _, block := range m.Content {
				// Content blocks can be text, thinking, tool uses, etc.
				// For this simple example, we just print the text content
				fmt.Printf("%+v\n", block)
			}
		case messages.SystemMessage:
			// System messages contain metadata and notifications
			fmt.Printf("System: %+v\n", m)
		default:
			// Other message types (UserMessage, ResultMessage, StreamEvent)
			fmt.Printf("Message: %+v\n", m)
		}
	}

	// Check for errors after all messages are processed
	// The error channel always receives exactly one value (nil or an error)
	if err := <-errCh; err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	fmt.Println("\nQuery completed successfully!")

	return nil
}
