package claude

import (
	"context"
	"encoding/json"
	"fmt"
)

// ExitReason represents reasons for session termination.
type ExitReason string

const (
	ExitReasonUserInterrupt   ExitReason = "user_interrupt"
	ExitReasonErrorMaxTurns   ExitReason = "error_max_turns"
	ExitReasonErrorDuringExec ExitReason = "error_during_execution"
	ExitReasonComplete        ExitReason = "complete"
	ExitReasonAborted         ExitReason = "aborted"
)

// ExitReasons documents valid exit reasons for session termination.
// These match the TypeScript SDK's EXIT_REASONS constant.
var ExitReasons = []ExitReason{
	ExitReasonUserInterrupt,
	ExitReasonErrorMaxTurns,
	ExitReasonErrorDuringExec,
	ExitReasonComplete,
	ExitReasonAborted,
}

// SessionStartSource enumerates session start origins.
type SessionStartSource string

const (
	SessionStartSourceStartup SessionStartSource = "startup"
	SessionStartSourceResume  SessionStartSource = "resume"
	SessionStartSourceClear   SessionStartSource = "clear"
	SessionStartSourceCompact SessionStartSource = "compact"
)

// CompactTrigger enumerates compaction triggers.
type CompactTrigger string

const (
	CompactTriggerManual CompactTrigger = "manual"
	CompactTriggerAuto   CompactTrigger = "auto"
)

// HookInput represents input to hook callbacks.
type HookInput interface {
	hookInput()
	// EventName returns the hook event type.
	EventName() HookEvent
	// SessionID returns the session identifier.
	SessionID() string
	// TranscriptPath returns the path to the transcript file.
	TranscriptPath() string
	// Cwd returns the current working directory.
	Cwd() string
}

// BaseHookInput contains common hook input fields.
type BaseHookInput struct {
	SessionIDField      string  `json:"session_id"`
	TranscriptPathField string  `json:"transcript_path"`
	CwdField            string  `json:"cwd"`
	PermissionMode      *string `json:"permission_mode,omitempty"`
}

func (b BaseHookInput) SessionID() string      { return b.SessionIDField }
func (b BaseHookInput) TranscriptPath() string { return b.TranscriptPathField }
func (b BaseHookInput) Cwd() string            { return b.CwdField }

// PreToolUseHookInput for PreToolUse event.
type PreToolUseHookInput struct {
	BaseHookInput
	HookEventName HookEvent `json:"hook_event_name"`
	ToolName      string    `json:"tool_name"`
	ToolInput     JSONValue `json:"tool_input"`
	ToolUseID     string    `json:"tool_use_id"`
}

func (PreToolUseHookInput) hookInput()           {}
func (PreToolUseHookInput) EventName() HookEvent { return HookEventPreToolUse }

// PostToolUseHookInput for PostToolUse event.
type PostToolUseHookInput struct {
	BaseHookInput
	HookEventName HookEvent `json:"hook_event_name"`
	ToolName      string    `json:"tool_name"`
	ToolInput     JSONValue `json:"tool_input"`
	ToolResponse  JSONValue `json:"tool_response"`
	ToolUseID     string    `json:"tool_use_id"`
}

func (PostToolUseHookInput) hookInput() {}
func (PostToolUseHookInput) EventName() HookEvent {
	return HookEventPostToolUse
}

// NotificationHookInput for Notification event.
type NotificationHookInput struct {
	BaseHookInput
	HookEventName    HookEvent `json:"hook_event_name"`
	Message          string    `json:"message"`
	Title            *string   `json:"title,omitempty"`
	NotificationType string    `json:"notification_type"`
}

func (NotificationHookInput) hookInput() {}
func (NotificationHookInput) EventName() HookEvent {
	return HookEventNotification
}

// UserPromptSubmitHookInput for UserPromptSubmit event.
type UserPromptSubmitHookInput struct {
	BaseHookInput
	HookEventName HookEvent `json:"hook_event_name"`
	Prompt        string    `json:"prompt"`
}

func (UserPromptSubmitHookInput) hookInput() {}
func (UserPromptSubmitHookInput) EventName() HookEvent {
	return HookEventUserPromptSubmit
}

// SessionStartHookInput for SessionStart event.
type SessionStartHookInput struct {
	BaseHookInput
	HookEventName HookEvent          `json:"hook_event_name"`
	Source        SessionStartSource `json:"source"`
}

func (SessionStartHookInput) hookInput() {}
func (SessionStartHookInput) EventName() HookEvent {
	return HookEventSessionStart
}

// StopHookInput for Stop event.
type StopHookInput struct {
	BaseHookInput
	HookEventName  HookEvent `json:"hook_event_name"`
	StopHookActive bool      `json:"stop_hook_active"`
}

func (StopHookInput) hookInput()           {}
func (StopHookInput) EventName() HookEvent { return HookEventStop }

// SubagentStopHookInput for SubagentStop event.
type SubagentStopHookInput struct {
	BaseHookInput
	HookEventName        HookEvent `json:"hook_event_name"`
	StopHookActive       bool      `json:"stop_hook_active"`
	AgentID              string    `json:"agent_id"`
	AgentTranscriptPath  string    `json:"agent_transcript_path"`
}

