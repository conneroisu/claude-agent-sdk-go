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

	opts := &options.AgentOptions{
		MaxTurns: intPtr(3),
		AllowedTools: []options.BuiltinTool{
			options.ToolRead,
			options.ToolGlob,
			options.ToolGrep,
		},
		DisallowedTools: []options.BuiltinTool{
			options.ToolBash,
			options.ToolWrite,
			options.ToolEdit,
		},
	}

	query := "Analyze the Go files in pkg/claude directory"
	fmt.Printf("Query: %s\n", query)
	fmt.Println("Allowed tools: Read, Glob, Grep")
	fmt.Println("Blocked tools: Bash, Write, Edit")

	msgCh, errCh := claude.Query(ctx, query, opts, nil)

	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				return
			}

			switch m := msg.(type) {
			case *messages.AssistantMessage:
				fmt.Println("\nClaude says:")
				for _, block := range m.Content {
					switch b := block.(type) {
					case messages.TextBlock:
						fmt.Printf("  %s\n", b.Text)

					case messages.ToolUseBlock:
						fmt.Printf(
							"  [Using tool: %s]\n",
							b.Name,
						)
					}
				}

			case *messages.ResultMessageSuccess:
				fmt.Printf(
					"\nCompleted in %dms (%d turns)\n",
					m.DurationMs,
					m.NumTurns,
				)

			case *messages.ResultMessageError:
				fmt.Printf(
					"\nError: %s\n",
					m.Subtype,
				)
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
