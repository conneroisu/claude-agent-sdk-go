package claude

import (
	"context"
	"fmt"
	"github.com/conneroisu/claude/pkg/claude/adapters/cli"
	"github.com/conneroisu/claude/pkg/claude/adapters/jsonrpc"
	"github.com/conneroisu/claude/pkg/claude/adapters/parse"
	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/querying"
)

// Query performs a one-shot query to Claude and returns channels for receiving
// messages and errors. This is the simplest way to interact with Claude for
// single queries that don't require maintaining conversation state.
//
// The message channel receives all messages from Claude including assistant
// responses and result messages. The error channel receives any errors that
// occur. Both channels will be closed when the query completes.
//
// Example:
//
//	msgCh, errCh := claude.Query(ctx, "What is 2+2?", nil, nil)
//	for msg := range msgCh {
//	    switch m := msg.(type) {
//	    case *messages.AssistantMessage:
//	        // Handle response
//	    }
//	}
func Query(ctx context.Context, prompt string, opts *options.AgentOptions, hooks map[hooking.HookEvent][]hooking.HookMatcher) (<-chan messages.Message, <-chan error) {
	if opts == nil {
		opts = &options.AgentOptions{}
	}
	// Wire up adapters (infrastructure layer)
	transport := cli.NewAdapter(opts)
	protocol := jsonrpc.NewAdapter(transport)
	parser := parse.NewAdapter()
	// Create domain services
	var hookingService *hooking.Service
	if hooks != nil {
		hookingService = hooking.NewService(hooks)
	}
	// Create permissions service from options
	var permissionsService *permissions.Service
	if opts.PermissionMode != nil {
		// TODO: Initialize permissions service based on permission mode
		// permissionsService = permissions.NewService(...)
	}
	// Initialize MCP servers from configuration
	mcpServers, err := initializeMCPServers(ctx, opts.MCPServers)
	if err != nil {
		msgCh := make(chan messages.Message)
		errCh := make(chan error, 1)
		errCh <- fmt.Errorf("failed to initialize MCP servers: %w", err)
		close(msgCh)
		close(errCh)

		return msgCh, errCh
	}
	queryService := querying.NewService(transport, protocol, parser, hookingService, permissionsService, mcpServers)
	// Execute domain logic
	return queryService.Execute(ctx, prompt, opts)
}
