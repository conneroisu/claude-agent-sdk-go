// Package main demonstrates MCP tool usage with Claude Agent SDK.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/conneroisu/claude-agent-sdk-go/pkg/claude"
	"github.com/conneroisu/claude-agent-sdk-go/pkg/clauderrs"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()

	// Define a custom echo tool.
	echoTool := claude.Tool(
		"echo",
		"Echo back the input text",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text": map[string]any{
					"type":        "string",
					"description": "The text to echo back",
				},
			},
			"required": []string{"text"},
		},
		func(
			_ context.Context,
			args map[string]any,
		) (*claude.McpToolResult, error) {
			text, ok := args["text"].(string)
			if !ok {
				return &claude.McpToolResult{
					Content: []claude.ContentBlock{
						claude.TextContentBlock{
							Type: "text",
							Text: "Error: text parameter must be a string",
						},
					},
					IsError: true,
				}, nil
			}

			return &claude.McpToolResult{
				Content: []claude.ContentBlock{
					claude.TextContentBlock{
						Type: "text",
						Text: fmt.Sprintf("Echo: %s", text),
					},
				},
			}, nil
		},
	)

	// Create an MCP server with the custom tool.
	server := claude.CreateSdkMcpServer(
		"custom-tools",
		"1.0.0",
		[]claude.McpTool{echoTool},
	)

	// Create client with MCP server.
	opts := &claude.Options{
		Model: "claude-sonnet-4-5",
		McpServers: map[string]claude.McpServerConfig{
			"custom-tools": server,
		},
		AllowedTools: []string{"mcp__custom-tools__echo"},
	}

	client, err := claude.NewClient(opts)
	if err != nil {
		return handleClientCreationError(err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			log.Printf("Failed to close client: %v", closeErr)
		}
	}()

	// Send a query that uses the custom tool
	fmt.Println("Sending query: Use the echo tool to echo 'Hello from Go!'")
	err = client.Query(ctx, "Use the echo tool to echo 'Hello from Go!'")
	if err != nil {
		return handleQueryError(err)
	}

	// Receive and process responses
	return receiveMessages(ctx, client)
}

func receiveMessages(
	ctx context.Context,
	client *claude.ClaudeSDKClient,
) error {
	msgChan, errChan := client.ReceiveMessages(ctx)

	for {
		select {
		case msg := <-msgChan:
			if msg == nil {
				fmt.Println("\nQuery completed")

				return nil
			}

			processMessage(msg)

		case err := <-errChan:
			if err != nil {
				return handleStreamError(err)
			}
		}
	}
}

// processMessage handles printing of different message types.
func processMessage(msg any) {
	switch m := msg.(type) {
	case *claude.SDKAssistantMessage:
		printAssistantMessage(m)
	case *claude.SDKResultMessage:
		printResultMessage(m)
	}
}

// printAssistantMessage prints assistant message content.
func printAssistantMessage(m *claude.SDKAssistantMessage) {
	fmt.Println("\nAssistant response:")
	for _, block := range m.Message.Content {
		printContentBlock(block)
	}
}

// printContentBlock prints a single content block.
func printContentBlock(block any) {
	switch b := block.(type) {
	case claude.TextBlock:
		fmt.Printf("  Text: %s\n", b.Text)
	case claude.ToolUseContentBlock:
		fmt.Printf("  Tool use: %s (id: %s)\n", b.Name, b.ID)
		marshaled, err := json.MarshalIndent(b.Input, "    ", "  ")
		if err != nil {
			fmt.Printf("    Input: <error marshaling: %v>\n", err)
		} else {
			fmt.Printf("    Input: %s\n", string(marshaled))
		}
	}
}

// printResultMessage prints result message information.
func printResultMessage(m *claude.SDKResultMessage) {
	fmt.Printf("\nResult: %s\n", m.Subtype)
	fmt.Printf("Cost: $%.4f\n", m.TotalCostUSD)
}

// handleClientCreationError provides detailed error handling for
// client creation.
func handleClientCreationError(err error) error {
	if clauderrs.IsValidationError(err) {
		return fmt.Errorf("invalid client configuration: %w", err)
	}

	if clauderrs.IsProcessError(err) {
		var procErr *clauderrs.ProcessError
		if errors.As(err, &procErr) {
			return fmt.Errorf(
				"failed to start Claude process (exit code %d): %s",
				procErr.ExitCode(),
				procErr.Stderr(),
			)
		}

		return fmt.Errorf("failed to start Claude process: %w", err)
	}

	return fmt.Errorf("failed to create client: %w", err)
}

// handleQueryError provides detailed error handling for
// query operations.
func handleQueryError(err error) error {
	// Check for API errors (rate limits, auth issues)
	if clauderrs.IsAPIError(err) {
		sdkErr, _ := clauderrs.AsSDKError(err)

		switch sdkErr.Code() {
		case clauderrs.ErrCodeAPIRateLimit:
			if retryAfter, ok :=
				sdkErr.Metadata()["retry_after_seconds"].(float64); ok {
				return fmt.Errorf(
					"rate limited. Please wait %v seconds before retrying",
					retryAfter,
				)
			}

			return fmt.Errorf("rate limited: %w", err)

		case clauderrs.ErrCodeAPIUnauthorized:
			return errors.New(
				"authentication failed. Check your ANTHROPIC_API_KEY",
			)

		case clauderrs.ErrCodeAPIServerError:
			return fmt.Errorf("API server error. Please retry later: %w", err)

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

	// Check for network issues
	if clauderrs.IsNetworkError(err) {
		if clauderrs.IsNetworkTimeout(err) {
			return errors.New(
				"request timed out. Try increasing the timeout",
			)
		}

		return fmt.Errorf("network error: %w", err)
	}

	// Check for client state issues
	if clauderrs.IsClientError(err) {
		sdkErr, _ := clauderrs.AsSDKError(err)
		if sdkErr.Code() == clauderrs.ErrCodeClientClosed {
			return errors.New("client is closed")
		}

		return fmt.Errorf("client error: %w", err)
	}

	return fmt.Errorf("query failed: %w", err)
}

// handleStreamError provides error handling during message streaming.
func handleStreamError(err error) error {
	// Protocol errors may indicate version mismatch
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

	// MCP tool execution errors
	if clauderrs.IsCallbackError(err) {
		return fmt.Errorf("MCP tool execution failed: %w", err)
	}

	// Network interruption during streaming
	if clauderrs.IsNetworkError(err) {
		return fmt.Errorf("network error during streaming: %w", err)
	}

	return fmt.Errorf("stream error: %w", err)
}
