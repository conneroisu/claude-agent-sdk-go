// Package main demonstrates basic usage of the Claude Agent SDK.
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
	// Create options with basic configuration
	opts := options.DefaultOptions()
	opts.Model = "claude-sonnet-4-5"
	opts.SystemPrompt = "You are a helpful assistant."

	// Execute a simple query
	msgCh, errCh := claude.Query(
		context.Background(),
		"What is the capital of France?",
		opts,
		nil, // no hooks
	)

	// Process messages
	for msg := range msgCh {
		switch m := msg.(type) {
		case *messages.AssistantMessage:
			printAssistantMessage(m)
		case *messages.ResultMessageSuccess:
			fmt.Println("\n✓ Conversation completed successfully")
		case *messages.ResultMessageError:
			fmt.Printf("\n✗ Error: %s - %s\n", m.ErrorType, m.ErrorMessage)
		}
	}

	// Check for errors
	if err := <-errCh; err != nil {
		log.Fatalf("Query failed: %v", err)
	}
}

func printAssistantMessage(msg *messages.AssistantMessage) {
	fmt.Println("\nAssistant:")
	for _, block := range msg.Content {
		switch b := block.(type) {
		case *messages.TextBlock:
			fmt.Println(b.Text)
		case *messages.ThinkingBlock:
			fmt.Printf("[Thinking: %s]\n", b.Thinking)
		case *messages.ToolUseBlock:
			fmt.Printf("[Tool Use: %s(%v)]\n", b.Name, b.Input)
		}
	}
}