func (SubagentStopHookInput) hookInput() {}
func (SubagentStopHookInput) EventName() HookEvent {
	return HookEventSubagentStop
}

// PreCompactHookInput for PreCompact event.
type PreCompactHookInput struct {
	BaseHookInput
	HookEventName      HookEvent      `json:"hook_event_name"`
	Trigger            CompactTrigger `json:"trigger"`
	CustomInstructions *string        `json:"custom_instructions"`
}

func (PreCompactHookInput) hookInput()           {}
func (PreCompactHookInput) EventName() HookEvent { return HookEventPreCompact }

// SessionEndHookInput for SessionEnd event.
type SessionEndHookInput struct {
	BaseHookInput
	HookEventName HookEvent  `json:"hook_event_name"`
	Reason        ExitReason `json:"reason"`
}

func (SessionEndHookInput) hookInput()           {}
func (SessionEndHookInput) EventName() HookEvent { return HookEventSessionEnd }

// PermissionRequestHookInput for PermissionRequest event.
// This hook is triggered when a tool requires permission before execution.
type PermissionRequestHookInput struct {
	BaseHookInput
	HookEventName HookEvent `json:"hook_event_name"`
	ToolName      string    `json:"tool_name"`
	ToolInput     JSONValue `json:"tool_input"`
}

func (PermissionRequestHookInput) hookInput() {}
func (PermissionRequestHookInput) EventName() HookEvent {
	return HookEventPermissionRequest
}

// SubagentStartHookInput for SubagentStart event.
// This hook is triggered when a subagent begins execution.
type SubagentStartHookInput struct {
	BaseHookInput
	HookEventName HookEvent `json:"hook_event_name"`
	AgentID       string    `json:"agent_id"`
	AgentType     string    `json:"agent_type"`
}

func (SubagentStartHookInput) hookInput() {}
func (SubagentStartHookInput) EventName() HookEvent {
	return HookEventSubagentStart
}

// HookJSONOutput represents output from hook callbacks.
type HookJSONOutput interface {
	hookOutput()
}

// AsyncHookOutput indicates async hook processing.
type AsyncHookOutput struct {
	Async        bool `json:"async"`
	AsyncTimeout *int `json:"asyncTimeout,omitempty"`
}

func (AsyncHookOutput) hookOutput() {}

// HookSpecificOutput captures additional payloads that differ per hook.
type HookSpecificOutput interface {
	// EventName returns the hook event type for this output.
	EventName() HookEvent
}

type PermissionDecision string

const (
	// PermissionDecisionAllow is the hook output permission decision if the
	// PreToolUse hook allows the tool use.
	PermissionDecisionAllow PermissionDecision = "allow"
	// PermissionDecisionDeny is the hook output permission decision if the
	// PreToolUse hook denies the use of the tool.
	PermissionDecisionDeny PermissionDecision = "deny"
	// PermissionDecisionAsk is the hook output permission decision if the
	// PreToolUse hook requires the asking of the user.
	PermissionDecisionAsk PermissionDecision = "ask"
)

// PreToolUseHookOutput conveys permission decisions for PreToolUse hooks.
type PreToolUseHookOutput struct {
	HookEventName            HookEvent               `json:"hookEventName"` // "PreToolUse"
	PermissionDecision       *string                 `json:"permissionDecision,omitempty"`
	PermissionDecisionReason *string                 `json:"permissionDecisionReason,omitempty"` //nolint:revive
	UpdatedInput             *map[string]interface{} `json:"updatedInput,omitempty"`
}

func (PreToolUseHookOutput) EventName() HookEvent { return HookEventPreToolUse }

// UserPromptSubmitHookOutput adds extra context for prompt submissions.
type UserPromptSubmitHookOutput struct {
	HookEventName     HookEvent `json:"hookEventName"` // "UserPromptSubmit"
	AdditionalContext *string   `json:"additionalContext,omitempty"`
}

func (UserPromptSubmitHookOutput) EventName() HookEvent {
	return HookEventUserPromptSubmit
}

// SessionStartHookOutput enriches session start events.
type SessionStartHookOutput struct {
	HookEventName     HookEvent `json:"hookEventName"` // "SessionStart"
	AdditionalContext *string   `json:"additionalContext,omitempty"`
}

func (SessionStartHookOutput) EventName() HookEvent {
	return HookEventSessionStart
}

// PostToolUseHookOutput adds tool execution context.
type PostToolUseHookOutput struct {
	HookEventName         HookEvent   `json:"hookEventName"` // "PostToolUse"
	AdditionalContext     *string     `json:"additionalContext,omitempty"`
	UpdatedMCPToolOutput interface{} `json:"updatedMCPToolOutput,omitempty"`
}

func (PostToolUseHookOutput) EventName() HookEvent {
	return HookEventPostToolUse
}

// PermissionRequestDecision represents a decision type for permission requests.
// Implement this interface to create custom permission decisions.
type PermissionRequestDecision interface {
	permissionRequestDecision()
}

