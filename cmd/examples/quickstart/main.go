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

	// Create agent options
	opts := &options.AgentOptions{
		MaxTurns: intPtr(1),
	}

	// Execute a simple query
	msgCh, errCh := claude.Query(ctx, "What is 2 + 2?", opts, nil)

	// Process messages
	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				// Channel closed, query complete
				return
			}

			// Handle different message types
			switch m := msg.(type) {
			case *messages.AssistantMessage:
				fmt.Println("Claude says:")
				for _, block := range m.Content {
					if textBlock, ok := block.(messages.TextBlock); ok {
						fmt.Printf("  %s\n", textBlock.Text)
					}
				}

			case *messages.ResultMessageSuccess:
				fmt.Printf("\nQuery completed successfully in %dms\n", m.DurationMs)

			case *messages.ResultMessageError:
				fmt.Printf("\nQuery failed: %s\n", m.Subtype)
			}

		case err := <-errCh:
			if err != nil {
				log.Fatalf("Error: %v", err)
			}
			return
		}
	}
}

func intPtr(i int) *int {
	return &i
}
