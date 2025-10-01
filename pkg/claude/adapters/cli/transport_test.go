package cli

import (
	"testing"

	"github.com/conneroisu/claude/pkg/claude/options"
)

func TestNewAdapter(t *testing.T) {
	t.Run("creates adapter with default buffer size", func(t *testing.T) {
		opts := &options.AgentOptions{}
		adapter := NewAdapter(opts)

		if adapter == nil {
			t.Fatal("Expected adapter to be created")
		}
		if adapter.maxBufferSize != defaultMaxBufferSize {
			t.Errorf("Expected default buffer size %d, got %d", defaultMaxBufferSize, adapter.maxBufferSize)
		}
	})

	t.Run("creates adapter with custom buffer size", func(t *testing.T) {
		customSize := 2 * 1024 * 1024
		opts := &options.AgentOptions{
			MaxBufferSize: &customSize,
		}
		adapter := NewAdapter(opts)

		if adapter.maxBufferSize != customSize {
			t.Errorf("Expected buffer size %d, got %d", customSize, adapter.maxBufferSize)
		}
	})
}

func TestBuildCommand(t *testing.T) {
	t.Run("builds basic command", func(t *testing.T) {
		opts := &options.AgentOptions{}
		adapter := NewAdapter(opts)
		adapter.cliPath = "/usr/bin/claude"

		cmd, err := adapter.BuildCommand()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(cmd) < 3 {
			t.Fatal("Expected command to have at least 3 parts")
		}
		if cmd[0] != "/usr/bin/claude" {
			t.Errorf("Expected first arg to be CLI path, got %s", cmd[0])
		}
		if !contains(cmd, "--output-format") {
			t.Error("Expected --output-format flag")
		}
		if !contains(cmd, "stream-json") {
			t.Error("Expected stream-json format")
		}
	})

	t.Run("includes model flag", func(t *testing.T) {
		model := "claude-sonnet-4-5-20250929"
		opts := &options.AgentOptions{
			Model: &model,
		}
		adapter := NewAdapter(opts)
		adapter.cliPath = "claude"

		cmd, err := adapter.BuildCommand()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !contains(cmd, "--model") {
			t.Error("Expected --model flag")
		}
		if !contains(cmd, model) {
			t.Error("Expected model value in command")
		}
	})

	t.Run("includes max turns flag", func(t *testing.T) {
		maxTurns := 10
		opts := &options.AgentOptions{
			MaxTurns: &maxTurns,
		}
		adapter := NewAdapter(opts)
		adapter.cliPath = "claude"

		cmd, err := adapter.BuildCommand()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !contains(cmd, "--max-turns") {
			t.Error("Expected --max-turns flag")
		}
		if !contains(cmd, "10") {
			t.Error("Expected max turns value in command")
		}
	})

	t.Run("includes allowed tools", func(t *testing.T) {
		opts := &options.AgentOptions{
			AllowedTools: []string{"Read", "Write"},
		}
		adapter := NewAdapter(opts)
		adapter.cliPath = "claude"

		cmd, err := adapter.BuildCommand()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !contains(cmd, "--allowedTools") {
			t.Error("Expected --allowedTools flag")
		}
		if !contains(cmd, "Read,Write") {
			t.Error("Expected allowed tools value in command")
		}
	})

	t.Run("includes disallowed tools", func(t *testing.T) {
		opts := &options.AgentOptions{
			DisallowedTools: []string{"Bash"},
		}
		adapter := NewAdapter(opts)
		adapter.cliPath = "claude"

		cmd, err := adapter.BuildCommand()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !contains(cmd, "--disallowedTools") {
			t.Error("Expected --disallowedTools flag")
		}
		if !contains(cmd, "Bash") {
			t.Error("Expected disallowed tools value in command")
		}
	})

	t.Run("includes system prompt", func(t *testing.T) {
		opts := &options.AgentOptions{
			SystemPrompt: options.StringSystemPrompt("You are helpful"),
		}
		adapter := NewAdapter(opts)
		adapter.cliPath = "claude"

		cmd, err := adapter.BuildCommand()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !contains(cmd, "--system-prompt") {
			t.Error("Expected --system-prompt flag")
		}
		if !contains(cmd, "You are helpful") {
			t.Error("Expected system prompt value in command")
		}
	})

	t.Run("includes preset system prompt append", func(t *testing.T) {
		appendText := "Additional context"
		opts := &options.AgentOptions{
			SystemPrompt: options.PresetSystemPrompt{
				Type:   "preset",
				Preset: "default",
				Append: &appendText,
			},
		}
		adapter := NewAdapter(opts)
		adapter.cliPath = "claude"

		cmd, err := adapter.BuildCommand()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !contains(cmd, "--append-system-prompt") {
			t.Error("Expected --append-system-prompt flag")
		}
		if !contains(cmd, appendText) {
			t.Error("Expected append text in command")
		}
	})

	t.Run("includes permission mode", func(t *testing.T) {
		mode := options.PermissionModeBypassPermissions
		opts := &options.AgentOptions{
			PermissionMode: &mode,
		}
		adapter := NewAdapter(opts)
		adapter.cliPath = "claude"

		cmd, err := adapter.BuildCommand()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !contains(cmd, "--permission-mode") {
			t.Error("Expected --permission-mode flag")
		}
		if !contains(cmd, string(mode)) {
			t.Error("Expected permission mode value in command")
		}
	})

	t.Run("includes session flags", func(t *testing.T) {
		resume := "session_123"
		opts := &options.AgentOptions{
			ContinueConversation: true,
			Resume:               &resume,
			ForkSession:          true,
		}
		adapter := NewAdapter(opts)
		adapter.cliPath = "claude"

		cmd, err := adapter.BuildCommand()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !contains(cmd, "--continue") {
			t.Error("Expected --continue flag")
		}
		if !contains(cmd, "--resume") {
			t.Error("Expected --resume flag")
		}
		if !contains(cmd, "session_123") {
			t.Error("Expected session ID in command")
		}
		if !contains(cmd, "--fork-session") {
			t.Error("Expected --fork-session flag")
		}
	})

	t.Run("includes settings", func(t *testing.T) {
		settings := "/path/to/settings.json"
		opts := &options.AgentOptions{
			Settings: &settings,
			SettingSources: []options.SettingSource{
				options.SettingSourceUser,
				options.SettingSourceProject,
			},
		}
		adapter := NewAdapter(opts)
		adapter.cliPath = "claude"

		cmd, err := adapter.BuildCommand()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !contains(cmd, "--settings") {
			t.Error("Expected --settings flag")
		}
		if !contains(cmd, settings) {
			t.Error("Expected settings path in command")
		}
		if !contains(cmd, "--setting-sources") {
			t.Error("Expected --setting-sources flag")
		}
	})

	t.Run("includes additional directories", func(t *testing.T) {
		opts := &options.AgentOptions{
			AddDirs: []string{"/dir1", "/dir2"},
		}
		adapter := NewAdapter(opts)
		adapter.cliPath = "claude"

		cmd, err := adapter.BuildCommand()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Should have two --add-dir flags
		count := 0
		for _, arg := range cmd {
			if arg == "--add-dir" {
				count++
			}
		}
		if count != 2 {
			t.Errorf("Expected 2 --add-dir flags, got %d", count)
		}
	})

	t.Run("includes MCP servers config", func(t *testing.T) {
		opts := &options.AgentOptions{
			MCPServers: map[string]options.MCPServerConfig{
				"test": options.StdioServerConfig{
					Type:    "stdio",
					Command: "test-server",
				},
			},
		}
		adapter := NewAdapter(opts)
		adapter.cliPath = "claude"

		cmd, err := adapter.BuildCommand()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !contains(cmd, "--mcp-config") {
			t.Error("Expected --mcp-config flag")
		}
	})

	t.Run("includes extra arguments", func(t *testing.T) {
		value := "test-value"
		opts := &options.AgentOptions{
			ExtraArgs: map[string]*string{
				"custom-flag": &value,
				"bool-flag":   nil,
			},
		}
		adapter := NewAdapter(opts)
		adapter.cliPath = "claude"

		cmd, err := adapter.BuildCommand()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !contains(cmd, "--custom-flag") {
			t.Error("Expected --custom-flag")
		}
		if !contains(cmd, "test-value") {
			t.Error("Expected custom flag value")
		}
		if !contains(cmd, "--bool-flag") {
			t.Error("Expected --bool-flag")
		}
	})
}

func TestIsReady(t *testing.T) {
	t.Run("returns false before connect", func(t *testing.T) {
		opts := &options.AgentOptions{}
		adapter := NewAdapter(opts)

		if adapter.IsReady() {
			t.Error("Expected IsReady to return false before connect")
		}
	})
}

func TestClose(t *testing.T) {
	t.Run("can close without error when not connected", func(t *testing.T) {
		opts := &options.AgentOptions{}
		adapter := NewAdapter(opts)

		err := adapter.Close()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("marks as not ready after close", func(t *testing.T) {
		opts := &options.AgentOptions{}
		adapter := NewAdapter(opts)
		adapter.ready = true

		_ = adapter.Close()

		if adapter.IsReady() {
			t.Error("Expected IsReady to return false after close")
		}
	})
}

func TestEndInput(t *testing.T) {
	t.Run("returns nil when stdin is nil", func(t *testing.T) {
		opts := &options.AgentOptions{}
		adapter := NewAdapter(opts)

		err := adapter.EndInput()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

// Helper function
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}

	return false
}
