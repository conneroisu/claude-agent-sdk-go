//nolint:revive // Command building functions intentionally modify and return cmd slice
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
	cmd := []string{
		a.cliPath,
		"--output-format",
		"stream-json",
		"--verbose",
	}

	cmd = a.addSystemPrompt(cmd)
	cmd = a.addToolOptions(cmd)
	cmd = a.addModelAndTurns(cmd)
	cmd = a.addPermissions(cmd)
	cmd = a.addSession(cmd)
	cmd = a.addSettings(cmd)
	cmd = a.addDirectories(cmd)

	mcpCmd, err := a.addMCPServers(cmd)
	if err != nil {
		return nil, err
	}
	cmd = mcpCmd

	cmd = a.addExtraArgs(cmd)

	return cmd, nil
}

// addSystemPrompt adds system prompt flags to the command.
func (a *Adapter) addSystemPrompt(cmd []string) []string {
	if a.options.SystemPrompt == nil {
		return cmd
	}

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

func (a *Adapter) addToolOptions(cmd []string) []string {
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
		cmd = append(
			cmd,
			"--disallowedTools",
			strings.Join(tools, ","),
		)
	}

	return cmd
}

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

	return cmd
}

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

func (a *Adapter) addSession(cmd []string) []string {
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

func (a *Adapter) addDirectories(cmd []string) []string {
	for _, dir := range a.options.AddDirs {
		cmd = append(cmd, "--add-dir", dir)
	}

	return cmd
}

func (a *Adapter) addMCPServers(cmd []string) ([]string, error) {
	if len(a.options.MCPServers) == 0 {
		return cmd, nil
	}

	mcpConfig := map[string]any{"mcpServers": a.options.MCPServers}
	jsonBytes, err := json.Marshal(mcpConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MCP config: %w", err)
	}

	return append(cmd, "--mcp-config", string(jsonBytes)), nil
}

func (a *Adapter) addExtraArgs(cmd []string) []string {
	for flag, value := range a.options.ExtraArgs {
		if value == nil {
			cmd = append(cmd, "--"+flag)
		} else {
			cmd = append(cmd, "--"+flag, *value)
		}
	}

	return cmd
}
