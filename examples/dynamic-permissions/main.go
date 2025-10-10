// Package main demonstrates dynamic permission management with the
// Claude Agent SDK.
//
// This example shows how to change permission modes dynamically during
// a conversation using the SetPermissionMode() control method.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/conneroisu/claude-agent-sdk-go/pkg/claude"
	"github.com/conneroisu/claude-agent-sdk-go/pkg/clauderrs"
)

const (
	// maxTurns defines the maximum number of conversation turns.
	maxTurns = 10
	// outputFormat defines the format string for indented output.
	outputFormat = "  %s\n"
)

// Dynamic Permission Management Example.
//
// This example demonstrates how to change permission modes dynamically during
// a conversation using the SetPermissionMode() control method.
//
// Key features demonstrated:
// - Starting with restrictive permissions (plan mode).
// - Monitoring Claude's planned actions.
// - Switching to a more permissive mode after approval.
// - Using SetPermissionMode() to change behavior mid-conversation.
//
// Permission modes:
// - "plan": Claude plans actions without executing them.
// - "default": Normal permission checking.
// - "acceptEdits": Auto-approve file edits.
// - "bypassPermissions": Skip all permission checks (use with caution!).
//
// This pattern is useful for:
// - Staged execution workflows.
// - User approval workflows.
// - Testing and debugging.
// - Progressive trust scenarios.

func main() {
	ctx := context.Background()

	// Start with plan mode - Claude will plan but not execute.
	fmt.Println("=== Dynamic Permission Management Example ===")
	fmt.Println(
		"Starting in PLAN mode - Claude will plan actions" +
			" without executing",
	)

	opts := &claude.Options{
		Model:          "claude-sonnet-4-5",
		MaxTurns:       maxTurns,
		PermissionMode: claude.PermissionModePlan,
	}

	client, err := createClient(opts)
	if err != nil {
		return
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			log.Printf("Failed to close client: %v", closeErr)
		}
	}()

	// Phase 1: Planning phase
	_, err = runPlanningPhase(ctx, client)
	if err != nil {
		return
	}

	// Phase 2: Execution phase
	finalResult, err := runExecutionPhase(
		ctx,
		client,
	)
	if err != nil {
		return
	}

	// Phase 3: Verification phase
	finalResult, err = runVerificationPhase(ctx, client, finalResult)
	if err != nil {
		return
	}

	// Print summary
	printSummary(finalResult)
}

// createClient creates and configures a new Claude Agent client.
func createClient(
	opts *claude.Options,
) (*claude.ClaudeSDKClient, error) {
	client, err := claude.NewClient(opts)
	if err != nil {
		// Handle specific error types
		if clauderrs.IsValidationError(err) {
			log.Printf("Invalid configuration: %v", err)

			return nil, err
		}
		if clauderrs.IsProcessError(err) {
			log.Printf("Failed to start Claude process: %v", err)

			return nil, err
		}
		log.Printf("Failed to create client: %v", err)

		return nil, err
	}

	return client, nil
}

// runPlanningPhase executes the planning phase where Claude plans actions
// without executing them.
func runPlanningPhase(
	ctx context.Context,
	client *claude.ClaudeSDKClient,
) (string, error) {
	fmt.Println("--- Phase 1: Planning Phase ---")
	fmt.Println(
		"Query: Create a file called test.txt with" +
			" 'Hello World' in it",
	)
	fmt.Println()

	query := "Create a file called test.txt with the content" +
		" 'Hello World'. Explain what you would do."
	if err := client.Query(ctx, query); err != nil {
		handleQueryError(err)

		return "", err
	}

	// Collect the plan
	var plan string
	for msg := range client.ReceiveResponse(ctx) {
		assistant, ok := msg.(*claude.SDKAssistantMessage)
		if !ok {
			continue
		}
		for _, block := range assistant.Message.Content {
			switch b := block.(type) {
			case claude.TextBlock:
				plan += b.Text
			case claude.TextContentBlock:
				plan += b.Text
			}
		}
	}

	fmt.Println("Claude's Plan:")
	fmt.Println(plan)
	fmt.Println()

	// Simulate user approval
	fmt.Println("User approves the plan...")
	time.Sleep(1 * time.Second)
	fmt.Println()

	return plan, nil
}

