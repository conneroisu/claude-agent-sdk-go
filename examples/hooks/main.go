// Package main demonstrates hook callbacks in the Claude Agent SDK.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/conneroisu/claude-agent-sdk-go/pkg/claude"
)

// Hook Callbacks Example
//
// This example demonstrates how to use hook callbacks to monitor and observe
// Claude's behavior during execution. Hooks provide visibility into:
// - Session lifecycle (start/end)
// - Tool usage (before/after execution)
// - Notifications and warnings
// - User prompts and interactions
//
// Key features demonstrated:
// - Session lifecycle hooks (SessionStart, SessionEnd)
// - Tool monitoring hooks (PreToolUse, PostToolUse)
// - Notification hooks for warnings/errors
// - Tracking metrics across the session
//
// Hook callbacks are useful for:
// - Logging and observability
// - Metrics collection
// - Debugging and troubleshooting
// - Implementing custom permission logic
// - Auditing tool usage
//
// Available hook events:
// - PreToolUse: Before a tool executes (can deny execution)
// - PostToolUse: After a tool executes
// - SessionStart: When a session begins
// - SessionEnd: When a session ends
// - Notification: System notifications/warnings
// - UserPromptSubmit: When user submits a prompt
// - PreCompact: Before conversation compaction
// - Stop: When execution stops
// - SubagentStop: When a subagent stops

// SessionTracker tracks session lifecycle metrics.
type SessionTracker struct {
	startTime time.Time
	toolCalls int
}

var sessionTracker = &SessionTracker{}

func main() {
	ctx := context.Background()

	fmt.Println("=== Hook Callbacks Example ===")
	fmt.Println("This example demonstrates using hooks to monitor Claude's behavior")

	// Create client with hooks for observability
	// Hooks allow you to observe and react to various events during execution
	opts := &claude.Options{
		Model:    "claude-sonnet-4-5",
		MaxTurns: 5,
		Hooks: map[claude.HookEvent][]claude.HookCallbackMatcher{
			// Session lifecycle hooks
			// These track when sessions start and end
			claude.HookEventSessionStart: {
				{Hooks: []claude.HookCallback{onSessionStart}},
			},
			claude.HookEventSessionEnd: {
				{Hooks: []claude.HookCallback{onSessionEnd}},
			},

			// Tool usage hooks for monitoring
			// PreToolUse fires before each tool execution
			// PostToolUse fires after each tool completes
			claude.HookEventPreToolUse: {
				{Hooks: []claude.HookCallback{onPreToolUse}},
			},
			claude.HookEventPostToolUse: {
				{Hooks: []claude.HookCallback{onPostToolUse}},
			},

			// Notification hook for warnings/errors
			// Captures system notifications and warnings
			claude.HookEventNotification: {
				{Hooks: []claude.HookCallback{onNotification}},
			},
		},
	}

	client, err := claude.NewClient(opts)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			log.Printf("Failed to close client: %v", closeErr)
		}
	}()

	// Send a query that will trigger tool usage
	query := "What files are in the current directory? List them."
	fmt.Printf("Query: %s\n\n", query)

	if err = client.Query(ctx, query); err != nil {
		log.Printf("Failed to send query: %v", err)

		return
	}

	// Receive responses
	msgChan, errChan := client.ReceiveMessages(ctx)

	for {
		select {
		case msg := <-msgChan:
			if msg == nil {
				return
			}

			switch m := msg.(type) {
			case *claude.SDKAssistantMessage:
				fmt.Println("\n📝 Assistant response:")
				for _, block := range m.Message.Content {
					switch b := block.(type) {
					case claude.TextBlock:
						fmt.Printf("  %s\n", b.Text)
					case claude.TextContentBlock:
						fmt.Printf("  %s\n", b.Text)
					}
				}

			case *claude.SDKResultMessage:
				fmt.Printf("\n✓ Result: %s\n", m.Subtype)
			}

		case err := <-errChan:
			if err != nil {
				log.Printf("Error: %v", err)

				return
			}
		}
	}
}

// onSessionStart is called when a session begins.
func onSessionStart(
	_ context.Context,
	input claude.HookInput,
	_ *string,
) (claude.HookJSONOutput, error) {
	sessionInput, ok := input.(*claude.SessionStartHookInput)
	if !ok {
		return nil, errors.New("unexpected input type")
	}

	sessionTracker.startTime = time.Now()
	sessionTracker.toolCalls = 0

	fmt.Printf("🚀 Session started\n")
	fmt.Printf("   Source: %s\n", sessionInput.Source)
	fmt.Printf("   Session ID: %s\n", sessionInput.SessionID())
	fmt.Printf("   Working dir: %s\n\n", sessionInput.Cwd())

	return claude.SyncHookOutput{}, nil
}

// onSessionEnd is called when a session ends.
func onSessionEnd(
	_ context.Context,
	input claude.HookInput,
	_ *string,
) (claude.HookJSONOutput, error) {
	sessionInput, ok := input.(*claude.SessionEndHookInput)
	if !ok {
		return nil, errors.New("unexpected input type")
	}

	duration := time.Since(sessionTracker.startTime)

	fmt.Printf("\n🏁 Session ended\n")
	fmt.Printf("   Reason: %s\n", sessionInput.Reason)
	fmt.Printf("   Duration: %v\n", duration.Round(time.Millisecond))
	fmt.Printf("   Total tool calls: %d\n", sessionTracker.toolCalls)

	return claude.SyncHookOutput{}, nil
}

// onPreToolUse is called before a tool is executed.
func onPreToolUse(
	_ context.Context,
	input claude.HookInput,
	_ *string,
) (claude.HookJSONOutput, error) {
	toolInput, ok := input.(*claude.PreToolUseHookInput)
	if !ok {
		return nil, errors.New("unexpected input type")
	}

	sessionTracker.toolCalls++

	// Pretty print the tool input
	var prettyInput any
	if err := json.Unmarshal(toolInput.ToolInput, &prettyInput); err != nil {
		prettyInput = string(toolInput.ToolInput)
	}
	inputJSON, _ := json.MarshalIndent(prettyInput, "     ", "  ")

	fmt.Printf("\n🔧 Tool call #%d: %s\n", sessionTracker.toolCalls, toolInput.ToolName)
	fmt.Printf("   Input: %s\n", string(inputJSON))

	// Allow the tool to proceed
	return claude.SyncHookOutput{}, nil
}

// onPostToolUse is called after a tool has executed.
func onPostToolUse(
	_ context.Context,
	input claude.HookInput,
	_ *string,
) (claude.HookJSONOutput, error) {
	toolInput, ok := input.(*claude.PostToolUseHookInput)
	if !ok {
		return nil, errors.New("unexpected input type")
	}

	// Pretty print the tool response
	var prettyResponse any
	if err := json.Unmarshal(toolInput.ToolResponse, &prettyResponse); err != nil {
		prettyResponse = string(toolInput.ToolResponse)
	}

	fmt.Printf("   ✓ %s completed\n", toolInput.ToolName)

	return claude.SyncHookOutput{}, nil
}

// onNotification is called for warnings and errors.
func onNotification(
	_ context.Context,
	input claude.HookInput,
	_ *string,
) (claude.HookJSONOutput, error) {
	notifInput, ok := input.(*claude.NotificationHookInput)
	if !ok {
		return nil, errors.New("unexpected input type")
	}

	title := "Notification"
	if notifInput.Title != nil {
		title = *notifInput.Title
	}

	fmt.Printf("\n⚠️  %s: %s\n", title, notifInput.Message)

	return claude.SyncHookOutput{}, nil
}
