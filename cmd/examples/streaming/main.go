// Package main demonstrates streaming conversations with the Claude Agent SDK.
//
// This example shows:
//   - Creating a persistent Client connection
//   - Sending multiple messages in a conversation
//   - Receiving streaming responses
//   - Proper resource cleanup
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
	// Create a new client with default options
	client := claude.NewClient(
		nil, // Use default agent options
		nil, // No hooks for this example
		nil, // Use default permissions
	)

	// Always clean up resources when done
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("Error closing client: %v", err)
		}
	}()

	// Create a context with timeout for the entire conversation
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

	if err := run(ctx, client); err != nil {
		cancel()
		log.Printf("Error: %v", err)

		return
	}

	cancel()
}

func run(ctx context.Context, client *claude.Client) error {
	// Connect to Claude with an initial greeting
	initialPrompt := "Hello! I'd like to have a brief conversation about Go programming."
	fmt.Printf("Connecting to Claude with: %s\n\n", initialPrompt)

	if err := client.Connect(ctx, &initialPrompt); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Start receiving messages in a goroutine
	msgCh, errCh := client.ReceiveMessages(ctx)
	go handleMessages(msgCh)

	// Wait a moment for the initial response
	time.Sleep(2 * time.Second)

	// Send a follow-up message
	followUp := "What are some key features of Go's concurrency model?"
	fmt.Printf("\nSending follow-up: %s\n\n", followUp)

	if err := client.SendMessage(ctx, followUp); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Wait a moment for the response
	time.Sleep(2 * time.Second)

	// Send another message to continue the conversation
	finalQuestion := "Can you give me a simple example of a goroutine?"
	fmt.Printf("\nSending final question: %s\n\n", finalQuestion)

	if err := client.SendMessage(ctx, finalQuestion); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Wait for responses to complete
	time.Sleep(3 * time.Second)

	// Check for any errors
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("streaming error: %w", err)
		}
	default:
		// No error
	}

	fmt.Println("\nConversation completed successfully!")

	return nil
}

// handleMessages processes incoming messages from Claude.
func handleMessages(msgCh <-chan messages.Message) {
	for msg := range msgCh {
		switch m := msg.(type) {
		case messages.AssistantMessage:
			// Print Claude's response content
			fmt.Println("Claude says:")
			for _, block := range m.Content {
				fmt.Printf("  %+v\n", block)
			}
		case messages.SystemMessage:
			// System messages contain metadata
			fmt.Printf("[System: %+v]\n", m)
		case messages.StreamEvent:
			// Stream events show real-time API activity
			eventType, _ := m.Event["type"].(string)
			fmt.Printf("[Stream: %s]\n", eventType)
		default:
			fmt.Printf("[Message: %T]\n", m)
		}
	}
}
