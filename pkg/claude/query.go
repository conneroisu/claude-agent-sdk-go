package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/conneroisu/claude-agent-sdk-go/internal/transport"
	"github.com/conneroisu/claude-agent-sdk-go/pkg/clauderrs"
	"github.com/google/uuid"
)

const (
	// Message channel and control request buffer sizes.
	msgChanBufferSize        = 100
	controlRequestChanBuffer = 10

	// Control protocol message types and subtypes.
	messageTypeUser            = "user"
	messageTypeControlRequest  = "control_request"
	messageTypeControlResponse = "control_response"
	messageTypeHookCallback    = "hook_callback"

	// Request ID format.
	requestIDFormat = "req_%d_%s"

	// JSON field names.
	fieldType      = "type"
	fieldUUID      = "uuid"
	fieldSessionID = "session_id"
	fieldRequestID = "request_id"
	fieldRequest   = "request"
	fieldSubtype   = "subtype"
)

// Query represents an active query session.
//
// Design Note: The TypeScript SDK's Query interface extends AsyncGenerator,
// allowing for iteration and control methods to be called during iteration.
// In Go, we achieve similar functionality through the Query interface methods,
// though the pattern differs from TypeScript's async generator approach.
// The Next() method provides sequential message access similar to TypeScript's
// for-await-of loop, while control methods (Interrupt, SetModel, etc.) can be
// called at any time during iteration.
//
// Key differences from TypeScript:
//   - TypeScript: Uses AsyncGenerator with yield for messages
//   - Go: Uses explicit Next() method with error return for
//     idiomatic error handling
//   - TypeScript: Control methods can be called on the generator object
//     during iteration
//   - Go: Same capability via Query interface methods, but more explicit
//
// Both approaches provide equivalent functionality with idiomatic
// patterns for each language.
type Query interface {
	// Next returns the next message from the query stream.
	Next(ctx context.Context) (SDKMessage, error)
	// Close closes the query and cleans up resources.
	Close() error

	// SendUserMessage sends a text user message to the process.
	SendUserMessage(ctx context.Context, text string) error
	// SendUserMessageWithContent sends a user message with structured
	// content blocks.
	SendUserMessageWithContent(ctx context.Context, content []ContentBlock) error

	// Interrupt interrupts the current query.
	Interrupt(ctx context.Context) error
	// SetPermissionMode changes the permission mode.
	SetPermissionMode(ctx context.Context, mode PermissionMode) error
	// SetModel changes the model.
	SetModel(ctx context.Context, model *string) error
	// SupportedCommands returns available slash commands.
	SupportedCommands(ctx context.Context) ([]SlashCommand, error)
	// SupportedModels returns available models.
	SupportedModels(ctx context.Context) ([]ModelInfo, error)
	// McpServerStatus returns MCP server status.
	McpServerStatus(ctx context.Context) ([]McpServerStatus, error)
	// GetServerInfo returns the initialization result stored during Initialize.
	GetServerInfo() (map[string]any, error)
}

// queryImpl implements the Query interface.
type queryImpl struct {
	proc                    *transport.Process
	msgChan                 chan SDKMessage
	errChan                 chan error
	closeChan               chan struct{}
	opts                    *Options
	sessionID               string
	mu                      sync.Mutex
	closed                  bool
	requestCounter          int
	pendingControlResponses map[string]chan *SDKControlResponse
	initializationResult    map[string]any
	hookCallbacks           map[string]HookCallback // Maps callback IDs to hook
	// functions
	nextCallbackID int // Counter for generating
	// callback IDs
	controlRequestChan chan json.RawMessage // Channel for incoming
	// control requests
}

// newQueryImpl creates a new query implementation.
func newQueryImpl(prompt string, opts *Options) (*queryImpl, error) {
	if opts == nil {
		opts = &Options{}
	}

	q := &queryImpl{
		msgChan:                 make(chan SDKMessage, msgChanBufferSize),
		errChan:                 make(chan error, 1),
		closeChan:               make(chan struct{}),
		opts:                    opts,
		sessionID:               uuid.New().String(),
		pendingControlResponses: make(map[string]chan *SDKControlResponse),
		hookCallbacks:           make(map[string]HookCallback),
		nextCallbackID:          0,
		controlRequestChan: make(chan json.RawMessage,
			controlRequestChanBuffer),
	}

	// Start the process
	if err := q.start(prompt); err != nil {
		return nil, err
	}

	return q, nil
}

// start initializes the process and message handling.
func (q *queryImpl) start(prompt string) error {
	// Build process args
	args := q.buildArgs()

	// Build environment
	env := q.buildEnv()

	// Create process config
	config := &transport.ProcessConfig{
		Executable:    q.opts.PathToClaudeCodeExecutable,
		Args:          args,
		Env:           env,
		Cwd:           q.opts.Cwd,
		StderrHandler: q.opts.Stderr,
	}

	// Start process
	proc, err := transport.NewProcess(context.Background(), config)
	if err != nil {
		return clauderrs.CreateProcessError(
			clauderrs.ErrCodeProcessSpawnFailed,
			"failed to start Claude Code process",
			err,
			0,
			"",
		).
			WithCommand(fmt.Sprintf("%s %v",
				q.opts.PathToClaudeCodeExecutable, args)).
			WithSessionID(q.sessionID)
	}
	q.proc = proc

	// Start message reading goroutine
	go q.readMessages()

	// Start control request handler goroutine
	go q.handleControlRequests()

	// Send initial prompt
	if prompt != "" {
		if err := q.SendUserMessage(context.Background(), prompt); err != nil {
			_ = q.Close()

			return clauderrs.NewProtocolError(clauderrs.ErrCodeProtocolError,
				"failed to send initial prompt", err).
				WithSessionID(q.sessionID).
				WithMessageType("user")
		}
	}

	return nil
}

// buildArgs builds the command line arguments for the process.
func (q *queryImpl) buildArgs() []string {
	// Start with required flags for stream-json protocol
	args := []string{
		"--print",
		"--output-format=stream-json",
		"--input-format=stream-json",
		"--verbose",
	}

	if q.opts.Model != "" {
		args = append(args, "--model", q.opts.Model)
	}

	if q.opts.Continue {
		args = append(args, "--continue")
	}

	if q.opts.Resume != "" {
		args = append(args, "--resume", q.opts.Resume)
	}

	if q.opts.PermissionMode != "" {
		args = append(args, "--permission-mode", string(q.opts.PermissionMode))
	}

	// Add additional directories
	for _, dir := range q.opts.AdditionalDirectories {
		args = append(args, "--add-dir", dir)
	}

	// Add allowed tools
	for _, tool := range q.opts.AllowedTools {
		args = append(args, "--allowed-tools", tool)
	}

	// Add disallowed tools
	for _, tool := range q.opts.DisallowedTools {
		args = append(args, "--disallowed-tools", tool)
	}

	// Add include partial messages flag for streaming
	if q.opts.IncludePartialMessages {
		args = append(args, "--include-partial-messages")
	}

	return args
}

// buildEnv builds the environment variables for the process.
func (q *queryImpl) buildEnv() []string {
	env := make([]string, 0)

	for key, value := range q.opts.Env {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}
