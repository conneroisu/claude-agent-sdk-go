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

	// Configure options
	opts := &options.AgentOptions{
		Model: strPtr("claude-sonnet-4-5-20250929"),
	}

	// Perform a one-shot query
	msgCh, errCh := claude.Query(
		ctx,
		"What is the capital of France?",
		opts,
		nil, // no hooks
		nil, // no custom permissions
	)

	// Process messages
	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				return
			}
			handleMessage(msg)

		case err, ok := <-errCh:
			if !ok {
				return
			}
			if err != nil {
				log.Fatalf("Error: %v", err)
			}
		}
	}
}

func handleMessage(msg messages.Message) {
	switch m := msg.(type) {
	case *messages.AssistantMessage:
		fmt.Println("Assistant:")
		for _, block := range m.Content {
			switch b := block.(type) {
			case messages.TextBlock:
				fmt.Println(b.Text)
			case messages.ThinkingBlock:
				fmt.Printf("[Thinking: %s]\n", b.Thinking)
			}
		}

	case *messages.ResultMessage:
		fmt.Printf("\nResult: %v\n", m.Result)
		fmt.Printf("Duration: %dms\n", m.DurationMs)
		if m.TotalCostUSD != nil {
			fmt.Printf("Cost: $%.4f\n", *m.TotalCostUSD)
		}

	case *messages.SystemMessage:
		fmt.Printf("System: %s\n", m.Subtype)
	}
}

func strPtr(s string) *string {
	return &s
}
