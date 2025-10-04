// Package main demonstrates tool selection with the Claude Agent SDK.
//
// This example shows:
//   - Restricting allowed tools
//   - Using AllToolsExcept helper
//   - Configuring tool permissions
//   - Tool matchers for fine-grained control
//
// Prerequisites: Claude CLI must be installed and configured
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
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Example 1: Allow only specific tools
	restrictedOptions := &options.AgentOptions{
		AllowedTools: []options.BuiltinTool{
			options.ToolRead, // Allow reading files
			options.ToolGrep, // Allow searching files
			options.ToolGlob, // Allow finding files
			// Notably missing: ToolWrite, ToolEdit, ToolBash
		},
	}

	prompt1 := "Find all Go files in the project and show me the main function."
	fmt.Println("Example 1: Restricted to read-only tools")
	fmt.Printf("Prompt: %s\n\n", prompt1)

	runQuery(ctx, prompt1, restrictedOptions)

	// Example 2: Allow all tools except dangerous ones
	safeOptions := &options.AgentOptions{
		DisallowedTools: []options.BuiltinTool{
			options.ToolBash,      // Disallow command execution
			options.ToolWrite,     // Disallow file writing
			options.ToolEdit,      // Disallow file editing
			options.ToolWebFetch,  // Disallow web access
			options.ToolWebSearch, // Disallow web search
		},
	}

	prompt2 := "Analyze the structure of this project."
	fmt.Println("\nExample 2: All tools except dangerous ones")
	fmt.Printf("Prompt: %s\n\n", prompt2)

	runQuery(ctx, prompt2, safeOptions)

	// Example 3: Fine-grained control with tool matchers
	// Only allow git commands via Bash tool
	gitOnlyOptions := &options.AgentOptions{
		AllowedTools: []options.BuiltinTool{
			options.ToolRead, // Reading is safe
		},
	}

	prompt3 := "Show me the git status and recent commits."
	fmt.Println("\nExample 3: Fine-grained control (git commands only)")
	fmt.Printf("Prompt: %s\n\n", prompt3)

	runQuery(ctx, prompt3, gitOnlyOptions)
}

// runQuery executes a query with the given options and prints the response.
func runQuery(ctx context.Context, prompt string, opts *options.AgentOptions) {
	msgCh, errCh := claude.Query(ctx, prompt, opts, nil)

	// Process messages
	for msg := range msgCh {
		switch m := msg.(type) {
		case messages.AssistantMessage:
			fmt.Println("Response received:")
			for _, block := range m.Content {
				fmt.Printf("  %+v\n", block)
			}
		}
	}

	// Check for errors
	if err := <-errCh; err != nil {
		log.Printf("Query error: %v", err)
		return
	}

	fmt.Println("Query completed successfully!")
}