// runExecutionPhase executes the actual file operations after switching
// to accept edits mode.
func runExecutionPhase(
	ctx context.Context,
	client *claude.ClaudeSDKClient,
) (*claude.SDKResultMessage, error) {
	fmt.Println("--- Phase 2: Execution Phase ---")
	fmt.Println(
		"Switching to ACCEPT_EDITS mode to allow file operations",
	)

	// Change permission mode to allow edits
	err := switchToAcceptEditsMode(ctx, client)
	if err != nil {
		return nil, err
	}

	fmt.Println("✓ Permission mode changed to: acceptEdits")
	fmt.Println()

	// Now execute the actual task
	fmt.Println("Query: Now actually create the test.txt file")
	query := "Now actually create the test.txt file with" +
		" 'Hello World'."
	if err = client.Query(ctx, query); err != nil {
		handleQueryError(err)

		return nil, err
	}

	finalResult := processExecutionResponses(
		ctx,
		client,
	)

	return finalResult, nil
}

// switchToAcceptEditsMode changes the permission mode to accept edits.
func switchToAcceptEditsMode(
	ctx context.Context,
	client *claude.ClaudeSDKClient,
) error {
	err := client.SetPermissionMode(
		ctx,
		claude.PermissionModeAcceptEdits,
	)
	if err != nil {
		// Handle control protocol errors
		if clauderrs.IsProtocolError(err) {
			log.Printf(
				"Protocol error changing permission mode: %v",
				err,
			)

			return err
		}
		if clauderrs.IsClientError(err) {
			log.Printf(
				"Client not ready to change permission mode: %v",
				err,
			)

			return err
		}
		log.Printf("Failed to set permission mode: %v", err)

		return err
	}

	return nil
}

// processExecutionResponses processes responses during execution phase.
func processExecutionResponses(
	ctx context.Context,
	client *claude.ClaudeSDKClient,
) *claude.SDKResultMessage {
	var finalResult *claude.SDKResultMessage

	for msg := range client.ReceiveResponse(ctx) {
		switch m := msg.(type) {
		case *claude.SDKAssistantMessage:
			fmt.Println("Claude:")
			for _, block := range m.Message.Content {
				switch b := block.(type) {
				case claude.TextBlock:
					fmt.Printf(outputFormat, b.Text)
				case claude.TextContentBlock:
					fmt.Printf(outputFormat, b.Text)
				case claude.ToolUseContentBlock:
					fmt.Printf("  🔧 Using tool: %s\n", b.Name)
				}
			}
		case *claude.SDKResultMessage:
			finalResult = m
		}
	}

	return finalResult
}

// runVerificationPhase switches to default mode and verifies the file
// was created.
func runVerificationPhase(
	ctx context.Context,
	client *claude.ClaudeSDKClient,
	result *claude.SDKResultMessage,
) (*claude.SDKResultMessage, error) {
	fmt.Println("\n--- Phase 3: Cleanup Phase ---")
	fmt.Println("Switching to DEFAULT mode for verification")

	err := client.SetPermissionMode(
		ctx,
		claude.PermissionModeDefault,
	)
	if err != nil {
		log.Printf("Warning: Failed to set permission mode: %v", err)
		// Continue anyway - not critical
	}

	fmt.Println("✓ Permission mode changed to: default")
	fmt.Println()

	// Verify the file was created
	fmt.Println("Query: Verify the file exists and show its contents")
	query := "Can you verify that test.txt exists and show me" +
		" its contents?"
	err = client.Query(ctx, query)
	if err != nil {
		handleQueryError(err)

		return result, err
	}

	finalResult := result
	for msg := range client.ReceiveResponse(ctx) {
		switch m := msg.(type) {
		case *claude.SDKAssistantMessage:
			fmt.Println("Claude:")
			for _, block := range m.Message.Content {
				switch b := block.(type) {
				case claude.TextBlock:
					fmt.Printf(outputFormat, b.Text)
				case claude.TextContentBlock:
					fmt.Printf(outputFormat, b.Text)
				}
			}
		case *claude.SDKResultMessage:
			finalResult = m
		}
	}

	return finalResult, nil
}

