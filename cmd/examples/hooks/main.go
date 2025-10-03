package main

import (
	"context"
	"fmt"
	"log"

	"github.com/conneroisu/claude/pkg/claude"
	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
)

func main() {
	ctx := context.Background()

	hooks := map[hooking.HookEvent][]hooking.HookMatcher{
		hooking.HookEventPreToolUse: {
			{
				Matcher: "Bash",
				Hooks: []hooking.HookCallback{
					bashSecurityHook(),
				},
			},
		},
		hooking.HookEventPostToolUse: {
			{
				Matcher: "*",
				Hooks: []hooking.HookCallback{
					loggingHook(),
				},
			},
		},
	}

	opts := &options.AgentOptions{
		MaxTurns: intPtr(3),
	}

	query := "List all files and then create a file named test.txt"
	fmt.Printf("Query: %s\n\n", query)

	msgCh, errCh := claude.Query(ctx, query, opts, hooks)

	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				return
			}

			switch m := msg.(type) {
			case *messages.AssistantMessage:
				fmt.Println("Claude says:")
				for _, block := range m.Content {
					if textBlock, ok := block.(messages.TextBlock); ok {
						fmt.Printf("  %s\n", textBlock.Text)
					}
				}

			case *messages.ResultMessageSuccess:
				fmt.Printf(
					"\nCompleted in %dms (%d turns)\n",
					m.DurationMs,
					m.NumTurns,
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

func bashSecurityHook() hooking.HookCallback {
	forbiddenPatterns := []string{"rm -rf", "sudo", ">"}

	return claude.BlockBashPatternHook(forbiddenPatterns)
}

func loggingHook() hooking.HookCallback {
	return func(
		input map[string]any,
		toolUseID *string,
		ctx hooking.HookContext,
	) (map[string]any, error) {
		toolName, _ := input["tool_name"].(string)
		fmt.Printf("[POST-HOOK] Tool %s completed\n", toolName)

		return map[string]any{}, nil
	}
}

func intPtr(i int) *int {
	return &i
}
