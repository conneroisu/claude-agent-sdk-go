// Package main demonstrates interactive chat with Claude Agent SDK.
//
// This example shows how to create an interactive chat session where users
// can have a continuous conversation with Claude. It demonstrates proper
// client lifecycle management, message handling, and graceful shutdown.
//
// The example uses a scanner for reading user input and processes both
// text responses and tool usage from Claude.
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/conneroisu/claude-agent-sdk-go/pkg/claude"
)

const (
	// maxConversationTurns is the maximum number of turns allowed in
	// interactive conversation.
	maxConversationTurns = 50
)

// main is the entry point for the interactive chat example.
// It sets up the client, runs the chat loop, and ensures proper cleanup.
func main() {
	ctx := context.Background()

	printWelcome()

	client, err := createClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer closeClient(client)

	runInteractiveLoop(ctx, client)
}

// printWelcome displays the welcome message and instructions.
func printWelcome() {
	fmt.Println("Claude Interactive Chat")
	fmt.Println("========================")
	fmt.Println("Type your messages and press Enter.")
	fmt.Println("Type 'exit' or 'quit' to end the session.")
	fmt.Println()
}

// createClient creates and returns a new Claude SDK client.
func createClient() (*claude.ClaudeSDKClient, error) {
	opts := &claude.Options{
		Model:    "claude-sonnet-4-5",
		MaxTurns: maxConversationTurns,
	}

	return claude.NewClient(opts)
}

// closeClient safely closes the Claude SDK client.
func closeClient(client *claude.ClaudeSDKClient) {
	if closeErr := client.Close(); closeErr != nil {
		log.Printf("Failed to close client: %v", closeErr)
	}
}

// runInteractiveLoop runs the main interactive chat loop.
// It continuously reads user input, processes commands, and displays
// responses from Claude until the user exits.
func runInteractiveLoop(
	ctx context.Context,
	client *claude.ClaudeSDKClient,
) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\n> ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		if shouldExit(input) {
			fmt.Println("Goodbye!")

			break
		}

		if input == "" {
			continue
		}

		err := processInput(ctx, client, input)
		if err != nil {
			log.Printf("Error: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading input: %v", err)
	}
}

// shouldExit checks if the input is an exit command.
func shouldExit(input string) bool {
	return input == "exit" || input == "quit"
}

// processInput sends input to Claude and displays the response.
func processInput(
	ctx context.Context,
	client *claude.ClaudeSDKClient,
	input string,
) error {
	err := client.Query(ctx, input)
	if err != nil {
		return fmt.Errorf("sending query: %w", err)
	}

	err = receiveAndDisplayResponse(ctx, client)
	if err != nil {
		return fmt.Errorf("receiving response: %w", err)
	}

	return nil
}

// receiveAndDisplayResponse receives messages and displays them.
func receiveAndDisplayResponse(
	ctx context.Context,
	client *claude.ClaudeSDKClient,
) error {
	fmt.Print("\nClaude: ")

	var hasContent bool

	for msg := range client.ReceiveResponse(ctx) {
		hasContent = handleMessage(msg, hasContent)
	}

	if !hasContent {
		fmt.Println("[no response]")
	}

	return nil
}

// handleMessage routes messages to handlers and tracks content display.
func handleMessage(msg claude.SDKMessage, had bool) bool {
	switch m := msg.(type) {
	case *claude.SDKAssistantMessage:
		shown := had
		for _, block := range m.Message.Content {
			shown = displayBlock(block, shown)
		}

		return shown

	case *claude.SDKStreamEvent:
		if evt, ok := m.Event.(claude.ContentBlockDeltaEvent); ok {
			if evt.Delta.TextDelta != nil {
				fmt.Print(*evt.Delta.TextDelta)

				return true
			}
		}

		return had

	case *claude.SDKResultMessage:
		if m.IsError {
			fmt.Printf("\n[Error: %s]", m.Subtype)
		}

		return had
	}

	return had
}

// displayBlock displays content blocks and returns if content was shown.
func displayBlock(block claude.ContentBlock, had bool) bool {
	switch b := block.(type) {
	case claude.TextBlock:
		if b.Text != "" {
			fmt.Print(b.Text)

			return true
		}
	case claude.TextContentBlock:
		if b.Text != "" {
			fmt.Print(b.Text)

			return true
		}
	case claude.ThinkingBlock:
		return had
	case claude.ToolUseContentBlock:
		fmt.Printf("\n[Using tool: %s]", b.Name)

		return true
	}

	return had
}