// printSummary prints the final session summary and key takeaways.
func printSummary(
	result *claude.SDKResultMessage,
) {
	// Summary
	if result != nil {
		fmt.Println("\n=== Session Summary ===")
		fmt.Printf("Total turns: %d\n", result.NumTurns)
		fmt.Printf("Duration: %dms\n", result.DurationMS)
		fmt.Printf("Cost: $%.4f\n", result.TotalCostUSD)
		fmt.Println(
			"✓ File operations completed successfully!",
		)
	}

	fmt.Println("\n=== Key Takeaways ===")
	fmt.Println("1. Started in PLAN mode to review actions safely")
	fmt.Println(
		"2. Switched to ACCEPT_EDITS mode after user approval",
	)
	fmt.Println("3. Switched to DEFAULT mode for verification")
	fmt.Println(
		"4. SetPermissionMode() allows dynamic security control",
	)
	fmt.Println(
		"\nThis pattern enables safe, staged execution workflows!",
	)
}

// handleQueryError provides detailed error handling for query operations.
func handleQueryError(err error) {
	if handleAPIError(err) {
		return
	}
	if handleNetworkError(err) {
		return
	}
	if handleClientError(err) {
		return
	}
	if handleProcessError(err) {
		return
	}

	// Generic error handling
	log.Printf("Query failed: %v", err)
}

// handleAPIError handles API-specific errors.
func handleAPIError(err error) bool {
	if !clauderrs.IsAPIError(err) {
		return false
	}

	sdkErr, _ := clauderrs.AsSDKError(err)

	switch sdkErr.Code() {
	case clauderrs.ErrCodeAPIRateLimit:
		handleRateLimitError(err, sdkErr)
	case clauderrs.ErrCodeAPIUnauthorized:
		log.Printf(
			"Invalid API key. Please set ANTHROPIC_API_KEY" +
				" environment variable",
		)
	case clauderrs.ErrCodeAPIServerError:
		log.Printf(
			"API server error. Please retry in a few moments: %v",
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
		log.Printf("API error: %v", err)
	default:
		log.Printf("Unexpected error code: %v", err)
	}

	return true
}

// handleRateLimitError handles rate limit errors with retry info.
func handleRateLimitError(err error, sdkErr clauderrs.SDKError) {
	metadata := sdkErr.Metadata()
	if retryAfter, ok := metadata["retry_after_seconds"]; ok {
		retryFloat, _ := retryAfter.(float64)
		log.Printf(
			"Rate limited. Please wait %v seconds"+
				" before retrying",
			retryFloat,
		)
	} else {
		log.Printf("Rate limited: %v", err)
	}
}

// handleNetworkError handles network-specific errors.
func handleNetworkError(err error) bool {
	if !clauderrs.IsNetworkError(err) {
		return false
	}

	if clauderrs.IsNetworkTimeout(err) {
		log.Printf(
			"Request timed out. Try increasing the timeout"+
				" or simplifying the query: %v",
			err,
		)

		return true
	}
	log.Printf(
		"Network error. Check your internet connection: %v",
		err,
	)

	return true
}

// handleClientError handles client state errors.
func handleClientError(err error) bool {
	if !clauderrs.IsClientError(err) {
		return false
	}

	sdkErr, _ := clauderrs.AsSDKError(err)
	if sdkErr.Code() == clauderrs.ErrCodeClientClosed {
		log.Print("Client is closed. Cannot send query")

		return true
	}
	log.Printf("Client error: %v", err)

	return true
}

// handleProcessError handles process-specific errors.
func handleProcessError(err error) bool {
	if !clauderrs.IsProcessError(err) {
		return false
	}

	var procErr *clauderrs.ProcessError
	if errors.As(err, &procErr) {
		log.Printf(
			"Process error (exit code %d): %s",
			procErr.ExitCode(),
			procErr.Stderr(),
		)

		return true
	}
	log.Printf("Process error: %v", err)

	return true
}
