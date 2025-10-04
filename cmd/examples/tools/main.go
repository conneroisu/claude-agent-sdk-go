// Package main demonstrates tool filtering.
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

	// Configure tool access
	opts := buildOptions()

	// Execute query
	msgCh, errCh := claude.Query(
		ctx,
		"Read the README.md file",
		opts,
		nil,
	)

	// Process responses
	if err := processResponses(msgCh, errCh); err != nil {
		log.Fatal(err)
	}
}

// buildOptions creates options with allowed tools.
func buildOptions() *options.AgentOptions {
	maxTurns := 1

	return &options.AgentOptions{
		MaxTurns: &maxTurns,
		AllowedTools: []options.BuiltinTool{
			options.ToolBash,
			options.ToolRead,
		},
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

	printBlocks(assistantMsg.Content)
}

// printBlocks prints all content blocks.
func printBlocks(content []messages.ContentBlock) {
	for _, block := range content {
		printBlock(block)
	}
}

// printBlock prints a single content block.
func printBlock(block messages.ContentBlock) {
	switch b := block.(type) {
	case messages.TextBlock:
		fmt.Printf("Claude: %s\n", b.Text)
	case messages.ToolUseBlock:
		fmt.Printf("Tool used: %s\n", b.Name)
	}
}
