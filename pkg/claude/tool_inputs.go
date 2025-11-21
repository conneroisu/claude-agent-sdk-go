package claude

// This file provides tool input types for Claude agent operations.
//
// Additionally, this file defines input structures for all built-in Claude
// Code tools including:
//	- Agent
//	- AskUserQuestion
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
//	- TimeMachine
//
// The high number of public structs is intentional to
// support the comprehensive Claude Code tool ecosystem.

// ToolInput is the interface all tool inputs implement.
type ToolInput interface {
	toolInput()
}

// AgentInput represents Agent tool input.
//
// The Agent tool allows spawning subagents that run in their own isolated context windows.
// Each subagent can have its own model and can be resumed from previous checkpoints.
type AgentInput struct {
	// Description provides a brief description of what the subagent should do.
	Description string `json:"description"`

	// Prompt contains the detailed instructions and context for the subagent.
	Prompt string `json:"prompt"`

	// SubagentType specifies which type of specialized subagent to invoke (e.g., "coder", "tester", "stuck").
	SubagentType string `json:"subagent_type"`

	// Model specifies which Claude model the subagent should use.
	// Valid values are: "sonnet", "opus", "haiku"
	// If not specified, the subagent inherits the model from its parent agent.
	// This allows for strategic model selection - using faster/cheaper models for simple tasks
	// and more capable models for complex reasoning.
	Model *string `json:"model,omitempty"`

	// Resume enables resuming a subagent from a previous checkpoint.
	// When provided with a checkpoint ID, the subagent will continue from where it left off
	// rather than starting fresh. This is useful for long-running tasks that may be interrupted
	// or for iterative refinement workflows where you want to build on previous agent work.
	Resume *string `json:"resume,omitempty"`
}

func (AgentInput) toolInput() {}