// PermissionRequestAllow represents an allow decision for a permission request.
// It allows the tool to execute, optionally with modified input.
type PermissionRequestAllow struct {
	Behavior     string                  `json:"behavior"`
	UpdatedInput *map[string]interface{} `json:"updatedInput,omitempty"`
}

func (PermissionRequestAllow) permissionRequestDecision() {}

// PermissionRequestDeny represents a deny decision for a permission request.
// It blocks the tool execution, optionally with a custom message and interrupt flag.
type PermissionRequestDeny struct {
	Behavior  string  `json:"behavior"`
	Message   *string `json:"message,omitempty"`
	Interrupt *bool   `json:"interrupt,omitempty"`
}

func (PermissionRequestDeny) permissionRequestDecision() {}

// PermissionRequestHookOutput conveys permission decisions for PermissionRequest hooks.
type PermissionRequestHookOutput struct {
	HookEventName HookEvent                 `json:"hookEventName"` // "PermissionRequest"
	Decision      PermissionRequestDecision `json:"decision"`
}

func (PermissionRequestHookOutput) EventName() HookEvent {
	return HookEventPermissionRequest
}

// SubagentStartHookOutput enriches subagent start events.
type SubagentStartHookOutput struct {
	HookEventName     HookEvent `json:"hookEventName"` // "SubagentStart"
	AdditionalContext *string   `json:"additionalContext,omitempty"`
}

func (SubagentStartHookOutput) EventName() HookEvent {
	return HookEventSubagentStart
}

// HookDecision represents the decision made by a hook. It is a
// string.
type HookDecision string

const (
	HookDecisionApprove HookDecision = "approve"
	HookDecisionBlock   HookDecision = "block"
)

// SyncHookOutput represents synchronous hook result.
type SyncHookOutput struct {
	Continue           *bool              `json:"continue,omitempty"`
	SuppressOutput     *bool              `json:"suppressOutput,omitempty"`
	StopReason         *string            `json:"stopReason,omitempty"`
	Decision           *HookDecision      `json:"decision,omitempty"`
	SystemMessage      *string            `json:"systemMessage,omitempty"`
	Reason             *string            `json:"reason,omitempty"`
	HookSpecificOutput HookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

func (SyncHookOutput) hookOutput() {}

// HookCallback is a function called for hook events.
type HookCallback func(
	ctx context.Context,
	input HookInput,
	toolUseID *string,
) (HookJSONOutput, error)

// HookCallbackMatcher matches hooks with optional matcher pattern.
type HookCallbackMatcher struct {
	Matcher *string        `json:"matcher,omitempty"`
	Hooks   []HookCallback `json:"-"`
	// Timeout specifies the maximum duration in milliseconds to wait for hook execution.
	// If not specified, a default timeout applies. A timeout of 0 or negative value
	// will use the default timeout behavior.
	Timeout *int `json:"timeout,omitempty"`
}

// DecodeHookInput decodes a JSON message into the appropriate HookInput type.
func DecodeHookInput(data []byte) (HookInput, error) {
	// First, parse the hook_event_name to determine the type
	var envelope struct {
		HookEventName HookEvent `json:"hook_event_name"`
	}
	err := json.Unmarshal(data, &envelope)
	if err != nil {
		return nil,
			fmt.Errorf("failed to unmarshal hook event envelope: %w", err)
	}
	var input HookInput

	// Decode based on the event type
	switch envelope.HookEventName {
	case HookEventPreToolUse:
		var concrete PreToolUseHookInput
		err = json.Unmarshal(data, &concrete)
		input = concrete
	case HookEventPostToolUse:
		var concrete PostToolUseHookInput
		err = json.Unmarshal(data, &concrete)
		input = concrete
	case HookEventNotification:
		var concrete NotificationHookInput
		err = json.Unmarshal(data, &concrete)
		input = concrete
	case HookEventUserPromptSubmit:
		var concrete UserPromptSubmitHookInput
		err = json.Unmarshal(data, &concrete)
		input = concrete
	case HookEventSessionStart:
		var concrete SessionStartHookInput
		err = json.Unmarshal(data, &concrete)
		input = concrete
	case HookEventSessionEnd:
		var concrete SessionEndHookInput
		err = json.Unmarshal(data, &concrete)
		input = concrete
	case HookEventStop:
		var concrete StopHookInput
		err = json.Unmarshal(data, &concrete)
		input = concrete
	case HookEventSubagentStop:
		var concrete SubagentStopHookInput
		err = json.Unmarshal(data, &concrete)
		input = concrete
	case HookEventPreCompact:
		var concrete PreCompactHookInput
		err = json.Unmarshal(data, &concrete)
		input = concrete
	case HookEventPermissionRequest:
		var concrete PermissionRequestHookInput
		err = json.Unmarshal(data, &concrete)
		input = concrete
	case HookEventSubagentStart:
		var concrete SubagentStartHookInput
		err = json.Unmarshal(data, &concrete)
		input = concrete
	default:
		return nil,
			fmt.Errorf("unknown hook event type: %s", envelope.HookEventName)
	}
	if err != nil {
		return nil,
			fmt.Errorf("failed to unmarshal %s: %w", input.EventName(), err)
	}

	return input, nil
}
