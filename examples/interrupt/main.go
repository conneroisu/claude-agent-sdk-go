// Package main demonstrates interrupt handling in Claude Agent SDK.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
)

const (
	maxInterruptTurns     = 20
	interruptDelaySeconds = 3
	maxTextDisplayLength  = 100
	interruptTimeout      = 5 * time.Second
	resultWaitTimeout     = 2 * time.Second
)

// Interrupt Handling Example
//
// This example demonstrates how to gracefully interrupt a long-running query
// using the Interrupt() control method.
//
// Key features demonstrated:
// - Starting a potentially long-running query
// - Monitoring query progress in a goroutine
// - Interrupting execution with Interrupt()
// - Handling interrupted responses gracefully
// - Using context with timeout for safety
//
// Interrupt use cases:
// - User cancellation
// - Timeout scenarios
// - Resource constraints
// - Error recovery
// - Cost control (stopping expensive operations)
//
// The example simulates a scenario where Claude is asked to perform
// multiple tasks, but we interrupt after a short period.

func main() {
	ctx := context.Background()

	printHeader()

	client, err := createClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer closeClient(client)

	if err := startQuery(ctx, client); err != nil {
		log.Printf("Failed to send query: %v", err)

		return
	}

	result, interrupted := runQueryWithInterrupt(ctx, client)

	printSummary(result, interrupted)
	printKeyTakeaways()
}

func printHeader() {
	fmt.Println("=== Interrupt Handling Example ===")
	fmt.Println("This example demonstrates interrupting a long-running query")
}

func createClient() (*claude.ClaudeSDKClient, error) {
	opts := &claude.Options{
		Model:    "claude-sonnet-4-5",
		MaxTurns: maxInterruptTurns,
	}

	return claude.NewClient(opts)
}

func closeClient(client *claude.ClaudeSDKClient) {
	if closeErr := client.Close(); closeErr != nil {
		log.Printf("Failed to close client: %v", closeErr)
	}
}

func startQuery(ctx context.Context, client *claude.ClaudeSDKClient) error {
	longQuery := `Please perform the following tasks:
1. List all files in the current directory
2. Calculate the factorial of 10
3. Explain what a binary search tree is
4. Write a simple function to reverse a string
5. Analyze the time complexity of bubble sort

Take your time and be thorough with each task.`

	fmt.Println("Starting long-running query...")
	fmt.Println("Query:", longQuery)
	fmt.Println()

	return client.Query(ctx, longQuery)
}

func runQueryWithInterrupt(
	ctx context.Context,
	client *claude.ClaudeSDKClient,
) (*claude.SDKResultMessage, bool) {
	interruptDone := make(chan bool)
	queryComplete := make(chan *claude.SDKResultMessage)

	go receiveMessages(ctx, client, queryComplete)
	go scheduleInterrupt(client, interruptDone)

	return waitForCompletion(queryComplete, interruptDone)
}

func receiveMessages(
	ctx context.Context,
	client *claude.ClaudeSDKClient,
	queryComplete chan *claude.SDKResultMessage,
) {
	for msg := range client.ReceiveResponse(ctx) {
		switch m := msg.(type) {
		case *claude.SDKSystemMessage:
			handleSystemMessage(m)
		case *claude.SDKAssistantMessage:
			handleAssistantMessage(m)
		case *claude.SDKResultMessage:
			queryComplete <- m

			return
		}
	}
	close(queryComplete)
}

func handleSystemMessage(m *claude.SDKSystemMessage) {
	if m.Subtype == "init" {
		fmt.Printf("✓ Session started with model: %v\n\n", m.Data["model"])
	}
}

func handleAssistantMessage(m *claude.SDKAssistantMessage) {
	fmt.Println("Claude is working...")
	for _, block := range m.Message.Content {
		displayContentBlock(block)
	}
}

func displayContentBlock(block claude.ContentBlock) {
	switch b := block.(type) {
	case claude.TextBlock:
		displayText(b.Text)
	case claude.TextContentBlock:
		displayText(b.Text)
	case claude.ToolUseContentBlock:
		fmt.Printf("  🔧 Using tool: %s\n", b.Name)
	}
}

func displayText(text string) {
	if len(text) > maxTextDisplayLength {
		text = text[:maxTextDisplayLength] + "..."
	}
	fmt.Printf("  %s\n", text)
}

func scheduleInterrupt(
	client *claude.ClaudeSDKClient,
	interruptDone chan bool,
) {
	interruptDelay := interruptDelaySeconds * time.Second
	fmt.Printf("⏱️  Will interrupt after %v...\n\n", interruptDelay)

	time.Sleep(interruptDelay)

	fmt.Println("\n⚠️  Time limit reached! Interrupting query...")

	interruptCtx, cancel := context.WithTimeout(context.Background(), interruptTimeout)
	defer cancel()

	if err := client.Interrupt(interruptCtx); err != nil {
		log.Printf("Error sending interrupt: %v", err)
	} else {
		fmt.Println("✓ Interrupt signal sent")
	}

	interruptDone <- true
}

func waitForCompletion(
	queryComplete chan *claude.SDKResultMessage,
	interruptDone chan bool,
) (*claude.SDKResultMessage, bool) {
	select {
	case result := <-queryComplete:
		fmt.Println("\n✓ Query completed naturally before interrupt")

		return result, false
	case <-interruptDone:
		return waitForInterruptedResult(queryComplete), true
	}
}

func waitForInterruptedResult(
	queryComplete chan *claude.SDKResultMessage,
) *claude.SDKResultMessage {
	select {
	case result := <-queryComplete:
		fmt.Println("✓ Received interrupted result")

		return result
	case <-time.After(resultWaitTimeout):
		fmt.Println("⚠️  Timed out waiting for result after interrupt")

		return nil
	}
}

func printSummary(result *claude.SDKResultMessage, interrupted bool) {
	fmt.Println("\n=== Execution Summary ===")
	if interrupted {
		fmt.Println("Status: ⚠️  INTERRUPTED")
	} else {
		fmt.Println("Status: ✓ COMPLETED")
	}

	if result != nil {
		printResultDetails(result)
	}
}

func printResultDetails(result *claude.SDKResultMessage) {
	fmt.Printf("Duration: %dms\n", result.DurationMS)
	fmt.Printf("Turns completed: %d\n", result.NumTurns)
	fmt.Printf("Cost: $%.4f\n", result.TotalCostUSD)
	fmt.Printf("Tokens used: %d input, %d output\n",
		result.Usage.InputTokens, result.Usage.OutputTokens)

	if result.IsError {
		fmt.Printf("Error occurred: %s\n", result.Subtype)
	}

	if result.Result != nil {
		fmt.Printf("Partial result: %s\n", *result.Result)
	}
}

func printKeyTakeaways() {
	fmt.Println("\n=== Key Takeaways ===")
	fmt.Println("1. Interrupt() provides graceful cancellation")
	fmt.Println("2. Useful for time limits, user cancellation, and cost control")
	fmt.Println("3. Partial results may be available after interruption")
	fmt.Println("4. Always handle interrupt errors appropriately")
	fmt.Println("5. Use context timeouts for additional safety")
	fmt.Println("\n✓ Interrupt handling example complete!")
}
