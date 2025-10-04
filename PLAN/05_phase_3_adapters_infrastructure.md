## Phase 3: Adapters (Infrastructure)
### 3.1 CLI Transport Adapter (adapters/cli/transport.go)
Priority: Critical
This adapter implements the Transport port using subprocess CLI.
```go
package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Adapter implements ports.Transport using CLI subprocess
type Adapter struct {
	options              *options.AgentOptions
	cliPath              string
	cmd                  *exec.Cmd
	stdin                io.WriteCloser
	stdout               io.ReadCloser
	stderr               io.ReadCloser
	ready                bool
	exitErr              error
	closeStdinAfterWrite bool // For one-shot queries
	mu                   sync.RWMutex
	maxBufferSize        int
}

// Verify interface compliance at compile time
var _ ports.Transport = (*Adapter)(nil)

const defaultMaxBufferSize = 1024 * 1024 // 1MB

func NewAdapter(opts *options.AgentOptions) *Adapter {
	maxBuf := defaultMaxBufferSize
	if opts.MaxBufferSize != nil {
		maxBuf = *opts.MaxBufferSize
	}
	return &Adapter{
		options:       opts,
		maxBufferSize: maxBuf,
	}
}

// findCLI locates the Claude CLI binary
func (a *Adapter) findCLI() (string, error) {
	// Check PATH first
	if path, err := exec.LookPath("claude"); err == nil {
		return path, nil
	}
	// Check common installation locations
	homeDir, _ := os.UserHomeDir()
	locations := []string{
		filepath.Join(homeDir, ".npm-global", "bin", "claude"),
		"/usr/local/bin/claude",
		filepath.Join(homeDir, ".local", "bin", "claude"),
		filepath.Join(homeDir, "node_modules", ".bin", "claude"),
		filepath.Join(homeDir, ".yarn", "bin", "claude"),
	}
	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc, nil
		}
	}
	return "", fmt.Errorf("claude CLI not found in PATH or common locations")
}

// BuildCommand constructs the CLI command with all options
// Exported for testing purposes
func (a *Adapter) BuildCommand() ([]string, error) {
	cmd := []string{a.cliPath, "--output-format", "stream-json", "--verbose"}
	// System prompt
	if a.options.SystemPrompt != nil {
		switch sp := a.options.SystemPrompt.(type) {
		case options.StringSystemPrompt:
			cmd = append(cmd, "--system-prompt", string(sp))
		case options.PresetSystemPrompt:
			if sp.Append != nil {
				cmd = append(cmd, "--append-system-prompt", *sp.Append)
			}
		}
	}
	// Tools
	if len(a.options.AllowedTools) > 0 {
		cmd = append(cmd, "--allowedTools", strings.Join(a.options.AllowedTools, ","))
	}
	if len(a.options.DisallowedTools) > 0 {
		cmd = append(cmd, "--disallowedTools", strings.Join(a.options.DisallowedTools, ","))
	}
	// Model and turns
	if a.options.Model != nil {
		cmd = append(cmd, "--model", *a.options.Model)
	}
	if a.options.MaxTurns != nil {
		cmd = append(cmd, "--max-turns", fmt.Sprintf("%d", *a.options.MaxTurns))
	}
	// Permissions
	if a.options.PermissionMode != nil {
		cmd = append(cmd, "--permission-mode", string(*a.options.PermissionMode))
	}
	if a.options.PermissionPromptToolName != nil {
		cmd = append(cmd, "--permission-prompt-tool", *a.options.PermissionPromptToolName)
	}
	// Session
	if a.options.ContinueConversation {
		cmd = append(cmd, "--continue")
	}
	if a.options.Resume != nil {
		cmd = append(cmd, "--resume", *a.options.Resume)
	}
	if a.options.ForkSession {
		cmd = append(cmd, "--fork-session")
	}
	// Settings
	if a.options.Settings != nil {
		cmd = append(cmd, "--settings", *a.options.Settings)
	}
	if len(a.options.SettingSources) > 0 {
		sources := make([]string, len(a.options.SettingSources))
		for i, s := range a.options.SettingSources {
			sources[i] = string(s)
		}
		cmd = append(cmd, "--setting-sources", strings.Join(sources, ","))
	}
	// Directories
	for _, dir := range a.options.AddDirs {
		cmd = append(cmd, "--add-dir", dir)
	}
	// MCP servers (configuration only, instances handled separately)
	if len(a.options.MCPServers) > 0 {
		// Convert to JSON config
		mcpConfig := map[string]any{"mcpServers": a.options.MCPServers}
		jsonBytes, err := json.Marshal(mcpConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal MCP config: %w", err)
		}
		cmd = append(cmd, "--mcp-config", string(jsonBytes))
	}
	// Extra arguments
	for flag, value := range a.options.ExtraArgs {
		if value == nil {
			cmd = append(cmd, "--"+flag)
		} else {
			cmd = append(cmd, "--"+flag, *value)
		}
	}
	return cmd, nil
}
func (a *Adapter) Connect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.ready {
		return nil
	}
	// Find CLI
	cliPath, err := a.findCLI()
	if err != nil {
		return fmt.Errorf("CLI discovery failed: %w", err)
	}
	a.cliPath = cliPath
	// Build command
	cmdArgs, err := a.BuildCommand()
	if err != nil {
		return fmt.Errorf("command construction failed: %w", err)
	}
	// Set up environment
	env := os.Environ()
	env = append(env, "CLAUDE_CODE_ENTRYPOINT=sdk-go")
	for k, v := range a.options.Env {
		env = append(env, k+"="+v)
	}
	// Create command
	a.cmd = exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	a.cmd.Env = env
	if a.options.Cwd != nil {
		a.cmd.Dir = *a.options.Cwd
	}
	// Set up pipes
	stdin, err := a.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe failed: %w", err)
	}
	a.stdin = stdin
	stdout, err := a.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe failed: %w", err)
	}
	a.stdout = stdout
	stderr, err := a.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe failed: %w", err)
	}
	a.stderr = stderr
	// Start process
	if err := a.cmd.Start(); err != nil {
		return fmt.Errorf("process start failed: %w", err)
	}
	// Start stderr handler if callback is set
	if a.options.StderrCallback != nil {
		go a.handleStderr()
	}
	// Note: One-shot vs streaming mode is determined by the domain service
	// The closeStdinAfterWrite flag is managed internally by the adapter
	// and set via Write() method behavior, not through options
	a.ready = true
	return nil
}
func (a *Adapter) handleStderr() {
	scanner := bufio.NewScanner(a.stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if a.options.StderrCallback != nil {
			a.options.StderrCallback(line)
		}
	}
}

func (a *Adapter) Write(ctx context.Context, data string) error {
	a.mu.RLock()
	shouldClose := a.closeStdinAfterWrite
	a.mu.RUnlock()
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.ready {
		return fmt.Errorf("transport not ready")
	}
	if a.exitErr != nil {
		return fmt.Errorf("transport has exited: %w", a.exitErr)
	}
	_, err := a.stdin.Write([]byte(data))
	if err != nil {
		return err
	}
	// Close stdin after write for one-shot queries
	if shouldClose {
		a.closeStdinAfterWrite = false
		a.stdin.Close()
	}
	return nil
}
func (a *Adapter) ReadMessages(ctx context.Context) (<-chan map[string]any, <-chan error) {
	msgCh := make(chan map[string]any, 10)
	errCh := make(chan error, 1)
	go func() {
		defer close(msgCh)
		defer close(errCh)
		scanner := bufio.NewScanner(a.stdout)
		// Configure scanner buffer to handle large Claude responses
		// Default is 64KB which is insufficient for large responses
		scanBuf := make([]byte, 64*1024)
		scanner.Buffer(scanBuf, a.maxBufferSize)
		buffer := ""
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}
			line := scanner.Text()
			buffer += line
			// Check buffer size
			if len(buffer) > a.maxBufferSize {
				errCh <- fmt.Errorf("message buffer exceeded %d bytes", a.maxBufferSize)
				return
			}
			// Try to parse JSON
			var msg map[string]any
			if err := json.Unmarshal([]byte(buffer), &msg); err == nil {
				buffer = ""
				msgCh <- msg
			}
			// Continue buffering if incomplete
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
		}
		// Check exit status
		if a.cmd != nil {
			if err := a.cmd.Wait(); err != nil {
				errCh <- fmt.Errorf("process exited with error: %w", err)
			}
		}
	}()
	return msgCh, errCh
}
func (a *Adapter) EndInput() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stdin != nil {
		return a.stdin.Close()
	}
	return nil
}

func (a *Adapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ready = false
	// Close stdin
	if a.stdin != nil {
		a.stdin.Close()
	}
	// Terminate process
	if a.cmd != nil && a.cmd.Process != nil {
		a.cmd.Process.Kill()
		a.cmd.Wait()
	}
	return nil
}

func (a *Adapter) IsReady() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.ready
}
```
### 3.2 JSON-RPC Protocol Adapter (adapters/jsonrpc/protocol.go)
Priority: High
Key Design: This adapter implements `ports.ProtocolHandler` and manages all control protocol state (pending requests, request IDs, etc.). The domain services delegate this infrastructure concern to the adapter.
```go
package jsonrpc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/conneroisu/claude/pkg/claude/hooking"
	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/permissions"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Adapter implements ports.ProtocolHandler for control protocol
// This is an INFRASTRUCTURE adapter - it handles protocol state management
type Adapter struct {
	transport ports.Transport
	// Control protocol state (managed by adapter, not domain)
	pendingReqs    map[string]chan result
	requestCounter int
	mu             sync.Mutex
}

// Verify interface compliance at compile time
var _ ports.ProtocolHandler = (*Adapter)(nil)

type result struct {
	data map[string]any
	err  error
}

func NewAdapter(transport ports.Transport) *Adapter {
	return &Adapter{
		transport:   transport,
		pendingReqs: make(map[string]chan result),
	}
}

// Initialize is a no-op - initialization happens implicitly in StartMessageRouter
func (a *Adapter) Initialize(ctx context.Context, config any) (map[string]any, error) {
	return nil, nil
}

// SendControlRequest sends a control request and waits for response
// This method handles all request ID generation and timeout logic
func (a *Adapter) SendControlRequest(ctx context.Context, req map[string]any) (map[string]any, error) {
	// Generate unique request ID: req_{counter}_{randomHex}
	a.mu.Lock()
	a.requestCounter++
	requestID := fmt.Sprintf("req_%d_%s", a.requestCounter, randomHex(4))
	a.mu.Unlock()
	// Create result channel for this request
	resCh := make(chan result, 1)
	a.mu.Lock()
	a.pendingReqs[requestID] = resCh
	a.mu.Unlock()
	// Build control request envelope
	controlReq := map[string]any{
		"type":       "control_request",
		"request_id": requestID,
		"request":    req,
	}
	// Send via transport
	reqBytes, err := json.Marshal(controlReq)
	if err != nil {
		a.mu.Lock()
		delete(a.pendingReqs, requestID)
		a.mu.Unlock()
		return nil, fmt.Errorf("marshal control request: %w", err)
	}
	if err := a.transport.Write(ctx, string(reqBytes)+"\n"); err != nil {
		a.mu.Lock()
		delete(a.pendingReqs, requestID)
		a.mu.Unlock()
		return nil, fmt.Errorf("write control request: %w", err)
	}
	// Wait for response with 60s timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	select {
	case <-timeoutCtx.Done():
		a.mu.Lock()
		delete(a.pendingReqs, requestID)
		a.mu.Unlock()
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("control request timeout: %s", req["subtype"])
		}
		return nil, timeoutCtx.Err()
	case res := <-resCh:
		if res.err != nil {
			return nil, res.err
		}
		return res.data, nil
	}
}

// HandleControlRequest routes inbound control requests by subtype
func (a *Adapter) HandleControlRequest(
	ctx context.Context,
	req map[string]any,
	perms *permissions.Service,
	hooks map[string]hooking.HookCallback,
	mcpServers map[string]ports.MCPServer,
) (map[string]any, error) {
	request, _ := req["request"].(map[string]any)
	subtype, _ := request["subtype"].(string)
	switch subtype {
	case "can_use_tool":
		return a.handleCanUseTool(ctx, request, perms)
	case "hook_callback":
		return a.handleHookCallback(ctx, request, hooks)
	case "mcp_message":
		return a.handleMCPMessage(ctx, request, mcpServers)
	default:
		return nil, fmt.Errorf("unsupported control request subtype: %s", subtype)
	}
}

// StartMessageRouter continuously reads transport and partitions messages
func (a *Adapter) StartMessageRouter(
	ctx context.Context,
	msgCh chan<- map[string]any,
	errCh chan<- error,
	perms *permissions.Service,
	hooks map[string]hooking.HookCallback,
	mcpServers map[string]ports.MCPServer,
) error {
	go func() {
		transportMsgCh, transportErrCh := a.transport.ReadMessages(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-transportMsgCh:
				if !ok {
					return
				}
				msgType, _ := msg["type"].(string)
				switch msgType {
				case "control_response":
					// Route to pending request
					a.routeControlResponse(msg)
				case "control_request":
					// Handle inbound control request
					go a.handleControlRequestAsync(ctx, msg, perms, hooks, mcpServers)
				case "control_cancel_request":
					// Cancel a pending control request
					requestID, _ := msg["request_id"].(string)
					a.mu.Lock()
					if ch, exists := a.pendingReqs[requestID]; exists {
						// Send error to indicate cancellation
						select {
						case ch <- result{err: fmt.Errorf("request cancelled")}:
						default:
						}
						close(ch)
						delete(a.pendingReqs, requestID)
					}
					a.mu.Unlock()
					continue
				default:
					// Forward SDK messages to public stream
					select {
					case msgCh <- msg:
					case <-ctx.Done():
						return
					}
				}
			case err := <-transportErrCh:
				if err != nil {
					select {
					case errCh <- err:
					case <-ctx.Done():
					}
					return
				}
			}
		}
	}()
	return nil
}

// routeControlResponse routes control_response messages to pending requests
func (a *Adapter) routeControlResponse(msg map[string]any) {
	response, _ := msg["response"].(map[string]any)
	requestID, _ := response["request_id"].(string)
	a.mu.Lock()
	defer a.mu.Unlock()
	if ch, exists := a.pendingReqs[requestID]; exists {
		subtype, _ := response["subtype"].(string)
		if subtype == "error" {
			errorMsg, _ := response["error"].(string)
			ch <- result{err: fmt.Errorf("control error: %s", errorMsg)}
		} else {
			responseData, _ := response["response"].(map[string]any)
			ch <- result{data: responseData}
		}
		delete(a.pendingReqs, requestID)
	}
}

// handleControlRequestAsync handles inbound control requests asynchronously
// Dependencies (perms, hooks, mcpServers) must be passed by the domain service that starts the router
func (a *Adapter) handleControlRequestAsync(
	ctx context.Context,
	msg map[string]any,
	perms *permissions.Service,
	hooks map[string]hooking.HookCallback,
	mcpServers map[string]ports.MCPServer,
) {
	requestID, _ := msg["request_id"].(string)
	// Handle the request
	responseData, err := a.HandleControlRequest(ctx, msg, perms, hooks, mcpServers)
	// Build response
	var response map[string]any
	if err != nil {
		response = map[string]any{
			"type": "control_response",
			"response": map[string]any{
				"subtype":    "error",
				"request_id": requestID,
				"error":      err.Error(),
			},
		}
	} else {
		response = map[string]any{
			"type": "control_response",
			"response": map[string]any{
				"subtype":    "success",
				"request_id": requestID,
				"response":   responseData,
			},
		}
	}
	// Send response
	resBytes, _ := json.Marshal(response)
	a.transport.Write(ctx, string(resBytes)+"\n")
}

// handleCanUseTool handles can_use_tool control requests
func (a *Adapter) handleCanUseTool(ctx context.Context, request map[string]any, perms *permissions.Service) (map[string]any, error) {
	toolName, _ := request["tool_name"].(string)
	input, _ := request["input"].(map[string]any)

	// Parse permission suggestions from control request
	var suggestions []permissions.PermissionUpdate
	if suggestionsData, ok := request["permission_suggestions"].([]any); ok {
		for _, sugData := range suggestionsData {
			if sugMap, ok := sugData.(map[string]any); ok {
				update := parsePermissionUpdate(sugMap)
				suggestions = append(suggestions, update)
			}
		}
	}

	if perms == nil {
		return nil, fmt.Errorf("permissions callback not provided")
	}

	result, err := perms.CheckToolUse(ctx, toolName, input, suggestions)
	if err != nil {
		return nil, err
	}

	// Convert PermissionResult to response format
	switch r := result.(type) {
	case *permissions.PermissionResultAllow:
		response := map[string]any{"allow": true}
		if r.UpdatedInput != nil {
			response["input"] = r.UpdatedInput
		}
		// Include updated permissions in response (for "always allow" flow)
		if r.UpdatedPermissions != nil && len(r.UpdatedPermissions) > 0 {
			response["updated_permissions"] = r.UpdatedPermissions
		}
		return response, nil
	case *permissions.PermissionResultDeny:
		return map[string]any{
			"allow":  false,
			"reason": r.Message,
		}, nil
	default:
		return nil, fmt.Errorf("unknown permission result type")
	}
}

// parsePermissionUpdate parses a single permission update from raw data
func parsePermissionUpdate(data map[string]any) permissions.PermissionUpdate {
	update := permissions.PermissionUpdate{}

	if updateType, ok := data["type"].(string); ok {
		update.Type = updateType
	}

	if rulesData, ok := data["rules"].([]any); ok {
		for _, ruleData := range rulesData {
			if ruleMap, ok := ruleData.(map[string]any); ok {
				toolName, _ := ruleMap["toolName"].(string)
				ruleContent := getStringPtr(ruleMap, "ruleContent")
				update.Rules = append(update.Rules, permissions.PermissionRuleValue{
					ToolName:    toolName,
					RuleContent: ruleContent,
				})
			}
		}
	}

	if behaviorStr, ok := data["behavior"].(string); ok {
		behavior := permissions.PermissionBehavior(behaviorStr)
		update.Behavior = &behavior
	}

	if modeStr, ok := data["mode"].(string); ok {
		mode := options.PermissionMode(modeStr)
		update.Mode = &mode
	}

	if dirsData, ok := data["directories"].([]any); ok {
		for _, dirData := range dirsData {
			if dir, ok := dirData.(string); ok {
				update.Directories = append(update.Directories, dir)
			}
		}
	}

	if destStr, ok := data["destination"].(string); ok {
		dest := permissions.PermissionUpdateDestination(destStr)
		update.Destination = &dest
	}

	return update
}

// handleHookCallback handles hook_callback control requests
func (a *Adapter) handleHookCallback(ctx context.Context, request map[string]any, hooks map[string]hooking.HookCallback) (map[string]any, error) {
	callbackID, _ := request["callback_id"].(string)
	input, _ := request["input"].(map[string]any)
	// toolUseID is a string, not *string - JSON decoding produces plain string values
	var toolUseID *string
	if id, ok := request["tool_use_id"].(string); ok {
		toolUseID = &id
	}
	callback, exists := hooks[callbackID]
	if !exists {
		return nil, fmt.Errorf("no hook callback found for ID: %s", callbackID)
	}
	// Execute callback with context for cancellation support
	hookCtx := hooking.HookContext{
		Signal: ctx, // Pass context for cancellation/timeout
	}
	result, err := callback(input, toolUseID, hookCtx)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// handleMCPMessage handles mcp_message control requests by proxying
// the raw message to the appropriate MCP server adapter (client or SDK type).
// The mcpServers map contains both:
// - ClientAdapter instances (connected to external MCP servers via stdio/HTTP/SSE)
// - ServerAdapter instances (wrapping user's in-process *mcp.Server instances)
func (a *Adapter) handleMCPMessage(ctx context.Context, request map[string]any, mcpServers map[string]ports.MCPServer) (map[string]any, error) {
	serverName, _ := request["server_name"].(string)
	mcpMessage, _ := request["message"].(map[string]any)

	// Look up server adapter by name
	server, exists := mcpServers[serverName]
	if !exists {
		return a.mcpErrorResponse(mcpMessage, -32601, fmt.Sprintf("Server '%s' not found", serverName)), nil
	}

	// Marshal message to JSON-RPC format
	mcpMessageBytes, err := json.Marshal(mcpMessage)
	if err != nil {
		return a.mcpErrorResponse(mcpMessage, -32603, "failed to marshal mcp message"), nil
	}

	// Forward to adapter (ClientAdapter or ServerAdapter)
	// The adapter handles routing to either external server or in-process server
	responseBytes, err := server.HandleMessage(ctx, mcpMessageBytes)
	if err != nil {
		return a.mcpErrorResponse(mcpMessage, -32603, err.Error()), nil
	}

	// Unmarshal response for control protocol
	var mcpResponse map[string]any
	if err := json.Unmarshal(responseBytes, &mcpResponse); err != nil {
		return a.mcpErrorResponse(mcpMessage, -32603, "failed to unmarshal mcp response"), nil
	}

	return map[string]any{
		"mcp_response": mcpResponse,
	}, nil
}

// mcpErrorResponse creates an MCP JSON-RPC error response
func (a *Adapter) mcpErrorResponse(message map[string]any, code int, msg string) map[string]any {
	return map[string]any{
		"mcp_response": map[string]any{
			"jsonrpc": "2.0",
			"id":      message["id"],
			"error": map[string]any{
				"code":    code,
				"message": msg,
			},
		},
	}
}

// randomHex generates a random hex string of n bytes
func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
```
### 3.3 Message Parser Adapter (adapters/parse/parser.go)
Priority: High
This adapter implements `ports.MessageParser`, converting raw JSON messages from the transport into typed domain messages.
```go
package parse

import (
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/ports"
)

// Adapter implements ports.MessageParser
// This is an INFRASTRUCTURE adapter - handles low-level message parsing
type Adapter struct{}

// Verify interface compliance at compile time
var _ ports.MessageParser = (*Adapter)(nil)

func NewAdapter() *Adapter {
	return &Adapter{}
}

// Parse implements ports.MessageParser
func (a *Adapter) Parse(data map[string]any) (messages.Message, error) {
	msgType, ok := data["type"].(string)
	if !ok {
		return nil, fmt.Errorf("message missing type field")
	}
	switch msgType {
	case "user":
		return a.parseUserMessage(data)
	case "assistant":
		return a.parseAssistantMessage(data)
	case "system":
		return a.parseSystemMessage(data)
	case "result":
		return a.parseResultMessage(data)
	case "stream_event":
		return a.parseStreamEvent(data)
	default:
		return nil, fmt.Errorf("unknown message type: %s", msgType)
	}
}
func (a *Adapter) parseUserMessage(data map[string]any) (messages.Message, error) {
	msg, _ := data["message"].(map[string]any)

	// Parse content (can be string or array of blocks)
	var content messages.MessageContent
	if contentStr, ok := msg["content"].(string); ok {
		content = messages.StringContent(contentStr)
	} else if contentArr, ok := msg["content"].([]any); ok {
		blocks, err := parseContentBlocks(contentArr)
		if err != nil {
			return nil, fmt.Errorf("parse user message content blocks: %w", err)
		}
		content = messages.BlockListContent(blocks)
	} else {
		return nil, fmt.Errorf("user message content must be string or array")
	}

	parentToolUseID := getStringPtr(data, "parent_tool_use_id")
	isSynthetic, _ := data["isSynthetic"].(bool)

	return &messages.UserMessage{
		Content:         content,
		ParentToolUseID: parentToolUseID,
		IsSynthetic:     isSynthetic,
	}, nil
}

func (a *Adapter) parseSystemMessage(data map[string]any) (messages.Message, error) {
	subtype, ok := data["subtype"].(string)
	if !ok {
		return nil, fmt.Errorf("system message missing subtype field")
	}

	// Data field is intentionally kept as map[string]any
	// Users can parse it into specific SystemMessageData types if needed
	// (SystemMessageInit, SystemMessageCompactBoundary)
	systemData, _ := data["data"].(map[string]any)
	if systemData == nil {
		systemData = make(map[string]any)
	}

	return &messages.SystemMessage{
		Subtype: subtype,
		Data:    systemData,
	}, nil
}

func (a *Adapter) parseResultMessage(data map[string]any) (messages.Message, error) {
	subtype, ok := data["subtype"].(string)
	if !ok {
		return nil, fmt.Errorf("result message missing subtype field")
	}

	// Type-safe approach: marshal map to JSON, then unmarshal into typed struct
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal result message: %w", err)
	}

	switch subtype {
	case "success":
		var result messages.ResultMessageSuccess
		if err := json.Unmarshal(jsonBytes, &result); err != nil {
			return nil, fmt.Errorf("unmarshal success result: %w", err)
		}
		return &result, nil

	case "error_max_turns", "error_during_execution":
		var result messages.ResultMessageError
		if err := json.Unmarshal(jsonBytes, &result); err != nil {
			return nil, fmt.Errorf("unmarshal error result: %w", err)
		}
		return &result, nil

	default:
		return nil, fmt.Errorf("unknown result subtype: %s", subtype)
	}
}

func (a *Adapter) parseStreamEvent(data map[string]any) (messages.Message, error) {
	uuid, ok := data["uuid"].(string)
	if !ok {
		return nil, fmt.Errorf("stream event missing uuid field")
	}

	sessionID, ok := data["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("stream event missing session_id field")
	}

	event, ok := data["event"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("stream event missing event field")
	}

	parentToolUseID := getStringPtr(data, "parent_tool_use_id")

	return &messages.StreamEvent{
		UUID:            uuid,
		SessionID:       sessionID,
		Event:           event, // Keep as map[string]any (raw Anthropic API event)
		ParentToolUseID: parentToolUseID,
	}, nil
}
func (a *Adapter) parseAssistantMessage(data map[string]any) (messages.Message, error) {
	// Parse content blocks
	msg, _ := data["message"].(map[string]any)
	contentArray, _ := msg["content"].([]any)
	var blocks []messages.ContentBlock
	for _, item := range contentArray {
		block, _ := item.(map[string]any)
		blockType, _ := block["type"].(string)
		switch blockType {
		case "text":
			text, _ := block["text"].(string)
			blocks = append(blocks, messages.TextBlock{Text: text})
		case "thinking":
			thinking, _ := block["thinking"].(string)
			signature, _ := block["signature"].(string)
			blocks = append(blocks, messages.ThinkingBlock{
				Thinking:  thinking,
				Signature: signature,
			})
		case "tool_use":
			id, _ := block["id"].(string)
			name, _ := block["name"].(string)
			input, _ := block["input"].(map[string]any)
			blocks = append(blocks, messages.ToolUseBlock{
				ID:    id,
				Name:  name,
				Input: input,
			})
		case "tool_result":
			toolUseID, _ := block["tool_use_id"].(string)
			content := block["content"]
			isError, _ := block["is_error"].(*bool)
			blocks = append(blocks, messages.ToolResultBlock{
				ToolUseID: toolUseID,
				Content:   content,
				IsError:   isError,
			})
		}
	}
	model, _ := msg["model"].(string)
	parentToolUseID := getStringPtr(data, "parent_tool_use_id")
	return &messages.AssistantMessage{
		Content:         blocks,
		Model:           model,
		ParentToolUseID: parentToolUseID,
	}, nil
}

// Helper function for extracting optional string pointers
func getStringPtr(data map[string]any, key string) *string {
	if val, ok := data[key].(string); ok {
		return &val
	}
	return nil
}

// parseUsageStats parses usage statistics from raw data
func parseUsageStats(data any) (messages.UsageStats, error) {
	if data == nil {
		return messages.UsageStats{}, nil
	}

	usageMap, ok := data.(map[string]any)
	if !ok {
		return messages.UsageStats{}, fmt.Errorf("usage must be an object")
	}

	inputTokens, _ := usageMap["input_tokens"].(float64)
	outputTokens, _ := usageMap["output_tokens"].(float64)
	cacheReadInputTokens, _ := usageMap["cache_read_input_tokens"].(float64)
	cacheCreationInputTokens, _ := usageMap["cache_creation_input_tokens"].(float64)

	return messages.UsageStats{
		InputTokens:              int(inputTokens),
		OutputTokens:             int(outputTokens),
		CacheReadInputTokens:     int(cacheReadInputTokens),
		CacheCreationInputTokens: int(cacheCreationInputTokens),
	}, nil
}

// parseModelUsage parses per-model usage statistics
func parseModelUsage(data any) (map[string]messages.ModelUsage, error) {
	if data == nil {
		return make(map[string]messages.ModelUsage), nil
	}

	modelUsageMap, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("modelUsage must be an object")
	}

	result := make(map[string]messages.ModelUsage)
	for modelName, usageData := range modelUsageMap {
		usageMap, ok := usageData.(map[string]any)
		if !ok {
			continue
		}

		inputTokens, _ := usageMap["inputTokens"].(float64)
		outputTokens, _ := usageMap["outputTokens"].(float64)
		cacheReadInputTokens, _ := usageMap["cacheReadInputTokens"].(float64)
		cacheCreationInputTokens, _ := usageMap["cacheCreationInputTokens"].(float64)
		webSearchRequests, _ := usageMap["webSearchRequests"].(float64)
		costUSD, _ := usageMap["costUSD"].(float64)
		contextWindow, _ := usageMap["contextWindow"].(float64)

		result[modelName] = messages.ModelUsage{
			InputTokens:              int(inputTokens),
			OutputTokens:             int(outputTokens),
			CacheReadInputTokens:     int(cacheReadInputTokens),
			CacheCreationInputTokens: int(cacheCreationInputTokens),
			WebSearchRequests:        int(webSearchRequests),
			CostUSD:                  costUSD,
			ContextWindow:            int(contextWindow),
		}
	}

	return result, nil
}

// parsePermissionDenials parses array of permission denials
func parsePermissionDenials(data any) ([]messages.PermissionDenial, error) {
	if data == nil {
		return []messages.PermissionDenial{}, nil
	}

	denialsArray, ok := data.([]any)
	if !ok {
		return nil, fmt.Errorf("permission_denials must be an array")
	}

	result := make([]messages.PermissionDenial, 0, len(denialsArray))
	for _, denialData := range denialsArray {
		denialMap, ok := denialData.(map[string]any)
		if !ok {
			continue
		}

		toolName, _ := denialMap["tool_name"].(string)
		toolUseID, _ := denialMap["tool_use_id"].(string)
		toolInput, _ := denialMap["tool_input"].(map[string]any)

		result = append(result, messages.PermissionDenial{
			ToolName:  toolName,
			ToolUseID: toolUseID,
			ToolInput: toolInput,
		})
	}

	return result, nil
}

// parseContentBlocks parses an array of content blocks
func parseContentBlocks(contentArr []any) ([]messages.ContentBlock, error) {
	blocks := make([]messages.ContentBlock, 0, len(contentArr))

	for _, item := range contentArr {
		block, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("content block must be an object")
		}

		blockType, ok := block["type"].(string)
		if !ok {
			return nil, fmt.Errorf("content block missing type field")
		}

		switch blockType {
		case "text":
			textBlock, err := parseTextBlock(block)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, textBlock)

		case "thinking":
			thinkingBlock, err := parseThinkingBlock(block)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, thinkingBlock)

		case "tool_use":
			toolUseBlock, err := parseToolUseBlock(block)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, toolUseBlock)

		case "tool_result":
			toolResultBlock, err := parseToolResultBlock(block)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, toolResultBlock)

		default:
			return nil, fmt.Errorf("unknown content block type: %s", blockType)
		}
	}

	return blocks, nil
}

// parseTextBlock parses a text content block
func parseTextBlock(block map[string]any) (messages.TextBlock, error) {
	text, ok := block["text"].(string)
	if !ok {
		return messages.TextBlock{}, fmt.Errorf("text block missing text field")
	}

	return messages.TextBlock{
		Text: text,
	}, nil
}

// parseThinkingBlock parses a thinking content block
func parseThinkingBlock(block map[string]any) (messages.ThinkingBlock, error) {
	thinking, ok := block["thinking"].(string)
	if !ok {
		return messages.ThinkingBlock{}, fmt.Errorf("thinking block missing thinking field")
	}

	signature, _ := block["signature"].(string)

	return messages.ThinkingBlock{
		Thinking:  thinking,
		Signature: signature,
	}, nil
}

// parseToolUseBlock parses a tool_use content block
func parseToolUseBlock(block map[string]any) (messages.ToolUseBlock, error) {
	id, ok := block["id"].(string)
	if !ok {
		return messages.ToolUseBlock{}, fmt.Errorf("tool_use block missing id field")
	}

	name, ok := block["name"].(string)
	if !ok {
		return messages.ToolUseBlock{}, fmt.Errorf("tool_use block missing name field")
	}

	input, ok := block["input"].(map[string]any)
	if !ok {
		// Input can be missing or null
		input = make(map[string]any)
	}

	return messages.ToolUseBlock{
		ID:    id,
		Name:  name,
		Input: input,
	}, nil
}

// parseToolResultBlock parses a tool_result content block
func parseToolResultBlock(block map[string]any) (messages.ToolResultBlock, error) {
	toolUseID, ok := block["tool_use_id"].(string)
	if !ok {
		return messages.ToolResultBlock{}, fmt.Errorf("tool_result block missing tool_use_id field")
	}

	// Parse content (can be string or array of content blocks)
	var content messages.ToolResultContent
	if contentStr, ok := block["content"].(string); ok {
		content = messages.ToolResultStringContent(contentStr)
	} else if contentArr, ok := block["content"].([]any); ok {
		// Tool result content can be an array of raw content blocks (maps)
		blockMaps := make([]map[string]any, 0, len(contentArr))
		for _, item := range contentArr {
			if blockMap, ok := item.(map[string]any); ok {
				blockMaps = append(blockMaps, blockMap)
			}
		}
		content = messages.ToolResultBlockListContent(blockMaps)
	} else {
		return messages.ToolResultBlock{}, fmt.Errorf("tool_result content must be string or array")
	}

	// is_error is optional
	var isError *bool
	if isErrorVal, ok := block["is_error"].(bool); ok {
		isError = &isErrorVal
	}

	return messages.ToolResultBlock{
		ToolUseID: toolUseID,
		Content:   content,
		IsError:   isError,
	}, nil
}
```

