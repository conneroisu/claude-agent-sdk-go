// Package main demonstrates fine-grained permission control for Claude agents.
// This example shows how to use the CanUseTool callback to control which
// tools Claude can use, implementing allow/deny rules for different tool types.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/conneroisu/claude-agent-sdk-go/pkg/claude"
	"github.com/conneroisu/claude-agent-sdk-go/pkg/clauderrs"
)

const (
	maxTurns              = 10
	maxValueDisplayLength = 40
)

func main() {
	ctx := context.Background()

	// Create client with permission controls.
	opts := &claude.Options{
		Model:      "claude-sonnet-4-5",
		MaxTurns:   maxTurns,
		CanUseTool: canUseToolCallback, // Custom permission logic.
	}

	client, err := claude.NewClient(opts)
	if err != nil {
		log.Fatal(handleClientCreationError(err))
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			log.Printf("Failed to close client: %v", closeErr)
		}
	}()

	displayPermissionRules()

	// Send a query that will attempt various operations
	query := `List all .go files in the examples directory.
	Then read one of them and tell me what it does.
	Do NOT try to write any files or run any bash commands.`

	fmt.Printf("Query: %s\n\n", query)

	if err = client.Query(ctx, query); err != nil {
		if closeErr := client.Close(); closeErr != nil {
			log.Printf(
				"Failed to close client during error cleanup: %v",
				closeErr,
			)
		}
		log.Println(handleQueryError(err))

		return
	}

	// Receive responses
	processMessages(ctx, client)
}

// handleClientCreationError provides detailed error handling
// for client creation.
func handleClientCreationError(err error) error {
	if clauderrs.IsValidationError(err) {
		return fmt.Errorf("invalid client configuration: %w", err)
	}

	if clauderrs.IsProcessError(err) {
		var procErr *clauderrs.ProcessError
		if errors.As(err, &procErr) {
			return fmt.Errorf(
				"failed to start Claude process "+
					"(exit code %d): %s",
				procErr.ExitCode(),
				procErr.Stderr(),
			)
		}

		return fmt.Errorf("failed to start Claude process: %w", err)
	}

	return fmt.Errorf("failed to create client: %w", err)
}

// handleQueryError provides detailed error handling for query operations.
func handleQueryError(err error) error {
	// Check for API errors first
	if clauderrs.IsAPIError(err) {
		sdkErr, _ := clauderrs.AsSDKError(err)

		switch sdkErr.Code() {
		case clauderrs.ErrCodeAPIRateLimit:
			// Extract retry information
			meta := sdkErr.Metadata()
			if retryAfter, ok := meta["retry_after_seconds"].(float64); ok {
				return fmt.Errorf(
					"rate limited. Please wait %v seconds "+
						"before retrying",
					retryAfter,
				)
			}

			return fmt.Errorf("rate limited: %w", err)

		case clauderrs.ErrCodeAPIUnauthorized:
			return errors.New(
				"invalid API key. Please set " +
					"ANTHROPIC_API_KEY environment variable",
			)

		case clauderrs.ErrCodeAPIServerError:
			return fmt.Errorf(
				"API server error. "+
					"Please retry in a few moments: %w",
				err,
			)

		case clauderrs.ErrCodeAPIBadRequest,
			clauderrs.ErrCodeAPIForbidden,
			clauderrs.ErrCodeAPINotFound,
			clauderrs.ErrCodeClientClosed,
			clauderrs.ErrCodeNoActiveQuery,
			clauderrs.ErrCodeInvalidState,
			clauderrs.ErrCodeMissingAPIKey,
			clauderrs.ErrCodeInvalidConfig,
			clauderrs.ErrCodeNetworkTimeout,
			clauderrs.ErrCodeConnectionFailed,
			clauderrs.ErrCodeConnectionClosed,
			clauderrs.ErrCodeDNSError,
			clauderrs.ErrCodeInvalidMessage,
			clauderrs.ErrCodeMessageParseFailed,
			clauderrs.ErrCodeUnknownMessageType,
			clauderrs.ErrCodeProtocolError,
			clauderrs.ErrCodeIOError,
			clauderrs.ErrCodeReadFailed,
			clauderrs.ErrCodeWriteFailed,
			clauderrs.ErrCodeTransportInit,
			clauderrs.ErrCodeProcessNotFound,
			clauderrs.ErrCodeProcessSpawnFailed,
			clauderrs.ErrCodeProcessCrashed,
			clauderrs.ErrCodeProcessExited,
			clauderrs.ErrCodeMissingField,
			clauderrs.ErrCodeInvalidType,
			clauderrs.ErrCodeRangeViolation,
			clauderrs.ErrCodeInvalidFormat,
			clauderrs.ErrCodeToolDenied,
			clauderrs.ErrCodeDirectoryDenied,
			clauderrs.ErrCodeResourceDenied,
			clauderrs.ErrCodeCallbackFailed,
			clauderrs.ErrCodeCallbackTimeout,
			clauderrs.ErrCodeHookFailed,
			clauderrs.ErrCodeHookTimeout:
			return fmt.Errorf("API error: %w", err)

		default:
			return fmt.Errorf("unexpected error code: %w", err)
		}
	}

	// Check for network errors
	if clauderrs.IsNetworkError(err) {
		if clauderrs.IsNetworkTimeout(err) {
			return errors.New(
				"request timed out. " +
					"Try increasing the timeout",
			)
		}

		return fmt.Errorf(
			"network error. Check your internet connection: %w",
			err,
		)
	}

	// Check for client state errors
	if clauderrs.IsClientError(err) {
		sdkErr, _ := clauderrs.AsSDKError(err)
		if sdkErr.Code() == clauderrs.ErrCodeClientClosed {
			return errors.New("client is closed. Cannot send query")
		}

		return fmt.Errorf("client error: %w", err)
	}

	return fmt.Errorf("query failed: %w", err)
}

