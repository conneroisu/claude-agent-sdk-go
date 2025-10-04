package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/conneroisu/claude/pkg/claude/options"
)

// BuildCommand constructs the CLI command with all options.
// Exported for testing purposes.
func (a *Adapter) BuildCommand() ([]string, error) {
	cmd := []string{a.cliPath, "--output-format", "stream-json", "--verbose"}

	// System prompt
	if a.options.SystemPrompt != nil {
		cmd = a.addSystemPromptArgs(cmd)
	}

	// Tools
	cmd = a.addToolArgs(cmd)

	// Model and turns
	cmd = a.addModelArgs(cmd)

	// Permissions
	cmd = a.addPermissionArgs(cmd)

	// Session
	cmd = a.addSessionArgs(cmd)

	// Settings
	cmd = a.addSettingsArgs(cmd)

	// Directories
	for _, dir := range a.options.AddDirs {
		cmd = append(cmd, "--add-dir", dir)
	}

	// MCP servers
	if len(a.options.MCPServers) > 0 {
		mcpArg, err := a.buildMCPConfig()
		if err != nil {
			return nil, err
		}
		cmd = append(cmd, "--mcp-config", mcpArg)
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

func (a *Adapter) addSystemPromptArgs(cmd []string) []string {
	switch sp := a.options.SystemPrompt.(type) {
	case options.StringSystemPrompt:
		cmd = append(cmd, "--system-prompt", string(sp))
	case options.PresetSystemPrompt:
		if sp.Append != nil {
			cmd = append(cmd, "--append-system-prompt", *sp.Append)
		}
	}

	return cmd
}

func (a *Adapter) addToolArgs(cmd []string) []string {
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

	return cmd
}

func (a *Adapter) addModelArgs(cmd []string) []string {
	if a.options.Model != nil {
		cmd = append(cmd, "--model", *a.options.Model)
	}
	if a.options.MaxTurns != nil {
		cmd = append(cmd, "--max-turns", fmt.Sprintf("%d", *a.options.MaxTurns))
	}

	return cmd
}

func (a *Adapter) addPermissionArgs(cmd []string) []string {
	if a.options.PermissionMode != nil {
		cmd = append(cmd, "--permission-mode", string(*a.options.PermissionMode))
	}
	if a.options.PermissionPromptToolName != nil {
		cmd = append(cmd, "--permission-prompt-tool", *a.options.PermissionPromptToolName)
	}

	return cmd
}

func (a *Adapter) addSessionArgs(cmd []string) []string {
	if a.options.ContinueConversation {
		cmd = append(cmd, "--continue")
	}
	if a.options.Resume != nil {
		cmd = append(cmd, "--resume", *a.options.Resume)
	}
	if a.options.ForkSession {
		cmd = append(cmd, "--fork-session")
	}

	return cmd
}

func (a *Adapter) addSettingsArgs(cmd []string) []string {
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

	return cmd
}

func (a *Adapter) buildMCPConfig() (string, error) {
	mcpConfig := map[string]any{"mcpServers": a.options.MCPServers}
	jsonBytes, err := json.Marshal(mcpConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal MCP config: %w", err)
	}
	return string(jsonBytes), nil
}
