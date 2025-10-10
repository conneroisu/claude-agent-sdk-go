// Package main demonstrates MCP tool usage with Claude Agent SDK.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/conneroisu/claude-agent-sdk-go/pkg/claude"
	"github.com/conneroisu/claude-agent-sdk-go/pkg/clauderrs"
)

func main() {
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
		func(ctx context.Context, args map[string]any) (*claude.McpToolResult, error) {
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
		handleClientCreationError(err)

		return
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
		handleQueryError(err)

		return
	}

	// Receive and process responses
	msgChan, errChan := client.ReceiveMessages(ctx)

	for {
		select {
		case msg := <-msgChan:
			if msg == nil {
				fmt.Println("\nQuery completed")

				return
			}

			switch m := msg.(type) {
			case *claude.SDKAssistantMessage:
				fmt.Println("\nAssistant response:")
				for _, block := range m.Message.Content {
					switch b := block.(type) {
					case claude.TextBlock:
						fmt.Printf("  Text: %s\n", b.Text)
					case claude.ToolUseContentBlock:
						fmt.Printf("  Tool use: %s (id: %s)\n", b.Name, b.ID)
						fmt.Printf("    Input: %s\n", string(b.Input))
					}
				}

			case *claude.SDKResultMessage:
				fmt.Printf("\nResult: %s\n", m.Subtype)
				fmt.Printf("Cost: $%.4f\n", m.TotalCostUSD)
			}

		case err := <-errChan:
			if err != nil {
				handleStreamError(err)

				return
			}
		}
	}
}

// handleClientCreationError provides detailed error handling for client creation.
func handleClientCreationError(err error) {
	if clauderrs.IsValidationError(err) {
		log.Fatalf("Invalid client configuration: %v", err)
	}

	if clauderrs.IsProcessError(err) {
		var procErr *clauderrs.ProcessError
		if errors.As(err, &procErr) {
			log.Fatalf("Failed to start Claude process (exit code %d): %s",
				procErr.ExitCode(), procErr.Stderr())
		}
		log.Fatalf("Failed to start Claude process: %v", err)
	}

	log.Fatalf("Failed to create client: %v", err)
}

// handleQueryError provides detailed error handling for query operations.
func handleQueryError(err error) {
	// Check for API errors (rate limits, auth issues)
	if clauderrs.IsAPIError(err) {
		sdkErr, _ := clauderrs.AsSDKError(err)

		switch sdkErr.Code() {
		case clauderrs.ErrCodeAPIRateLimit:
			if retryAfter, ok := sdkErr.Metadata()["retry_after_seconds"].(float64); ok {
				log.Fatalf("Rate limited. Please wait %v seconds before retrying", retryAfter)
			}
			log.Fatalf("Rate limited: %v", err)

		case clauderrs.ErrCodeAPIUnauthorized:
			log.Fatalf("Authentication failed. Check your ANTHROPIC_API_KEY")

		case clauderrs.ErrCodeAPIServerError:
			log.Fatalf("API server error. Please retry later: %v", err)

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
			log.Fatalf("API error: %v", err)

		default:
			log.Fatalf("Unexpected error code: %v", err)
		}

		return
	}

	// Check for network issues
	if clauderrs.IsNetworkError(err) {
		if clauderrs.IsNetworkTimeout(err) {
			log.Fatalf("Request timed out. Try increasing the timeout")
		}
		log.Fatalf("Network error: %v", err)
	}

	// Check for client state issues
	if clauderrs.IsClientError(err) {
		sdkErr, _ := clauderrs.AsSDKError(err)
		if sdkErr.Code() == clauderrs.ErrCodeClientClosed {
			log.Fatalf("Client is closed")
		}
		log.Fatalf("Client error: %v", err)
	}

	log.Fatalf("Query failed: %v", err)
}

// handleStreamError provides error handling during message streaming.
func handleStreamError(err error) {
	// Protocol errors may indicate version mismatch
	if clauderrs.IsProtocolError(err) {
		sdkErr, _ := clauderrs.AsSDKError(err)
		if sdkErr.Code() == clauderrs.ErrCodeMessageParseFailed {
			log.Fatalf("Failed to parse message. This may indicate a version mismatch: %v", err)
		}
		log.Fatalf("Protocol error: %v", err)
	}

	// MCP tool execution errors
	if clauderrs.IsCallbackError(err) {
		log.Fatalf("MCP tool execution failed: %v", err)
	}

	// Network interruption during streaming
	if clauderrs.IsNetworkError(err) {
		log.Fatalf("Network error during streaming: %v", err)
	}

	log.Fatalf("Stream error: %v", err)
}
