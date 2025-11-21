// Package main demonstrates basic usage of the Claude Agent SDK.
//
// This example shows how to:
//   - Create a Claude Agent client with basic configuration
//   - Send a simple query to the assistant
//   - Receive and process different types of messages
//   - Handle errors and properly close the client
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
)

const (
	// maxTurns is the maximum number of conversation turns allowed.
	maxTurns = 5
)

func main() {
	ctx := context.Background()

	// Create client with basic options
	opts := &claude.Options{
		Model:    "claude-sonnet-4-5",
		MaxTurns: maxTurns,
	}

	client, err := claude.NewClient(opts)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			log.Printf("Failed to close client: %v", closeErr)
		}
	}()

	// Send a simple query
	fmt.Println("Sending query: What is 2+2?")
	query := "What is 2+2? Just respond with the number."
	if err = client.Query(ctx, query); err != nil {
		log.Printf("Failed to send query: %v", err)

		return
	}

	// Receive and process responses
	msgChan, errChan := client.ReceiveMessages(ctx)

	for {
		select {
		case msg := <-msgChan:
			if msg == nil {
				fmt.Println("Query completed")

				return
			}

			handleMessage(msg)

		case err := <-errChan:
			if err != nil {
				log.Printf("Error: %v", err)

				return
			}
		}
	}
}

// handleMessage processes different types of SDK messages.
func handleMessage(msg claude.SDKMessage) {
	switch m := msg.(type) {
	case *claude.SDKSystemMessage:
		handleSystemMessage(m)
	case *claude.SDKAssistantMessage:
		handleAssistantMessage(m)
	case *claude.SDKResultMessage:
		handleResultMessage(m)
	}
}

// handleSystemMessage processes system initialization messages.
func handleSystemMessage(m *claude.SDKSystemMessage) {
	if m.Subtype == "init" {
		fmt.Printf("Initialized with model: %v\n", m.Data["model"])
	}
}

// handleAssistantMessage displays assistant response content.
func handleAssistantMessage(m *claude.SDKAssistantMessage) {
	fmt.Println("\nAssistant response:")
	for _, block := range m.Message.Content {
		displayContentBlock(block)
	}
}

// displayContentBlock prints text content from a content block.
func displayContentBlock(block claude.ContentBlock) {
	switch b := block.(type) {
	case claude.TextBlock:
		fmt.Printf("  %s\n", b.Text)
	case claude.TextContentBlock:
		fmt.Printf("  %s\n", b.Text)
	}
}

// handleResultMessage displays final result statistics.
func handleResultMessage(m *claude.SDKResultMessage) {
	fmt.Printf("\nResult: %s\n", m.Subtype)
	fmt.Printf("Duration: %dms\n", m.DurationMS)
	fmt.Printf("Cost: $%.4f\n", m.TotalCostUSD)
	fmt.Printf("Turns: %d\n", m.NumTurns)
}
