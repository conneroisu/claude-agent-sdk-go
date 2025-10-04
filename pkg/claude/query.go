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

// Query performs a one-shot query to Claude.
// This is the main entry point for simple, non-streaming interactions.
// It wires up adapters and domain services, executes the query,
// and returns channels for messages and errors.
//
// The function returns immediately with channels that will be populated
// as the query executes. Both channels are closed when the query completes.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - prompt: The user message to send to Claude
//   - opts: Agent configuration options (nil uses defaults)
//   - hooks: Hook callbacks for interception (nil disables hooks)
//
// Returns:
//   - msgCh: Channel that receives parsed messages from Claude
//   - errCh: Channel that receives any errors during execution
//
// Example:
//
//	msgCh, errCh := claude.Query(
//		context.Background(),
//		"What is the capital of France?",
//		nil,
//		nil,
//	)
//
//	for msg := range msgCh {
//		fmt.Printf("Message: %+v\n", msg)
//	}
//	if err := <-errCh; err != nil {
//		log.Fatal(err)
//	}
func Query(
	ctx context.Context,
	prompt string,
	opts *options.AgentOptions,
	hooks map[hooking.HookEvent][]hooking.HookMatcher,
) (<-chan messages.Message, <-chan error) {
	if opts == nil {
		opts = &options.AgentOptions{}
	}

	// Wire up infrastructure adapters
	transport := cli.NewAdapter(opts)
	protocol := jsonrpc.NewAdapter(transport)
	parser := parse.NewAdapter()

	// Wire up domain services
	var hookingService *hooking.Service
	if hooks != nil {
		hookingService = hooking.NewService(hooks)
	}

	var permissionsService *permissions.Service
	if opts.PermissionMode != nil {
		permissionsService = permissions.NewService(&permissions.PermissionsConfig{
			Mode: *opts.PermissionMode,
		})
	}

	// Initialize MCP servers from configuration
	mcpServers, err := initializeMCPServers(ctx, opts.MCPServers)
	if err != nil {
		return createErrorChannels(
			fmt.Errorf("failed to initialize MCP servers: %w", err),
		)
	}

	// Create query service with wired dependencies
	queryService := querying.NewService(
		transport,
		protocol,
		parser,
		hookingService,
		permissionsService,
		mcpServers,
	)

	// Execute the query and return channels
	return queryService.Execute(ctx, prompt, opts)
}
