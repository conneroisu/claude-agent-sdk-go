package claude

// This file provides tool input types for Claude agent operations.
//
// Additionally, this file defines input structures for all built-in Claude
// Code tools including:
//	- Agent
//	- Bash
//	- File operations (Read/Edit/Write)
//	- Glob
//	- Grep
//	- TodoWrite
//	- WebSearch
//	- WebFetch
//	- NotebookEdit
//	- Shell management (KillShell/BashOutput)
//	- MCP resources
//	- SlashCommand
//	- PlanMode
//
// The high number of public structs is intentional to
// support the comprehensive Claude Code tool ecosystem.

// ToolInput is the interface all tool inputs implement.
type ToolInput interface {
	toolInput()
}

// AgentInput represents Agent tool input.
type AgentInput struct {
	Description  string `json:"description"`
	Prompt       string `json:"prompt"`
	SubagentType string `json:"subagent_type"`
}

func (AgentInput) toolInput() {}

// BashInput represents Bash tool input.
type BashInput struct {
	Command         string  `json:"command"`
	Timeout         *int    `json:"timeout,omitempty"`
	Description     *string `json:"description,omitempty"`
	RunInBackground *bool   `json:"run_in_background,omitempty"`
}

func (BashInput) toolInput() {}

// FileReadInput represents Read tool input.
type FileReadInput struct {
	FilePath string `json:"file_path"`
	Offset   *int   `json:"offset,omitempty"`
	Limit    *int   `json:"limit,omitempty"`
}

func (FileReadInput) toolInput() {}

// FileEditInput represents Edit tool input.
type FileEditInput struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll *bool  `json:"replace_all,omitempty"`
}

func (FileEditInput) toolInput() {}

// FileWriteInput represents Write tool input.
type FileWriteInput struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

func (FileWriteInput) toolInput() {}

// GlobInput represents Glob tool input.
type GlobInput struct {
	Pattern string  `json:"pattern"`
	Path    *string `json:"path,omitempty"`
}

func (GlobInput) toolInput() {}

// GrepInput represents Grep tool input.
type GrepInput struct {
	Pattern    string  `json:"pattern"`
	Path       *string `json:"path,omitempty"`
	Glob       *string `json:"glob,omitempty"`
	OutputMode *string `json:"output_mode,omitempty"`
	B          *int    `json:"-B,omitempty"`
	A          *int    `json:"-A,omitempty"`
	C          *int    `json:"-C,omitempty"`
	N          *bool   `json:"-n,omitempty"`
	I          *bool   `json:"-i,omitempty"`
	Type       *string `json:"type,omitempty"`
	HeadLimit  *int    `json:"head_limit,omitempty"`
	Multiline  *bool   `json:"multiline,omitempty"`
}

func (GrepInput) toolInput() {}

// TodoWriteInput represents TodoWrite tool input.
type TodoWriteInput struct {
	Todos []TodoItem `json:"todos"`
}

type TodoItem struct {
	Content    string `json:"content"`
	Status     string `json:"status"` // "pending", "in_progress", "completed"
	ActiveForm string `json:"activeForm"`
}

func (TodoWriteInput) toolInput() {}

// WebSearchInput represents WebSearch tool input.
type WebSearchInput struct {
	Query          string   `json:"query"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	BlockedDomains []string `json:"blocked_domains,omitempty"`
}

func (WebSearchInput) toolInput() {}

// WebFetchInput represents WebFetch tool input.
type WebFetchInput struct {
	URL    string `json:"url"`
	Prompt string `json:"prompt"`
}

func (WebFetchInput) toolInput() {}

// NotebookEditInput represents NotebookEdit tool input.
type NotebookEditInput struct {
	NotebookPath string  `json:"notebook_path"`
	CellID       *string `json:"cell_id,omitempty"`
	NewSource    string  `json:"new_source"`
	CellType     *string `json:"cell_type,omitempty"`
	EditMode     *string `json:"edit_mode,omitempty"`
}

func (NotebookEditInput) toolInput() {}

// KillShellInput represents KillShell tool input.
type KillShellInput struct {
	ShellID string `json:"shell_id"`
}

func (KillShellInput) toolInput() {}

// BashOutputInput represents BashOutput tool input.
type BashOutputInput struct {
	BashID string  `json:"bash_id"`
	Filter *string `json:"filter,omitempty"`
}

func (BashOutputInput) toolInput() {}

// ReadMcpResourceInput represents ReadMcpResource tool input.
type ReadMcpResourceInput struct {
	Server string `json:"server"`
	URI    string `json:"uri"`
}

func (ReadMcpResourceInput) toolInput() {}

// ListMcpResourcesInput represents ListMcpResources tool input.
type ListMcpResourcesInput struct {
	Server *string `json:"server,omitempty"`
}

func (ListMcpResourcesInput) toolInput() {}

// SlashCommandInput represents slash command execution.
type SlashCommandInput struct {
	Command string `json:"command"`
}

func (SlashCommandInput) toolInput() {}

// ExitPlanModeInput represents exiting plan mode.
type ExitPlanModeInput struct {
	// Plan is the plan to present to the user for approval
	// (supports markdown).
	Plan string `json:"plan"`
}

func (ExitPlanModeInput) toolInput() {}

// MCPInput represents generic MCP tool input.
type MCPInput map[string]JSONValue

func (MCPInput) toolInput() {}
