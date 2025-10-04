// Package main demonstrates hook usage with the Claude Agent SDK.
//
// This example shows:
//   - Registering PreToolUse and PostToolUse hooks
//   - Implementing hook callbacks
//   - Inspecting tool usage in real-time
//   - Hook context and error handling
//
// Prerequisites: Claude CLI must be installed and configured
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/conneroisu/claude/pkg/claude"
	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/messages"
)

func main() {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	// Register hooks for tool usage monitoring
	hooks := map[hooking.HookEvent][]hooking.HookMatcher{
		// PreToolUse fires before any tool is executed
		hooking.HookEventPreToolUse: {
			{
				Matcher: "*", // Match all tools
				Hooks:   []hooking.HookCallback{preToolUseCallback},
			},
		},
		// PostToolUse fires after any tool completes
		hooking.HookEventPostToolUse: {
			{
				Matcher: "*", // Match all tools
				Hooks:   []hooking.HookCallback{postToolUseCallback},
			},
		},
	}

	// Create a prompt that will trigger tool usage
	// This asks Claude to list files, which uses the Glob or Bash tool
	prompt := "Please list the Go files in the current directory."

	fmt.Println("Executing query with hooks enabled...")
	fmt.Printf("Prompt: %s\n\n", prompt)

	if err := run(ctx, prompt, hooks); err != nil {
		cancel()
		log.Fatalf("Error: %v", err)
	}

	cancel()
	fmt.Println("\nQuery with hooks completed successfully!")
}

func run(
	ctx context.Context,
	prompt string,
	hooks map[hooking.HookEvent][]hooking.HookMatcher,
) error {
	// Execute the query with hooks
	msgCh, errCh := claude.Query(ctx, prompt, nil, hooks)

	// Process messages
	for msg := range msgCh {
		switch m := msg.(type) {
		case messages.AssistantMessage:
			fmt.Println("\nClaude's response:")
			for _, block := range m.Content {
				fmt.Printf("%+v\n", block)
			}
		}
	}

	// Check for errors
	if err := <-errCh; err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	return nil
}

// preToolUseCallback is called before a tool is executed.
func preToolUseCallback(
	input map[string]any,
	toolUseID *string,
	ctx hooking.HookContext,
) (map[string]any, error) {
	// Extract tool information from the hook input
	toolName, _ := input["tool_name"].(string)
	toolInput := input["tool_input"]

	fmt.Printf("\n[PRE-HOOK] Tool about to execute:\n")
	fmt.Printf("  Tool Name: %s\n", toolName)
	fmt.Printf("  Tool Input: %+v\n", toolInput)
	if toolUseID != nil {
		fmt.Printf("  Tool Use ID: %s\n", *toolUseID)
	}

	// Check if the hook context is cancelled
	select {
	case <-ctx.Signal.Done():
		return nil, fmt.Errorf("hook cancelled: %w", ctx.Signal.Err())
	default:
		// Continue execution
	}

	// Return nil to allow the tool to execute
	// You could return modified input to change tool behavior
	return nil, nil
}

// postToolUseCallback is called after a tool completes execution.
func postToolUseCallback(
	input map[string]any,
	toolUseID *string,
	ctx hooking.HookContext,
) (map[string]any, error) {
	// Extract tool information and results
	toolName, _ := input["tool_name"].(string)
	toolResponse := input["tool_response"]

	fmt.Printf("\n[POST-HOOK] Tool execution completed:\n")
	fmt.Printf("  Tool Name: %s\n", toolName)
	fmt.Printf("  Tool Response: %+v\n", toolResponse)
	if toolUseID != nil {
		fmt.Printf("  Tool Use ID: %s\n", *toolUseID)
	}

	// Check for cancellation
	select {
	case <-ctx.Signal.Done():
		return nil, fmt.Errorf("hook cancelled: %w", ctx.Signal.Err())
	default:
		// Continue execution
	}

	// Return nil to continue normally
	// You could log results, validate output, or modify the response
	return nil, nil
}