---

## Linting Compliance Notes

### File Size Requirements (175 line limit)

**All adapters require significant decomposition:**

**adapters/cli/ package:**
- ❌ Single `transport.go` (400+ lines planned)
- ✅ Split into 7 files:
  - `transport.go` - Adapter struct + interface (60 lines)
  - `connect.go` - Connection logic (70 lines)
  - `command.go` - Command building (80 lines)
  - `io.go` - I/O handling (90 lines)
  - `discovery.go` - CLI discovery (50 lines)
  - `process.go` - Process management (60 lines)
  - `errors.go` - Error types (40 lines)

**adapters/jsonrpc/ package:**
- ❌ Single `protocol.go` (350+ lines planned)
- ✅ Split into 5 files:
  - `protocol.go` - Handler struct + interface (50 lines)
  - `control.go` - Control request handling (80 lines)
  - `routing.go` - Message routing (90 lines)
  - `handlers.go` - Per-type request handlers (80 lines)
  - `state.go` - State tracking (50 lines)

**adapters/parse/ package:**
- ❌ Single `parser.go` shown above (1100+ lines!)
- ✅ Split into 9 files:
  - `parser.go` - Main parser interface (40 lines)
  - `user.go` - UserMessage parsing (60 lines)
  - `assistant.go` - AssistantMessage parsing (80 lines)
  - `system.go` - SystemMessage parsing (70 lines)
  - `result.go` - ResultMessage parsing (90 lines)
  - `stream.go` - StreamEvent parsing (50 lines)
  - `content.go` - ContentBlock parsing (80 lines)
  - `usage.go` - Usage stats parsing (60 lines)
  - `helpers.go` - Shared helper functions (40 lines)

