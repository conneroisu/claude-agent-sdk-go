// Package main demonstrates streaming conversation example.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/conneroisu/claude/pkg/claude"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
)

func main() {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		60*time.Second, //nolint:revive // Standard timeout duration
	)
	defer cancel()

	if err := run(ctx); err != nil {
		log.Fatal(err) //nolint:gocritic // Example accepts cancel not running
	}
}

// run executes the streaming conversation.
func run(ctx context.Context) error {
	// Create client
	maxTurns := 2
	client, err := claude.NewClient(&options.AgentOptions{
		MaxTurns: &maxTurns,
	})
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	// Connect to Claude
	if err := client.Connect(ctx, nil); err != nil {
		return err
	}

	// Send first message
	if err := client.SendMessage(ctx, "Hello! How are you?"); err != nil {
		return err
	}

	// Receive responses
	msgCh, errCh := client.ReceiveMessages(ctx)

	return processStream(ctx, msgCh, errCh)
}

// processStream handles the message stream.
func processStream(
	ctx context.Context,
	msgCh <-chan messages.Message,
	errCh <-chan error,
) error {
	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				fmt.Println("Conversation complete")

				return nil
			}

			handleMessage(msg)

		case err := <-errCh:
			return err

		case <-ctx.Done():
			fmt.Println("Timeout")

			return ctx.Err()
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
