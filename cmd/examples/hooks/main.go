// Package main demonstrates how to use hooks to intercept and control
// tool usage in Claude Agent SDK. Hooks allow you to block dangerous
// commands, log tool usage, and modify tool inputs before execution.
//
// This example shows:
//   - Blocking dangerous bash commands using pattern matching
//   - Logging all tool uses with custom hook callbacks
//   - Processing different message types from the agent
//   - Handling permission denials and errors
package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/conneroisu/claude/pkg/claude"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
)

// errChannelClosed signals that a channel has been closed.
var errChannelClosed = errors.New("channel closed")

// main demonstrates hook usage by executing a query with safety
// controls that prevent dangerous bash commands from running.
func main() {
	// Configure hooks to block dangerous bash commands.
	// Hooks are organized by event type (PreToolUse, PostToolUse, etc).
	// Each hook can match specific tools and apply multiple callbacks.
	hooks := createSecurityHooks()

	// Configure the agent with the Claude Sonnet 4.5 model.
	// This model provides excellent reasoning and tool use capabilities.
	opts := &options.AgentOptions{
		Model: stringPtr("claude-sonnet-4-5-20250929"),
	}

	// Execute a query with our security hooks enabled.
	// The hooks will intercept any bash commands before execution.
	ctx := context.Background()
	config := &claude.QueryConfig{
		Hooks: hooks,
	}
	query := "List the files in the current directory"
	msgCh, errCh := claude.Query(ctx, query, opts, config)

	// Process messages from the agent until completion.
	// Messages arrive on channels as the agent reasons and acts.
	if err := processChannels(msgCh, errCh); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

// createSecurityHooks builds a hook configuration that blocks dangerous
// bash commands and logs all tool usage for security monitoring.
func createSecurityHooks() map[claude.HookEvent][]claude.HookMatcher {
	// Define patterns that should never be allowed in bash commands.
	// These patterns can cause data loss or system damage.
	dangerousPatterns := []string{"rm -rf", "dd if=", "mkfs"}

	return map[claude.HookEvent][]claude.HookMatcher{
		// PreToolUse hooks run before any tool is executed.
		// This is the best place to block dangerous operations.
		claude.HookEventPreToolUse: {
			{
				// Only apply these hooks to the Bash tool.
				Matcher: "Bash",
				Hooks: []claude.HookCallback{
					// Block bash commands matching patterns.
					claude.BlockBashPatternHook(
						dangerousPatterns,
					),
					// Log all bash commands for monitoring.
					logToolUseHook,
				},
			},
		},
	}
}

// processChannels reads from message and error channels until both close.
// This is the main event loop for receiving agent responses.
// Returns an error if one occurs on the error channel.
func processChannels(
	msgCh <-chan messages.Message,
	errCh <-chan error,
) error {
	for {
		err := processNextEvent(msgCh, errCh)
		if errors.Is(err, errChannelClosed) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// channelEvent represents an event received from a channel.
type channelEvent struct {
	message messages.Message
	err     error
	closed  bool
}

// processNextEvent waits for and handles the next channel event.
// Returns errChannelClosed when channels close, or an error if one occurs.
func processNextEvent(
	msgCh <-chan messages.Message,
	errCh <-chan error,
) error {
	select {
	case msg, open := <-msgCh:
		return handleMessageEvent(channelEvent{
			message: msg,
			closed:  !open,
		})

	case err, open := <-errCh:
		return handleErrorEvent(channelEvent{
			err:    err,
			closed: !open,
		})
	}
}

// handleMessageEvent processes a message from the message channel.
// Returns errChannelClosed if the channel is closed.
func handleMessageEvent(event channelEvent) error {
	// Channel closed means agent finished processing.
	if event.closed {
		return errChannelClosed
	}
	handleMessage(event.message)

	return nil
}

// handleErrorEvent processes an error from the error channel.
// Returns errChannelClosed if channel closed, or the error if one occurred.
func handleErrorEvent(event channelEvent) error {
	// Channel closed means no more errors will arrive.
	if event.closed {
		return errChannelClosed
	}

	// Return any error for handling in main.
	return event.err
}

// logToolUseHook is a custom hook callback that logs all tool usage.
// It demonstrates how to inspect tool inputs before execution.
// The underscore parameters are required by the interface but unused.
func logToolUseHook(
	input map[string]any,
	_ *string,
	_ claude.HookContext,
) (map[string]any, error) {
	toolName, _ := input["tool_name"].(string)
	toolInput, _ := input["tool_input"].(map[string]any)

	fmt.Printf("\n[Hook] Tool use detected: %s\n", toolName)

	// For bash commands, log the actual command being executed.
	// This helps with security monitoring and debugging.
	if toolName == "Bash" {
		if cmd, ok := toolInput["command"].(string); ok {
			fmt.Printf("[Hook] Bash command: %s\n", cmd)
		}
	}

	// Return empty map to allow the tool use to proceed.
	// We could return modified input to change the tool's behavior.
	return make(map[string]any), nil
}

// handleMessage processes different message types from the agent.
// Each message type provides different information about the agent's
// reasoning, actions, and final results.
func handleMessage(msg messages.Message) {
	switch m := msg.(type) {
	case *messages.AssistantMessage:
		// Assistant messages contain the agent's reasoning and actions.
		handleAssistantMessage(m)

	case *messages.ResultMessageSuccess:
		// Success messages provide metrics about the completed query.
		handleSuccessMessage(m)

	case *messages.ResultMessageError:
		// Error messages indicate the query failed to complete.
		handleErrorMessage(m)

	case *messages.SystemMessage:
		// System messages provide lifecycle events.
		handleSystemMessage(m)
	}
}

// handleAssistantMessage processes content blocks from the assistant.
// Assistant messages can contain text, thinking, or tool use blocks.
func handleAssistantMessage(m *messages.AssistantMessage) {
	for _, block := range m.Content {
		switch b := block.(type) {
		case messages.TextBlock:
			// Text blocks contain the assistant's response.
			fmt.Printf("Assistant: %s\n", b.Text)
		case messages.ThinkingBlock:
			// Thinking blocks show the assistant's reasoning.
			fmt.Printf("[Thinking]: %s\n", b.Thinking)
		case messages.ToolUseBlock:
			// Tool use blocks indicate an action being taken.
			fmt.Printf("[Tool Use]: %s\n", b.Name)
		}
	}
}

// handleSuccessMessage displays metrics and warnings from successful
// query completion, including cost, duration, and permission denials.
func handleSuccessMessage(m *messages.ResultMessageSuccess) {
	fmt.Printf(
		"\n✓ Success in %dms (%d turns)\n",
		m.DurationMs,
		m.NumTurns,
	)
	fmt.Printf("  Cost: $%.4f\n", m.TotalCostUSD)

	// Permission denials occur when hooks block tool usage.
	// This is expected behavior when security controls trigger.
	if len(m.PermissionDenials) == 0 {
		return
	}

	fmt.Printf(
		"  ⚠ Permission denials: %d\n",
		len(m.PermissionDenials),
	)
	for _, denial := range m.PermissionDenials {
		fmt.Printf("    - %s\n", denial.ToolName)
	}
}

// handleErrorMessage displays information about query errors.
// The error subtype provides details about what went wrong.
func handleErrorMessage(m *messages.ResultMessageError) {
	fmt.Printf("\n✗ Error: %s\n", m.Subtype)
}

// handleSystemMessage processes system lifecycle events.
// The init event indicates the agent session started successfully.
func handleSystemMessage(m *messages.SystemMessage) {
	if m.Subtype == "init" {
		fmt.Println("Session initialized")
	}
}

// stringPtr is a helper function that returns a pointer to a string.
// This is useful for setting optional fields in option structs.
func stringPtr(s string) *string {
	return &s
}