// BashInput represents Bash tool input.
//
// The Bash tool executes shell commands in a sandboxed environment by default.
// Use DangerouslyDisableSandbox to bypass these restrictions when necessary.
type BashInput struct {
	Command         string  `json:"command"`
	Timeout         *int    `json:"timeout,omitempty"`
	Description     *string `json:"description,omitempty"`
	RunInBackground *bool   `json:"run_in_background,omitempty"`

	// DangerouslyDisableSandbox disables sandbox restrictions for this bash command.
	//
	// SECURITY WARNING: Setting this to true bypasses all sandbox protections and allows
	// the command to execute with full system access. This creates significant security
	// risks including:
	//   - Unrestricted file system access
	//   - Network access without restrictions
	//   - Ability to modify system configurations
	//   - Potential for data exfiltration or system compromise
	//
	// This field should ONLY be used in authorized and controlled contexts where you
	// explicitly trust the command being executed. Never enable this for user-provided
	// or dynamically generated commands without proper validation.
	//
	// Default behavior (when false or nil): Commands execute within sandbox restrictions,
	// providing isolation and protection against malicious or unintended operations.
	DangerouslyDisableSandbox *bool `json:"dangerouslyDisableSandbox,omitempty"`
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

// AskUserQuestionInput represents input for prompting users with structured questions.
//
// This tool enables interactive user input through a structured question format where
// each question presents multiple choice options. It's useful for gathering user
// preferences, decisions, or feedback during agent execution.
//
// The Questions field contains 1-4 questions that will be presented to the user.
// Each question must have between 2-4 answer options. Questions can be configured
// for single-select (radio buttons) or multi-select (checkboxes) mode.
//
// The optional Answers field can contain pre-filled answers or will be populated
// with user responses. Answer keys correspond to question headers, and values
// contain the selected option label(s). For multi-select questions, multiple
// labels are comma-separated.
//
// Example usage:
//
//	input := &claude.AskUserQuestionInput{
//	    Questions: []claude.QuestionDefinition{
//	        {
//	            Question: "Which programming language would you like to use?",
//	            Header: "Language",
//	            Options: []claude.QuestionOption{
//	                {
//	                    Label: "Go",
//	                    Description: "Fast, compiled language with excellent concurrency support",
//	                },
//	                {
//	                    Label: "Python",
//	                    Description: "High-level, interpreted language with rich ecosystem",
//	                },
//	                {
//	                    Label: "TypeScript",
//	                    Description: "JavaScript with static typing for safer web development",
//	                },
//	            },
//	            MultiSelect: false,
//	        },
//	    },
//	}
//
// Validation requirements:
//   - Questions array must contain 1-4 items (will fail if empty or more than 4)
//   - Each Question field must be non-empty
//   - Each Header field must be non-empty and max 12 characters
//   - Each Options array must contain 2-4 items
//   - Each option Label must be non-empty
//   - Each option Description must be non-empty
type AskUserQuestionInput struct {
	// Questions contains 1-4 questions to present to the user.
	// Each question must have a non-empty Question text, a Header (max 12 chars),
	// and 2-4 Options with labels and descriptions.
	Questions []QuestionDefinition `json:"questions"`

	// Answers optionally contains pre-filled or collected answers.
	// Keys are question headers, values are selected option label(s).
	// For multi-select questions, multiple labels are comma-separated.
	Answers map[string]string `json:"answers,omitempty"`
}

func (AskUserQuestionInput) toolInput() {}

// QuestionDefinition defines a single question with its answer options.
//
// Each question consists of the question text itself, a short header label
// for identification, and a list of possible answer options. Questions can
// be configured to allow single or multiple selections.
type QuestionDefinition struct {
	// Question is the full text of the question to ask the user.
	// Must be non-empty.
	Question string `json:"question"`

	// Header is a short label for this question (max 12 characters).
	// Used as the key in the Answers map and for display purposes.
	// Must be non-empty.
	Header string `json:"header"`

	// Options contains 2-4 possible answers for this question.
	// Each option must have a label and description.
	Options []QuestionOption `json:"options"`

	// MultiSelect determines whether the user can select multiple options.
	// When false (default), only one option can be selected (radio buttons).
	// When true, multiple options can be selected (checkboxes).
	MultiSelect bool `json:"multiSelect"`
}

// QuestionOption defines a single answer option for a question.
//
// Each option consists of a label (the selectable choice) and a description
// that explains what selecting this option means.
type QuestionOption struct {
	// Label is the display text for this answer option.
	// This is what the user will see and select.
	// Must be non-empty.
	Label string `json:"label"`

	// Description explains what this option means or what will happen if selected.
	// Helps users make informed choices.
	// Must be non-empty.
	Description string `json:"description"`
}

// TimeMachineInput represents time machine rewind operation input.
//
// The time machine tool enables rewinding execution to a previous point in the conversation
// and injecting new instructions (course correction) to change the execution path. This is
// useful for:
//   - Correcting mistakes without losing all progress
//   - Trying different approaches from a known good state
//   - Recovering from errors by going back and taking a different path
//
// The tool works by:
//  1. Finding a message in the conversation history that contains MessagePrefix
//  2. Rewinding the execution state to that point (discarding everything after)
//  3. Injecting the CourseCorrection instructions as new context
//  4. Optionally restoring code changes using file history if RestoreCode is true
//
// Example usage:
//
//	restoreCode := true
//	input := &claude.TimeMachineInput{
//	    MessagePrefix: "Created initial React component",
//	    CourseCorrection: "Use TypeScript instead of JavaScript for better type safety",
//	    RestoreCode: &restoreCode, // Restore files to the state at that message
//	}
//
// Validation requirements:
//   - MessagePrefix must be a non-empty string (will fail if empty)
//   - CourseCorrection must be a non-empty string (will fail if empty)
//   - If MessagePrefix is not found in conversation history, the operation will fail
type TimeMachineInput struct {
	// MessagePrefix is the text to search for in conversation history.
	// The tool will find the message containing this text and rewind to that point.
	// Must be non-empty and should be distinctive enough to identify the target message.
	MessagePrefix string `json:"message_prefix"`

	// CourseCorrection contains new instructions to inject after rewinding.
	// These instructions will guide the execution on a different path from the rewind point.
	// Must be non-empty and should clearly describe the desired changes.
	CourseCorrection string `json:"course_correction"`

	// RestoreCode determines whether to restore code changes using file history.
	// When true, files will be restored to their state at the rewind point.
	// When false or nil, only the conversation context is rewound without reverting file changes.
	// Optional - defaults to false if not specified.
	RestoreCode *bool `json:"restore_code,omitempty"`
}

func (TimeMachineInput) toolInput() {}

// MCPInput represents generic MCP tool input.
type MCPInput map[string]JSONValue

func (MCPInput) toolInput() {}
