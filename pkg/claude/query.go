package claude

import (
	"context"

	"github.com/conneroisu/claude/pkg/claude/adapters/cli"
	"github.com/conneroisu/claude/pkg/claude/adapters/jsonrpc"
	"github.com/conneroisu/claude/pkg/claude/adapters/parse"
	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
	"github.com/conneroisu/claude/pkg/claude/querying"
)

// Query performs a one-shot query to Claude
func Query(
	ctx context.Context,
	prompt string,
	opts *options.AgentOptions,
	hooks map[hooking.HookEvent][]hooking.HookMatcher,
	perms *permissions.PermissionsConfig,
) (<-chan messages.Message, <-chan error) {
	if opts == nil {
		opts = &options.AgentOptions{}
	}

	// Mark as non-streaming (one-shot query)
	opts.IsStreaming = false

	// Wire up adapters (infrastructure layer)
	transport := cli.NewAdapter(opts)
	protocol := jsonrpc.NewAdapter(transport)
	parser := parse.NewAdapter()

	// Create domain services
	var hookingService *hooking.Service
	if hooks != nil {
		hookingService = hooking.NewService(hooks)
	}

	var permissionsService *permissions.Service
	if perms != nil {
		permissionsService = permissions.NewService(perms)
	}

	// MCP servers (if any)
	var mcpServers map[string]ports.MCPServer
	// TODO: Initialize MCP servers from opts.MCPServers

	// Create querying service with dependencies
	queryService := querying.NewService(
		transport,
		protocol,
		parser,
		hookingService,
		permissionsService,
		mcpServers,
	)

	// Execute domain logic
	return queryService.Execute(ctx, prompt, opts)
}