**adapters/mcp/ package:**
- ❌ Single file approach insufficient - need TWO distinct adapters
- ✅ Split into 3 files (under 175 lines each):
  - `client.go` - ClientAdapter for external MCP servers (90 lines)
  - `server.go` - ServerAdapter for SDK MCP servers (85 lines)
  - `helpers.go` - Shared utilities (40 lines)

### Complexity Hotspots

**CLI adapter:**
- Command building → Use builder pattern with method chaining
- Process I/O → Extract reader/writer functions
- CLI discovery → Extract path search functions

**JSON-RPC adapter:**
- Request routing → Use handler registry map instead of switch
- State tracking → Extract state manager struct
- Timeout handling → Extract timeout wrapper function

**Parser adapter:**
- Type switching → Extract per-type parser functions
- Field extraction → Extract helper validators
- Content parsing → Extract block-specific parsers

### Parameter Reduction Patterns

Many adapter functions exceed 4-parameter limit:

```go
// BAD: 6 parameters
func (a *Adapter) HandleControlRequest(
    ctx context.Context,
    req map[string]any,
    perms *permissions.Service,
    hooks map[string]hooking.HookCallback,
    mcpServers map[string]ports.MCPServer,
    logger Logger,
) (map[string]any, error)

// GOOD: Use dependencies struct (3 parameters)
type ControlDependencies struct {
    Perms      *permissions.Service
    Hooks      map[string]hooking.HookCallback
    MCPServers map[string]ports.MCPServer
}

func (a *Adapter) HandleControlRequest(
    ctx context.Context,
    req map[string]any,
    deps ControlDependencies,
) (map[string]any, error)
```

### Checklist

- [ ] All files under 175 lines
- [ ] Parser split into per-message-type files
- [ ] CLI command builder uses fluent/builder pattern
- [ ] I/O operations in separate helper files
- [ ] Process management uses extracted functions
- [ ] Handler functions use dependency structs (≤4 params)
- [ ] All functions under 25 lines
- [ ] Max nesting depth ≤ 3 levels
