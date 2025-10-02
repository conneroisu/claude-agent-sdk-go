// Package main demonstrates the quickstart example for the Claude Agent SDK.
//
// This example shows how to:
//   - Create a simple Claude agent query
//   - Configure agent options (model selection)
//   - Process streaming messages from the agent
//   - Handle different message types (assistant, results, errors)
//   - Display agent responses and usage statistics
//
// Usage:
//
//	quickstart "What is the weather like today?"
//
// The agent will process your prompt and stream back responses, including:
//   - Assistant text responses
//   - Thinking process (if enabled)
//   - Tool usage information
//   - Final results with cost and token usage
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/conneroisu/claude/pkg/claude"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
)

// main is the entry point for the quickstart example.
// It validates arguments, configures the agent, and processes messages.
func main() {
	// Validate command-line arguments
	if err := validateArgs(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	prompt := os.Args[1]

	// Create agent options with Claude Sonnet 4.5 model
	opts := createAgentOptions()

	// Execute the agent query and process messages
	if err := processQuery(prompt, opts); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

// validateArgs checks if the required prompt argument is provided.
func validateArgs() error {
	if len(os.Args) < 2 {
		return errors.New("usage: quickstart <prompt>")
	}

	return nil
}

// createAgentOptions initializes the agent configuration.
// Returns options configured with the Claude Sonnet 4.5 model.
func createAgentOptions() *options.AgentOptions {
	return &options.AgentOptions{
		Model: stringPtr("claude-sonnet-4-5-20250929"),
	}
}

// processQuery executes the Claude query and handles the message stream.
// It listens to both message and error channels until completion.
// Returns an error if the query encounters an error.
func processQuery(
	prompt string,
	opts *options.AgentOptions,
) error {
	ctx := context.Background()
	msgCh, errCh := claude.Query(ctx, prompt, opts, nil)

	return listenToChannels(msgCh, errCh)
}

// listenToChannels processes messages from both channels until done.
// Returns any error encountered during processing.
func listenToChannels(
	msgCh <-chan messages.Message,
	errCh <-chan error,
) error {
	for {
		if err := processNextEvent(msgCh, errCh); err != nil {
			return handleEventError(err)
		}
	}
}

// processNextEvent processes the next event from either channel.
// Returns channelClosed sentinel when done, or actual errors.
func processNextEvent(
	msgCh <-chan messages.Message,
	errCh <-chan error,
) error {
	select {
	case msg, ok := <-msgCh:
		if !ok {
			return errChannelClosed
		}
		handleMessage(msg)

		return nil

	case err, ok := <-errCh:
		if !ok {
			return errChannelClosed
		}

		return err
	}
}

var errChannelClosed = errors.New("channel closed")

// handleEventError processes errors from event processing.
// Returns nil if channel closed (normal), otherwise returns error.
func handleEventError(err error) error {
	if errors.Is(err, errChannelClosed) {
		return nil
	}

	return err
}

// handleMessage processes different types of messages from the agent.
// Dispatches to specialized handlers based on message type.
func handleMessage(msg messages.Message) {
	if handled := handleResultMessages(msg); handled {
		return
	}
	handleOtherMessages(msg)
}

// handleResultMessages processes result-type messages.
// Returns true if the message was a result message.
func handleResultMessages(msg messages.Message) bool {
	switch m := msg.(type) {
	case *messages.ResultMessageSuccess:
		handleSuccessResult(m)

		return true
	case *messages.ResultMessageError:
		handleErrorResult(m)

		return true
	}

	return false
}

// handleOtherMessages processes non-result message types.
// Handles assistant, stream, and system messages.
func handleOtherMessages(msg messages.Message) {
	switch m := msg.(type) {
	case *messages.AssistantMessage:
		handleAssistantMessage(m)
	case *messages.StreamEvent:
		handleStreamEvent(m)
	case *messages.SystemMessage:
		handleSystemMessage(m)
	}
}

// handleStreamEvent processes stream events.
// Currently silent, but can be enabled for debugging.
func handleStreamEvent(_ *messages.StreamEvent) {
	// Stream events are handled silently in this example
	// Uncomment to see stream event UUIDs:
	// fmt.Printf("[Stream Event]: %s\n", m.UUID)
}

// handleAssistantMessage processes content blocks from assistant messages.
// Displays text responses, thinking processes, and tool usage.
func handleAssistantMessage(m *messages.AssistantMessage) {
	for _, block := range m.Content {
		handleContentBlock(block)
	}
}

// handleContentBlock processes individual content blocks.
// Supports text, thinking, and tool use blocks.
func handleContentBlock(block messages.ContentBlock) {
	switch b := block.(type) {
	case messages.TextBlock:
		fmt.Printf("Assistant: %s\n", b.Text)
	case messages.ThinkingBlock:
		fmt.Printf("[Thinking]: %s\n", b.Thinking)
	case messages.ToolUseBlock:
		fmt.Printf("[Tool Use]: %s\n", b.Name)
	}
}

// handleSuccessResult displays the final success result.
// Shows duration, turn count, cost, and token usage.
func handleSuccessResult(m *messages.ResultMessageSuccess) {
	fmt.Printf(
		"\n✓ Success in %dms (%d turns)\n",
		m.DurationMs,
		m.NumTurns,
	)
	fmt.Printf("  Cost: $%.4f\n", m.TotalCostUSD)
	fmt.Printf(
		"  Tokens: %d in / %d out\n",
		m.Usage.InputTokens,
		m.Usage.OutputTokens,
	)
}

// handleErrorResult displays error result information.
// Shows the error type, duration, and turn count.
func handleErrorResult(m *messages.ResultMessageError) {
	fmt.Printf("\n✗ Error: %s\n", m.Subtype)
	fmt.Printf(
		"  Duration: %dms (%d turns)\n",
		m.DurationMs,
		m.NumTurns,
	)
}

// handleSystemMessage processes system-level messages.
// Currently only handles initialization messages.
func handleSystemMessage(m *messages.SystemMessage) {
	if m.Subtype == "init" {
		fmt.Println("Session initialized")
	}
}

// stringPtr is a helper to create a string pointer.
// Useful for optional string fields in configuration.
func stringPtr(s string) *string {
	return &s
}
