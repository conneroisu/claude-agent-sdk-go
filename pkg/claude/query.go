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
	"github.com/conneroisu/claude/pkg/claude/querying"
)

// Query executes a one-shot query to Claude with automatic lifecycle
// management. It wires up all layers and returns channels for messages
// and errors.
//
// Both channels are closed when the query completes or fails.
// Error channel is buffered to prevent goroutine leaks.
func Query(
	ctx context.Context,
	prompt string,
	opts *options.AgentOptions,
	hooks map[hooking.HookEvent][]hooking.HookMatcher,
) (<-chan messages.Message, <-chan error) {
	queryOpts := opts
	if queryOpts == nil {
		queryOpts = &options.AgentOptions{}
	}

	msgCh, errCh := wireAndExecute(ctx, prompt, queryOpts, hooks)

	return msgCh, errCh
}

func wireAndExecute(
	ctx context.Context,
	prompt string,
	opts *options.AgentOptions,
	hooks map[hooking.HookEvent][]hooking.HookMatcher,
) (<-chan messages.Message, <-chan error) {
	transport := cli.NewAdapter(opts)
	protocol := jsonrpc.NewAdapter(transport)
	parser := parse.NewAdapter()

	hookSvc := hooking.NewService(hooks)
	permSvc := permissions.NewService(nil) // Use default permissions

	svc := querying.NewService(
		transport,
		protocol,
		parser,
		hookSvc,
		permSvc,
		nil,
	)

	return svc.Execute(ctx, prompt)
}