// handleStreamError provides error handling during message streaming.
func handleStreamError(err error) error {
	// Protocol errors during streaming
	if clauderrs.IsProtocolError(err) {
		sdkErr, _ := clauderrs.AsSDKError(err)
		if sdkErr.Code() == clauderrs.ErrCodeMessageParseFailed {
			return fmt.Errorf(
				"failed to parse message. "+
					"This may indicate a version mismatch: %w",
				err,
			)
		}

		return fmt.Errorf("protocol error: %w", err)
	}

	// Permission callback errors
	if clauderrs.IsPermissionError(err) {
		var permErr *clauderrs.PermissionError
		if errors.As(err, &permErr) {
			return fmt.Errorf(
				"permission denied: %w",
				err,
			)
		}

		return fmt.Errorf("permission error: %w", err)
	}

	// Callback execution errors
	if clauderrs.IsCallbackError(err) {
		sdkErr, _ := clauderrs.AsSDKError(err)
		if sdkErr.Code() == clauderrs.ErrCodeCallbackTimeout {
			return fmt.Errorf("permission callback timed out: %w", err)
		}

		return fmt.Errorf("callback error: %w", err)
	}

	return fmt.Errorf("stream error: %w", err)
}

// canUseToolCallback implements fine-grained permission control.
// This demonstrates proper error handling within permission callbacks.
func canUseToolCallback(
	ctx context.Context,
	toolName string,
	input map[string]claude.JSONValue,
	_ []claude.PermissionUpdate,
	toolUseID string,
	agentID *string,
	blockedPath *string,
	decisionReason *string,
) (claude.PermissionResult, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, clauderrs.NewCallbackError(
			clauderrs.ErrCodeCallbackTimeout,
			"canUseTool context cancelled",
			ctx.Err(),
			"canUseTool",
			true,
		)
	default:
	}

	// Parse input for logging
	inputStr := formatToolInput(input)

	// Define permission rules
	switch {
	// Allow read-only operations
	case toolName == "Read":
		fmt.Printf("✓ Allowing Read: %s\n", inputStr)

		return &claude.PermissionAllow{
			Behavior:     claude.PermissionBehaviorAllow,
			UpdatedInput: input,
		}, nil

	case toolName == "Glob":
		fmt.Printf("✓ Allowing Glob: %s\n", inputStr)

		return &claude.PermissionAllow{
			Behavior:     claude.PermissionBehaviorAllow,
			UpdatedInput: input,
		}, nil

	case toolName == "Grep":
		fmt.Printf("✓ Allowing Grep: %s\n", inputStr)

		return &claude.PermissionAllow{
			Behavior:     claude.PermissionBehaviorAllow,
			UpdatedInput: input,
		}, nil

	// Block bash commands for security
	case toolName == "Bash":
		fmt.Printf("🚫 BLOCKED Bash: %s\n", inputStr)

		return &claude.PermissionDeny{
			Behavior: claude.PermissionBehaviorDeny,
			Message: "Bash commands are not allowed in " +
				"this session for security reasons",
		}, nil

	// Block write operations (in a real app, you might prompt user)
	case toolName == "Write" || toolName == "Edit":
		fmt.Printf("🚫 BLOCKED %s: %s\n", toolName, inputStr)

		return &claude.PermissionDeny{
			Behavior: claude.PermissionBehaviorDeny,
			Message: fmt.Sprintf(
				"%s operations require explicit approval",
				toolName,
			),
		}, nil

	// For demo purposes, block MCP tools too
	case strings.HasPrefix(toolName, "mcp__"):
		fmt.Printf("🚫 BLOCKED MCP tool: %s\n", toolName)

		return &claude.PermissionDeny{
			Behavior: claude.PermissionBehaviorDeny,
			Message:  "MCP tools are disabled in this session",
		}, nil

	// Allow other tools by default (but log them)
	default:
		fmt.Printf("✓ Allowing %s: %s\n", toolName, inputStr)

		return &claude.PermissionAllow{
			Behavior:     claude.PermissionBehaviorAllow,
			UpdatedInput: input,
		}, nil
	}
}

