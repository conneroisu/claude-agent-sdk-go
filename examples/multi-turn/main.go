// Package main demonstrates multi-turn conversations with Claude.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
)

const (
	// maxConversationTurns defines the maximum number of conversation turns.
	maxConversationTurns = 10
)

// Multi-Turn Conversation Example
//
// This example demonstrates how to conduct a multi-turn conversation
// with Claude using the Query() method to send follow-up messages and
// ReceiveResponse() for cleaner single-response workflows.
//
// Key features demonstrated:
// - Initial query with Query()
// - Follow-up queries using Query() to maintain context
// - Using ReceiveResponse() for simpler message handling
// - Context preservation across multiple turns
//
// The example simulates a math tutor conversation where Claude helps
// solve a problem step by step.

func main() {
	ctx := context.Background()

	opts := &claude.Options{
		Model:    "claude-sonnet-4-5",
		MaxTurns: maxConversationTurns,
	}

	client, err := claude.NewClient(opts)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer closeClient(client)

	fmt.Println("=== Multi-Turn Conversation Example ===")
	fmt.Println("Demonstrating a conversation about solving a math problem")

	// Turn 1: Initial question
	runTurn1(ctx, client)

	// Turn 2: Follow-up question
	runTurn2(ctx, client)

	// Turn 3: Final verification
	finalResult := runTurn3(ctx, client)

	// Print conversation summary
	printSummary(finalResult)

	fmt.Println("\n✓ Multi-turn conversation completed successfully!")
}

// closeClient safely closes the client connection.
func closeClient(client *claude.ClaudeSDKClient) {
	if closeErr := client.Close(); closeErr != nil {
		log.Printf("Failed to close client: %v", closeErr)
	}
}

// runTurn1 executes the first conversation turn with the initial question.
func runTurn1(ctx context.Context, client *claude.ClaudeSDKClient) {
	fmt.Println("Turn 1: Initial Question")
	fmt.Println("User: Can you help me solve this equation: 2x + 5 = 13?")

	query := "Can you help me solve this equation: 2x + 5 = 13?" +
		" Just give me the first step."
	err := client.Query(ctx, query)
	if err != nil {
		log.Printf("Failed to send initial query: %v", err)

		return
	}

	processResponse(ctx, client)
}

// runTurn2 executes the second conversation turn with a follow-up question.
func runTurn2(ctx context.Context, client *claude.ClaudeSDKClient) {
	fmt.Println("\nTurn 2: Follow-up Question")
	fmt.Println("User: Great! What's the next step?")

	err := client.Query(ctx, "Great! What's the next step?")
	if err != nil {
		log.Printf("Failed to send follow-up query: %v", err)

		return
	}

	processResponse(ctx, client)
}

// runTurn3 executes the third turn and returns the final result.
func runTurn3(
	ctx context.Context,
	client *claude.ClaudeSDKClient,
) *claude.SDKResultMessage {
	fmt.Println("\nTurn 3: Verification")
	fmt.Println(
		"User: Can you verify the solution by substituting x back" +
			" into the original equation?",
	)

	query := "Can you verify the solution by substituting x back" +
		" into the original equation?"
	err := client.Query(ctx, query)
	if err != nil {
		log.Printf("Failed to send verification query: %v", err)

		return nil
	}

	var finalResult *claude.SDKResultMessage
	for msg := range client.ReceiveResponse(ctx) {
		switch m := msg.(type) {
		case *claude.SDKAssistantMessage:
			fmt.Print("Claude: ")
			printAssistantMessage(m)
			fmt.Println()
		case *claude.SDKResultMessage:
			finalResult = m
		}
	}

	return finalResult
}

// processResponse processes and displays assistant responses.
func processResponse(
	ctx context.Context,
	client *claude.ClaudeSDKClient,
) {
	for msg := range client.ReceiveResponse(ctx) {
		assistant, ok := msg.(*claude.SDKAssistantMessage)
		if !ok {
			continue
		}

		fmt.Print("Claude: ")
		printAssistantMessage(assistant)
		fmt.Println()
	}
}

// printSummary prints the conversation summary statistics.
func printSummary(result *claude.SDKResultMessage) {
	if result == nil {
		return
	}

	fmt.Println("\n=== Conversation Summary ===")
	fmt.Printf("Total turns: %d\n", result.NumTurns)
	fmt.Printf("Duration: %dms\n", result.DurationMS)
	fmt.Printf("Total cost: $%.4f\n", result.TotalCostUSD)
	fmt.Printf("Input tokens: %d\n", result.Usage.InputTokens)
	fmt.Printf("Output tokens: %d\n", result.Usage.OutputTokens)

	if result.Usage.CacheReadInputTokens > 0 {
		fmt.Printf(
			"Cache hit tokens: %d (saved API calls!)\n",
			result.Usage.CacheReadInputTokens,
		)
	}
}

// printAssistantMessage extracts and prints text from assistant messages.
func printAssistantMessage(msg *claude.SDKAssistantMessage) {
	for _, block := range msg.Message.Content {
		switch b := block.(type) {
		case claude.TextBlock:
			fmt.Print(b.Text)
		case claude.TextContentBlock:
			fmt.Print(b.Text)
		case claude.ThinkingBlock:
			// Optionally show thinking (usually hidden in production)
			// fmt.Printf("[Thinking: %s]\n", b.Thinking)
		}
	}
}
