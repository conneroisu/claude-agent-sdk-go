// Package main demonstrates streaming message handling with the
// Claude Agent SDK.
//
// This example shows how to:
// - Configure and connect to the Claude Agent API
// - Send messages to the agent
// - Receive and process streaming responses in real-time
// - Handle different message types (text, thinking, tool use)
// - Display usage metrics and costs
//
// The streaming approach allows you to see the agent's responses as
// they are generated, providing a more interactive experience compared
// to waiting for complete responses.
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/conneroisu/claude/pkg/claude"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
)

func main() {
	// Configure the agent with the latest Claude model.
	// AgentOptions allows customization of model, temperature, max tokens, etc.
	client := createClient()

	ctx := context.Background()

	// Get the initial prompt from the user to start the conversation.
	scanner, initialPrompt := getInitialPrompt()
	if initialPrompt == nil {
		return
	}

	// Establish a connection to the Claude Agent API with the initial prompt.
	// This initializes the conversation session.
	if err := client.Connect(ctx, initialPrompt); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	// Ensure the connection is properly closed when the program exits.
	defer closeClient(client)

	// Start receiving messages asynchronously.
	// Messages are processed as they arrive, enabling real-time streaming.
	startMessageReceiver(ctx, client)

	// Enter the main conversation loop where user input is sent to the agent.
	handleUserInput(ctx, client, scanner)
}

// createClient initializes and returns a new Claude client with
// default options. The client is configured to use the Claude
// Sonnet 4.5 model.
func createClient() *claude.Client {
	opts := &options.AgentOptions{
		Model: stringPtr("claude-sonnet-4-5-20250929"),
	}

	return claude.NewClient(opts, nil, nil)
}

// getInitialPrompt prompts the user for initial input and returns a
// scanner for continued input. It returns nil if the user wants to
// exit or if input cannot be read.
func getInitialPrompt() (*bufio.Scanner, *string) {
	fmt.Print("Enter your prompt (or 'exit' to quit): ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return nil, nil
	}
	initialPrompt := scanner.Text()

	if strings.TrimSpace(initialPrompt) == "exit" {
		return nil, nil
	}

	return scanner, &initialPrompt
}

// closeClient safely closes the client connection and logs any errors.
func closeClient(client *claude.Client) {
	if err := client.Close(); err != nil {
		log.Printf("Error closing client: %v", err)
	}
}

// startMessageReceiver begins processing incoming messages in a
// separate goroutine. It handles both successful messages and errors
// from the message channels.
func startMessageReceiver(ctx context.Context, client *claude.Client) {
	msgCh, errCh := client.ReceiveMessages(ctx)
	go processMessages(msgCh, errCh)
}

// processMessages handles the message and error channels in a loop.
// It displays messages and errors as they arrive until channels close.
func processMessages(msgCh <-chan messages.Message, errCh <-chan error) {
	for {
		if !processNextMessage(msgCh, errCh) {
			return
		}
	}
}

// processNextMessage waits for and processes the next message or error.
// Returns false if both channels are closed.
func processNextMessage(
	msgCh <-chan messages.Message,
	errCh <-chan error,
) bool {
	select {
	case msg, ok := <-msgCh:
		if !ok {
			return false
		}
		handleMessage(msg)

		return true
	case err, ok := <-errCh:
		if !ok {
			return false
		}
		displayError(err)

		return true
	}
}

// displayError prints an error message if the error is not nil.
func displayError(err error) {
	if err != nil {
		fmt.Printf("\nError: %v\n", err)
	}
}

// handleUserInput processes user input in a loop, sending messages
// to the agent. The loop continues until the user types 'exit'.
func handleUserInput(
	ctx context.Context,
	client *claude.Client,
	scanner *bufio.Scanner,
) {
	fmt.Println("\nEnter messages (or 'exit' to quit):")
	for scanner.Scan() {
		input := scanner.Text()
		if strings.TrimSpace(input) == "exit" {
			break
		}

		if err := client.SendMessage(ctx, input); err != nil {
			log.Printf("Failed to send message: %v", err)
		}
	}
}

// handleMessage processes and displays different types of messages
// from the agent. It routes each message type to the appropriate
// handler function.
func handleMessage(msg messages.Message) {
	switch m := msg.(type) {
	case *messages.AssistantMessage:
		handleAssistantMessage(m)
	case *messages.ResultMessageSuccess:
		handleSuccessResult(m)
	case *messages.ResultMessageError:
		handleErrorResult(m)
	case *messages.SystemMessage:
		handleSystemMessage(m)
	}
}

// handleAssistantMessage processes and displays content from
// assistant messages. Assistant messages can contain text, thinking
// processes, or tool usage.
func handleAssistantMessage(m *messages.AssistantMessage) {
	for _, block := range m.Content {
		switch b := block.(type) {
		case messages.TextBlock:
			fmt.Printf("\nAssistant: %s\n", b.Text)
		case messages.ThinkingBlock:
			fmt.Printf("\n[Thinking]: %s\n", b.Thinking)
		case messages.ToolUseBlock:
			fmt.Printf("\n[Tool Use]: %s(%v)\n", b.Name, b.Input)
		}
	}
}

// handleSuccessResult displays metrics for a successfully completed turn.
// This includes token usage and cost information.
func handleSuccessResult(m *messages.ResultMessageSuccess) {
	fmt.Printf("\n✓ Turn complete (Cost: $%.4f, Tokens: %d in / %d out)\n",
		m.TotalCostUSD, m.Usage.InputTokens, m.Usage.OutputTokens)
}

// handleErrorResult displays error information from failed operations.
func handleErrorResult(m *messages.ResultMessageError) {
	fmt.Printf("\n✗ Error: %s\n", m.Subtype)
}

// handleSystemMessage processes system-level messages such as
// session initialization.
func handleSystemMessage(m *messages.SystemMessage) {
	if m.Subtype == "init" {
		fmt.Println("✓ Session initialized")
	}
}

// stringPtr is a helper function that returns a pointer to a string.
// This is useful for optional fields that require pointer types.
func stringPtr(s string) *string {
	return &s
}
