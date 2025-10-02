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

// QueryConfig holds configuration for one-shot queries
type QueryConfig struct {
	Hooks       map[HookEvent][]HookMatcher
	Permissions *PermissionsConfig
}

// Query performs a one-shot query to Claude
// This is the main entry point that wires up domain services with adapters
func Query(
	ctx context.Context,
	prompt string,
	opts *options.AgentOptions,
	config *QueryConfig,
) (<-chan messages.Message, <-chan error) {
	// Create local copy to avoid modifying parameter
	localOpts := opts
	if localOpts == nil {
		localOpts = &options.AgentOptions{}
	} else {
		// Make a copy to avoid modifying the original
		optsCopy := *opts
		localOpts = &optsCopy
	}

	// Mark as non-streaming for one-shot query behavior
	localOpts.IsStreaming = false

	// Wire up adapters (infrastructure layer)
	transport := cli.NewAdapter(localOpts)
	protocol := jsonrpc.NewAdapter(transport)
	parser := parse.NewAdapter()

	// Create domain services from config
	deps := createDependencies(config, transport, protocol, parser)

	// Create query service with all dependencies
	queryService := querying.NewService(deps)

	// Execute domain logic
	return queryService.Execute(ctx, prompt, localOpts)
}

// createDependencies initializes domain services from configuration
func createDependencies(
	config *QueryConfig,
	transport ports.Transport,
	protocol ports.ProtocolHandler,
	parser ports.MessageParser,
) querying.Dependencies {
	return querying.Dependencies{
		Transport:   transport,
		Protocol:    protocol,
		Parser:      parser,
		Hooks:       createHookingService(config),
		Permissions: createPermissionsService(config),
		MCPServers:  nil, // TODO: Initialize MCP servers from options
	}
}

// createHookingService creates hooking service if configured
func createHookingService(config *QueryConfig) *hooking.Service {
	if config != nil && config.Hooks != nil {
		return hooking.NewService(config.Hooks)
	}

	return nil
}

// createPermissionsService creates permissions service if configured
func createPermissionsService(config *QueryConfig) *permissions.Service {
	if config != nil && config.Permissions != nil {
		return permissions.NewService(config.Permissions)
	}

	return nil
}
