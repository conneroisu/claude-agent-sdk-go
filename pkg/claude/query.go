package claude

import (
	"context"

	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/querying"
)

// Query is a convenience function for one-shot queries.
// It creates a client, connects, sends a prompt, and returns results.
func Query(
	ctx context.Context,
	prompt string,
	opts *options.AgentOptions,
	hooks map[HookEvent][]HookMatcher,
) (<-chan messages.Message, <-chan error) {
	effectiveOpts := opts
	if effectiveOpts == nil {
		effectiveOpts = &options.AgentOptions{}
	}

	client, err := NewClient(effectiveOpts)
	if err != nil {
		errCh := make(chan error, 1)
		errCh <- err
		close(errCh)

		return nil, errCh
	}

	if err := client.Connect(ctx, hooks); err != nil {
		errCh := make(chan error, 1)
		errCh <- err
		close(errCh)

		return nil, errCh
	}

	return client.Query(ctx, prompt)
}

// Query executes a one-shot query to Claude.
// This is for simple request-response interactions.
func (c *Client) Query(
	ctx context.Context,
	prompt string,
) (<-chan messages.Message, <-chan error) {
	cfg := &querying.Config{
		Transport:   c.transport,
		Protocol:    c.protocol,
		Parser:      c.parser,
		Hooks:       c.hooks,
		Permissions: c.permissions,
		MCPServers:  c.mcpServers,
	}

	svc := querying.NewService(cfg)

	return svc.Execute(ctx, prompt, c.options)
}
