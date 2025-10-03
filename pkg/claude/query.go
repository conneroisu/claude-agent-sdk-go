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

// Query performs a one-shot query to Claude.
// This is the main entry point that wires up domain services
// with adapters.
func Query(
	ctx context.Context,
	prompt string,
	opts *options.AgentOptions,
	hooks map[hooking.HookEvent][]hooking.HookMatcher,
) (<-chan messages.Message, <-chan error) {
	if opts == nil {
		opts = &options.AgentOptions{} //nolint:revive // modifies-parameter: nil check pattern
	}

	transport := cli.NewAdapter(opts)
	protocol := jsonrpc.NewAdapter(transport)
	parser := parse.NewAdapter()

	var hookingService *hooking.Service
	if hooks != nil {
		hookingService = hooking.NewService(hooks)
	}

	var permissionsService *permissions.Service

	var mcpServers map[string]ports.MCPServer

	queryService := querying.NewService(
		transport,
		protocol,
		parser,
		hookingService,
		permissionsService,
		mcpServers,
	)

	return queryService.Execute(ctx, prompt, opts)
}
