// Package cli provides a CLI adapter for the Claude transport interface.
package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/conneroisu/claude/pkg/claude/options"
)

// BuildCommand constructs the CLI command with all options
// Exported for testing purposes
func (a *Adapter) BuildCommand() ([]string, error) {
	cmd := []string{
		a.cliPath,
		"--output-format",
		"stream-json",
		"--verbose",
	}

	a.addSystemPromptArgs(&cmd)
	a.addToolArgs(&cmd)
	a.addModelArgs(&cmd)
	a.addPermissionArgs(&cmd)
	a.addSessionArgs(&cmd)
	a.addSettingArgs(&cmd)
	a.addDirectoryArgs(&cmd)

	if err := a.addMCPArgs(&cmd); err != nil {
		return nil, err
	}

	a.addExtraArgs(&cmd)

	return cmd, nil
}

// addSystemPromptArgs handles system prompt configuration.
// Supports both direct string prompts and preset-based prompts with
// optional append.
func (a *Adapter) addSystemPromptArgs(cmd *[]string) {
	if a.options.SystemPrompt == nil {
		return
	}

	switch sp := a.options.SystemPrompt.(type) {
	case options.StringSystemPrompt:
		*cmd = append(*cmd, "--system-prompt", string(sp))
	case options.PresetSystemPrompt:
		if sp.Append != nil {
			*cmd = append(*cmd, "--append-system-prompt", *sp.Append)
		}
	}
}

// addToolArgs configures tool access control.
// Both allow and deny lists can be specified simultaneously,
// with deny list taking precedence for overlapping tools.
func (a *Adapter) addToolArgs(cmd *[]string) {
	if len(a.options.AllowedTools) > 0 {
		*cmd = append(
			*cmd,
			"--allowedTools",
			strings.Join(a.options.AllowedTools, ","),
		)
	}
	if len(a.options.DisallowedTools) > 0 {
		*cmd = append(
			*cmd,
			"--disallowedTools",
			strings.Join(a.options.DisallowedTools, ","),
		)
	}
}

// addModelArgs configures model selection and conversation limits.
// MaxTurns helps prevent infinite loops in agent interactions.
func (a *Adapter) addModelArgs(cmd *[]string) {
	if a.options.Model != nil {
		*cmd = append(*cmd, "--model", *a.options.Model)
	}
	if a.options.MaxTurns != nil {
		*cmd = append(
			*cmd,
			"--max-turns",
			fmt.Sprintf("%d", *a.options.MaxTurns),
		)
	}
}

// addPermissionArgs configures permission handling for tool usage.
// The permission prompt tool enables custom authorization flows.
func (a *Adapter) addPermissionArgs(cmd *[]string) {
	if a.options.PermissionMode != nil {
		*cmd = append(
			*cmd,
			"--permission-mode",
			string(*a.options.PermissionMode),
		)
	}
	if a.options.PermissionPromptToolName != nil {
		*cmd = append(
			*cmd,
			"--permission-prompt-tool",
			*a.options.PermissionPromptToolName,
		)
	}
}

// addSessionArgs manages conversation continuity and session management.
// Resume allows picking up a specific session, while fork creates a branch.
func (a *Adapter) addSessionArgs(cmd *[]string) {
	if a.options.ContinueConversation {
		*cmd = append(*cmd, "--continue")
	}
	if a.options.Resume != nil {
		*cmd = append(*cmd, "--resume", *a.options.Resume)
	}
	if a.options.ForkSession {
		*cmd = append(*cmd, "--fork-session")
	}
}

// addSettingArgs configures settings file location and source priority.
// Setting sources determine the precedence order for configuration merging.
func (a *Adapter) addSettingArgs(cmd *[]string) {
	if a.options.Settings != nil {
		*cmd = append(*cmd, "--settings", *a.options.Settings)
	}
	if len(a.options.SettingSources) == 0 {
		return
	}
	sources := make([]string, len(a.options.SettingSources))
	for i, s := range a.options.SettingSources {
		sources[i] = string(s)
	}
	*cmd = append(
		*cmd,
		"--setting-sources",
		strings.Join(sources, ","),
	)
}

// addDirectoryArgs adds additional directories to the agent's context.
// Each directory is passed separately to preserve ordering.
func (a *Adapter) addDirectoryArgs(cmd *[]string) {
	for _, dir := range a.options.AddDirs {
		*cmd = append(*cmd, "--add-dir", dir)
	}
}

// addMCPArgs serializes MCP server configuration to JSON.
// MCP servers enable Claude to interact with external tools and services.
func (a *Adapter) addMCPArgs(cmd *[]string) error {
	if len(a.options.MCPServers) == 0 {
		return nil
	}

	mcpConfig := map[string]any{"mcpServers": a.options.MCPServers}
	jsonBytes, err := json.Marshal(mcpConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal MCP config: %w", err)
	}
	*cmd = append(*cmd, "--mcp-config", string(jsonBytes))

	return nil
}

// addExtraArgs appends user-provided flags for extensibility.
// Supports both boolean flags (nil value) and value flags.
func (a *Adapter) addExtraArgs(cmd *[]string) {
	for flag, value := range a.options.ExtraArgs {
		if value == nil {
			*cmd = append(*cmd, "--"+flag)
		} else {
			*cmd = append(*cmd, "--"+flag, *value)
		}
	}
}
