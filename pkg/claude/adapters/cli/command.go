package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/conneroisu/claude/pkg/claude/options"
)

// BuildCommand constructs the CLI command with all options.
// Exported for testing purposes.
// This method chains together all option-specific methods to build
// the complete command-line arguments for the Claude CLI.
func (a *Adapter) BuildCommand() ([]string, error) {
	// Start with base command and required flags.
	cmd := []string{
		a.cliPath,
		"--output-format",
		"stream-json",
		"--verbose",
	}

	// Add options in logical groups.
	cmd = a.addSystemPrompt(cmd)
	cmd = a.addTools(cmd)
	cmd = a.addModelAndTurns(cmd)
	cmd = a.addPermissions(cmd)
	cmd = a.addSession(cmd)
	cmd = a.addSettings(cmd)
	cmd = a.addDirectories(cmd)

	// MCP servers require JSON marshaling and can error.
	mcpCmd, err := a.addMCPServers(cmd)
	if err != nil {
		return nil, err
	}
	cmd = mcpCmd

	// Add any user-specified extra arguments last.
	cmd = a.addExtraArgs(cmd)

	return cmd, nil
}

// addSystemPrompt adds system prompt arguments.
// Handles both string prompts and preset-based prompts.
//nolint:revive // modifies-parameter: cmd modification is intentional
func (a *Adapter) addSystemPrompt(cmd []string) []string {
	if a.options.SystemPrompt == nil {
		return cmd
	}

	// Use type assertion to handle different prompt types.
	switch sp := a.options.SystemPrompt.(type) {
	case options.StringSystemPrompt:
		cmd = append(cmd, "--system-prompt", string(sp))
	case options.PresetSystemPrompt:
		if sp.Append != nil {
			cmd = append(
				cmd,
				"--append-system-prompt",
				*sp.Append,
			)
		}
	}

	return cmd
}
//nolint:revive // modifies-parameter: cmd modification is intentional

// addTools adds tool configuration arguments.
//nolint:revive // modifies-parameter: Builder pattern requires modification
func (a *Adapter) addTools(cmd []string) []string {
	if len(a.options.AllowedTools) > 0 {
		cmd = append(
			cmd,
			"--allowedTools",
			strings.Join(a.options.AllowedTools, ","),
		)
	}

	if len(a.options.DisallowedTools) > 0 {
		cmd = append(
			cmd,
			"--disallowedTools",
			strings.Join(a.options.DisallowedTools, ","),
		)
	}

	return cmd
}

// addModelAndTurns adds model and max turns arguments.
//nolint:revive // modifies-parameter: Builder pattern requires modification
func (a *Adapter) addModelAndTurns(cmd []string) []string {
	if a.options.Model != nil {
		cmd = append(cmd, "--model", *a.options.Model)
	}

	if a.options.MaxTurns != nil {
		cmd = append(
			cmd,
			"--max-turns",
			fmt.Sprintf("%d", *a.options.MaxTurns),
		)
	}
//nolint:revive // modifies-parameter: cmd modification is intentional

	return cmd
}

//nolint:revive // modifies-parameter: cmd modification is intentional
// addPermissions adds permission configuration arguments.
//nolint:revive // modifies-parameter: Builder pattern requires modification
func (a *Adapter) addPermissions(cmd []string) []string {
	if a.options.PermissionMode != nil {
		cmd = append(
			cmd,
			"--permission-mode",
			string(*a.options.PermissionMode),
		)
	}

	if a.options.PermissionPromptToolName != nil {
		cmd = append(
			cmd,
			"--permission-prompt-tool",
			*a.options.PermissionPromptToolName,
		)
	}

	return cmd
}

// addSession adds session management arguments.
//nolint:revive // modifies-parameter: Builder pattern requires modification
func (a *Adapter) addSession(cmd []string) []string {
	if a.options.ContinueConversation {
		cmd = append(cmd, "--continue")
	}

	if a.options.Resume != nil {
		cmd = append(cmd, "--resume", *a.options.Resume)
	}

	if a.options.ForkSession {
//nolint:revive // modifies-parameter: cmd modification is intentional
		cmd = append(cmd, "--fork-session")
	}

	return cmd
}

// addSettings adds settings configuration arguments.
//nolint:revive // modifies-parameter: Builder pattern requires modification
func (a *Adapter) addSettings(cmd []string) []string {
	if a.options.Settings != nil {
		cmd = append(cmd, "--settings", *a.options.Settings)
	}

	if len(a.options.SettingSources) > 0 {
		sources := make([]string, len(a.options.SettingSources))
		for i, s := range a.options.SettingSources {
			sources[i] = string(s)
		}
		cmd = append(
			cmd,
			"--setting-sources",
			strings.Join(sources, ","),
		)
	}

	return cmd
}

// addDirectories adds directory arguments.
//nolint:revive // modifies-parameter: Builder pattern requires modification
func (a *Adapter) addDirectories(cmd []string) []string {
	for _, dir := range a.options.AddDirs {
		cmd = append(cmd, "--add-dir", dir)
	}

	return cmd
}

// addMCPServers adds MCP server configuration.
// MCP servers are passed as JSON configuration to the CLI.
//nolint:revive // modifies-parameter: Builder pattern requires modification
func (a *Adapter) addMCPServers(cmd []string) ([]string, error) {
	if len(a.options.MCPServers) == 0 {
		return cmd, nil
	}

	// Wrap servers in expected configuration structure.
	mcpConfig := map[string]any{
		"mcpServers": a.options.MCPServers,
	}

	jsonBytes, err := json.Marshal(mcpConfig)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to marshal MCP config: %w",
			err,
		)
	}

	cmd = append(cmd, "--mcp-config", string(jsonBytes))

	return cmd, nil
}

// addExtraArgs adds extra command-line arguments.
//nolint:revive // modifies-parameter: cmd modification is intentional
// Supports both boolean flags and flags with values.
func (a *Adapter) addExtraArgs(cmd []string) []string {
	for flag, value := range a.options.ExtraArgs {
		// Boolean flags have nil values.
		if value == nil {
			cmd = append(cmd, "--"+flag)
		} else {
			// Flags with values use two arguments.
			cmd = append(cmd, "--"+flag, *value)
		}
	}

	return cmd
}
