package claude

import (
	"context"
	"io"
	"sync"

	"github.com/connerohnesorge/claude-agent-sdk-go/pkg/clauderrs"
)

const (
	// defaultMessageChannelBuffer is the buffer size for message channels.
	defaultMessageChannelBuffer = 10

	// Error messages.
	errNoActiveQuery = "no active query"
)

// ClaudeSDKClient provides a high-level interface to Claude Agent.
type ClaudeSDKClient struct {
	opts   *Options
	query  Query
	mu     sync.Mutex
	closed bool
}

// NewClient creates a new Claude SDK client.
func NewClient(opts *Options) (*ClaudeSDKClient, error) {
	options := opts
	if options == nil {
		options = &Options{}
	}

	return &ClaudeSDKClient{
		opts: options,
	}, nil
}

// Query sends a query to Claude.
func (c *ClaudeSDKClient) Query(ctx context.Context, prompt string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return clauderrs.NewClientError(
			clauderrs.ErrCodeClientClosed,
			"client is closed",
			nil,
		)
	}

	if c.query == nil {
		q, err := QueryFunc(prompt, c.opts)
		if err != nil {
			// Preserve and wrap underlying errors from query
			// creation
			if sdkErr, ok := clauderrs.AsSDKError(err); ok {
				return sdkErr
			}

			return clauderrs.NewClientError(
				clauderrs.ErrCodeInvalidState,
				"failed to create query",
				err,
			)
		}
		c.query = q

		return nil
	}

	// If query already exists, send a user message for multi-turn
	// conversation
	return c.query.SendUserMessage(ctx, prompt)
}

// SendMessage sends a message with structured content blocks to Claude.
//
// This is a convenience method for sending complex messages with images, tool
// results, etc.
func (c *ClaudeSDKClient) SendMessage(
	ctx context.Context,
	content []ContentBlock,
	_sessionID string,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return clauderrs.NewClientError(
			clauderrs.ErrCodeClientClosed,
			"client is closed",
			nil,
		)
	}

	if c.query == nil {
		return clauderrs.NewClientError(
			clauderrs.ErrCodeNoActiveQuery,
			errNoActiveQuery,
			nil,
		)
	}

	return c.query.SendUserMessageWithContent(ctx, content)
}

// ReceiveMessages receives all messages from the current query until EOF.
// This method continues to receive messages even after a ResultMessage,
// useful for monitoring the entire conversation stream.
func (c *ClaudeSDKClient) ReceiveMessages(
	ctx context.Context,
) (<-chan SDKMessage, <-chan error) {
	msgChan := make(chan SDKMessage, defaultMessageChannelBuffer)
	errChan := make(chan error, 1)

	go func() {
		defer close(msgChan)
		defer close(errChan)

		if c.query == nil {
			errChan <- clauderrs.NewClientError(
				clauderrs.ErrCodeNoActiveQuery,
				errNoActiveQuery,
				nil,
			)

			return
		}

		for {
			msg, err := c.query.Next(ctx)
			if err != nil {
				if err != io.EOF {
					errChan <- err
				}

				return
			}

			select {
			case msgChan <- msg:
			case <-ctx.Done():
				errChan <- ctx.Err()

				return
			}
		}
	}()

	return msgChan, errChan
}

// ReceiveResponse receives messages from the current query until a
// ResultMessage.
//
// This is a convenience method for single-response workflows.
//
// The channel automatically closes after receiving a result message.
func (c *ClaudeSDKClient) ReceiveResponse(
	ctx context.Context,
) <-chan SDKMessage {
	msgChan := make(chan SDKMessage, defaultMessageChannelBuffer)

	go func() {
		defer close(msgChan)

		if c.query == nil {
			return
		}

		for {
			msg, err := c.query.Next(ctx)
			if err != nil {
				return
			}

			select {
			case msgChan <- msg:
			case <-ctx.Done():
				return
			}

			// Check if this is a result message (end of query)
			if _, ok := msg.(*SDKResultMessage); ok {
				return
			}
		}
	}()

	return msgChan
}

// Interrupt interrupts the current query.
func (c *ClaudeSDKClient) Interrupt(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.query == nil {
		return clauderrs.NewClientError(
			clauderrs.ErrCodeNoActiveQuery,
			errNoActiveQuery,
			nil,
		)
	}

	return c.query.Interrupt(ctx)
}

// SetPermissionMode changes the permission mode.
func (c *ClaudeSDKClient) SetPermissionMode(
	ctx context.Context,
	mode PermissionMode,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.query == nil {
		return clauderrs.NewClientError(
			clauderrs.ErrCodeNoActiveQuery,
			errNoActiveQuery,
			nil,
		)
	}

	return c.query.SetPermissionMode(ctx, mode)
}

// SetModel changes the model.
func (c *ClaudeSDKClient) SetModel(ctx context.Context, model *string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.query == nil {
		return clauderrs.NewClientError(
			clauderrs.ErrCodeNoActiveQuery,
			errNoActiveQuery,
			nil,
		)
	}

	return c.query.SetModel(ctx, model)
}

// SupportedCommands returns available slash commands.
func (c *ClaudeSDKClient) SupportedCommands(
	ctx context.Context,
) ([]SlashCommand, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.query == nil {
		return nil, clauderrs.NewClientError(
			clauderrs.ErrCodeNoActiveQuery,
			errNoActiveQuery,
			nil,
		)
	}

	return c.query.SupportedCommands(ctx)
}

// SupportedModels returns available models.
func (c *ClaudeSDKClient) SupportedModels(
	ctx context.Context,
) ([]ModelInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.query == nil {
		return nil, clauderrs.NewClientError(
			clauderrs.ErrCodeNoActiveQuery,
			errNoActiveQuery,
			nil,
		)
	}

	return c.query.SupportedModels(ctx)
}

// McpServerStatus returns MCP server status.
func (c *ClaudeSDKClient) McpServerStatus(
	ctx context.Context,
) ([]McpServerStatus, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.query == nil {
		return nil, clauderrs.NewClientError(
			clauderrs.ErrCodeNoActiveQuery,
			errNoActiveQuery,
			nil,
		)
	}

	return c.query.McpServerStatus(ctx)
}

// GetServerInfo returns server information from the query.
func (c *ClaudeSDKClient) GetServerInfo() (map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.query == nil {
		return nil, clauderrs.NewClientError(
			clauderrs.ErrCodeNoActiveQuery,
			errNoActiveQuery,
			nil,
		)
	}

	return c.query.GetServerInfo()
}

// Close closes the client and cleans up resources.
func (c *ClaudeSDKClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	if c.query != nil {
		return c.query.Close()
	}

	return nil
}