// displayPermissionRules prints the permission control rules.
func displayPermissionRules() {
	fmt.Println("Permission Control Example")
	fmt.Println("==========================")
	fmt.Println(
		"This example demonstrates fine-grained " +
			"tool permission control.",
	)
	fmt.Println("Rules:")
	fmt.Println("  ✓ Read operations: Allowed")
	fmt.Println("  ✓ Glob/Grep: Allowed")
	fmt.Println(
		"  ⚠ Write operations: " +
			"Require approval (will deny for demo)",
	)
	fmt.Println("  ✗ Bash commands: Blocked")
	fmt.Println()
}

// processMessages handles message streaming from the client.
func processMessages(
	ctx context.Context,
	client *claude.ClaudeSDKClient,
) {
	msgChan, errChan := client.ReceiveMessages(ctx)

	for {
		select {
		case msg := <-msgChan:
			if msg == nil {
				return
			}
			handleMessage(msg)

		case err := <-errChan:
			if err != nil {
				panic(handleStreamError(err))
			}
		}
	}
}

// handleMessage processes a single message from the stream.
func handleMessage(msg claude.SDKMessage) {
	switch m := msg.(type) {
	case *claude.SDKAssistantMessage:
		handleAssistantMessage(m)
	case *claude.SDKResultMessage:
		handleResultMessage(m)
	}
}

// handleAssistantMessage processes assistant messages.
func handleAssistantMessage(m *claude.SDKAssistantMessage) {
	fmt.Println("\n📝 Assistant response:")
	for _, block := range m.Message.Content {
		switch b := block.(type) {
		case claude.TextBlock:
			fmt.Printf("  %s\n", b.Text)
		case claude.TextContentBlock:
			fmt.Printf("  %s\n", b.Text)
		case claude.ToolUseContentBlock:
			fmt.Printf("  🔧 Using tool: %s\n", b.Name)
		}
	}
}

// handleResultMessage processes result messages with permission denials.
func handleResultMessage(m *claude.SDKResultMessage) {
	fmt.Printf("\n✓ Result: %s\n", m.Subtype)
	if len(m.PermissionDenials) == 0 {
		return
	}

	fmt.Printf(
		"\n🚫 Permission denials: %d\n",
		len(m.PermissionDenials),
	)
	for i, denial := range m.PermissionDenials {
		fmt.Printf(
			"   %d. %s (ID: %s)\n",
			i+1,
			denial.ToolName,
			denial.ToolUseID,
		)
	}
}

// formatToolInput formats tool input for display.
// Handles JSON parsing errors gracefully.
func formatToolInput(input map[string]claude.JSONValue) string {
	if len(input) == 0 {
		return "{}"
	}

	// Try to create a compact representation
	parts := make([]string, 0, len(input))
	for key, value := range input {
		var v any
		if err := json.Unmarshal(value, &v); err != nil {
			// Handle JSON parsing error gracefully
			parts = append(parts, fmt.Sprintf("%s=<invalid>", key))

			continue
		}

		switch val := v.(type) {
		case string:
			// Truncate long strings
			if len(val) > maxValueDisplayLength {
				val = val[:maxValueDisplayLength] + "..."
			}
			parts = append(parts, fmt.Sprintf("%s=%q", key, val))
		default:
			parts = append(parts, fmt.Sprintf("%s=%v", key, val))
		}
	}

	return "{" + strings.Join(parts, ", ") + "}"
}
