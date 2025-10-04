// Package main demonstrates a simple quickstart example.
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
	ctx := context.Background()

	// Set max turns to limit conversation length
	maxTurns := 1
	opts := &options.AgentOptions{
		MaxTurns: &maxTurns,
	}

	// Execute a one-shot query
	msgCh, errCh := claude.Query(
		ctx,
		"What is 2 + 2? Please explain briefly.",
		opts,
		nil,
	)

	// Process responses
	if err := processResponses(msgCh, errCh); err != nil {
		log.Fatal(err)
	}
}

// processResponses handles incoming messages and errors.
func processResponses(
	msgCh <-chan messages.Message,
	errCh <-chan error,
) error {
	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				return nil
			}

			handleMessage(msg)

		case err := <-errCh:
			if err != nil {
				return err
			}

			return nil
		}
	}
}

// handleMessage processes a single message.
func handleMessage(msg messages.Message) {
	assistantMsg, ok := msg.(*messages.AssistantMessage)
	if !ok {
		return
	}

	printTextBlocks(assistantMsg.Content)
}

// printTextBlocks prints all text blocks from content.
func printTextBlocks(content []messages.ContentBlock) {
	for _, block := range content {
		textBlock, ok := block.(messages.TextBlock)
		if !ok {
			continue
		}

		fmt.Printf("Claude: %s\n", textBlock.Text)
	}
}
