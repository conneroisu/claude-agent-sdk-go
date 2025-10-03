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

// Adapter implements ports.Transport using CLI subprocess.
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

// Verify interface compliance at compile time.
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

// findCLI locates the Claude CLI binary.
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
// Exported for testing purposes.
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
		tools := make([]string, len(a.options.AllowedTools))
		for i, t := range a.options.AllowedTools {
			tools[i] = string(t)
		}
		cmd = append(cmd, "--allowedTools", strings.Join(tools, ","))
	}
	if len(a.options.DisallowedTools) > 0 {
		tools := make([]string, len(a.options.DisallowedTools))
		for i, t := range a.options.DisallowedTools {
			tools[i] = string(t)
		}
		cmd = append(cmd, "--disallowedTools", strings.Join(tools, ","))
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
