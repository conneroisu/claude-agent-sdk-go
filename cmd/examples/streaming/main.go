package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

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

	// Create a new client
	client := claude.NewClient(opts, nil, nil)

	// Connect with initial prompt
	if err := client.Connect(ctx, "Hello! I'd like to have a conversation."); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer func() {
		_ = client.Close()
	}()

	// Start receiving messages in a goroutine
	msgCh, errCh := client.ReceiveMessages(ctx)
	go func() {
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
	}()

	// Interactive conversation loop
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("\nYou can now chat with Claude. Type 'exit' to quit.")

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		if input == "exit" {

			break
		}

		if err := client.SendMessage(ctx, input); err != nil {
			log.Printf("Failed to send message: %v", err)

			break
		}
	}
}

func handleMessage(msg messages.Message) {
	switch m := msg.(type) {
	case *messages.AssistantMessage:
		fmt.Print("\nClaude: ")
		for _, block := range m.Content {
			switch b := block.(type) {
			case messages.TextBlock:
				fmt.Print(b.Text)
			case messages.ThinkingBlock:
				fmt.Printf("[Thinking...] ")
			case messages.ToolUseBlock:
				fmt.Printf("\n[Using tool: %s]\n", b.Name)
			}
		}
		fmt.Println()

	case *messages.ResultMessage:
		if m.IsError {
			fmt.Printf("\n[Error in conversation]\n")
		}

	case *messages.StreamEvent:
		// Handle streaming events if needed
	}
}

func strPtr(s string) *string {
	return &s
}
