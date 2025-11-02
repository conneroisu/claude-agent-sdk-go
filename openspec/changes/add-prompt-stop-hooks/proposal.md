# Proposal: Add Prompt-Based Stop Hook Support

## Overview

Add support for **prompt-based Stop and SubagentStop hooks** to the Go SDK. The Claude Code CLI and TypeScript SDK recently added support for hooks that execute AI prompts instead of just shell commands or callbacks. This proposal focuses specifically on Stop hooks, which are critical control points for agent execution flow.

## Motivation

Stop hooks are uniquely important because they fire at critical moments:
- **Stop**: When the main agent is about to stop execution
- **SubagentStop**: When a subagent (like the coder, tester, stuck agent) is about to stop

These hooks have unique characteristics not present in other hook types:
1. **`stop_hook_active` field**: Indicates whether a stop hook is currently active
2. **Execution control**: Can potentially influence whether stopping proceeds
3. **Session analysis**: Perfect opportunity for AI to analyze session state before termination
4. **Feedback generation**: AI can generate meaningful feedback about what was accomplished

### Current State

The Go SDK currently:
- ✅ Has `StopHookInput` and `SubagentStopHookInput` types defined
- ✅ Supports callback hooks (Go functions)
- ❌ Does NOT support command hooks (shell commands)
- ❌ Does NOT support prompt hooks (AI prompts)
- ❌ Has NO examples of Stop hook usage

### TypeScript SDK Reference

The TypeScript SDK supports three hook types:

```typescript
type HookConfig =
  | { type: "command", command: string, timeout?: number }
  | { type: "prompt", prompt: string, timeout?: number }  // NEW!
  | { type: "callback", callback: Function }
```

For Stop hooks specifically:

```typescript
export type StopHookInput = BaseHookInput & {
    hook_event_name: 'Stop';
    stop_hook_active: boolean;
};

export type SubagentStopHookInput = BaseHookInput & {
    hook_event_name: 'SubagentStop';
    stop_hook_active: boolean;
};
```

## Proposed Changes

### 1. Core Hook Configuration Types

Add support for command and prompt hook configurations:

```go
// HookConfig represents a hook configuration (command, prompt, or callback)
type HookConfig interface {
    hookConfig()
}

// CommandHookConfig executes a shell command
type CommandHookConfig struct {
    Type    string  `json:"type"` // "command"
    Command string  `json:"command"`
    Timeout *int    `json:"timeout,omitempty"` // milliseconds
}

// PromptHookConfig sends a prompt to Claude
type PromptHookConfig struct {
    Type    string  `json:"type"` // "prompt"
    Prompt  string  `json:"prompt"`
    Timeout *int    `json:"timeout,omitempty"` // milliseconds
}

// CallbackHookConfig executes a Go callback (existing behavior)
type CallbackHookConfig struct {
    Type     string       `json:"type"` // "callback"
    Callback HookCallback `json:"-"`
}
```

### 2. Update Options Struct

The `Options.Hooks` field needs to support the new config types:

```go
type Options struct {
    // ... existing fields ...

    // Hooks support command, prompt, and callback types
    Hooks map[HookEvent][]HookConfig
}
```

### 3. Stop Hook Examples

Create examples demonstrating:
- **Stop hook with AI analysis**: AI analyzes the session and provides feedback
- **SubagentStop hook with validation**: AI validates subagent completed its task
- **Stop hook with custom prompts**: Different prompts for different stop scenarios

### 4. Hook Execution Pipeline

Update the hook execution to:
1. Detect hook type (command, prompt, callback)
2. For prompt hooks: Send configuration to Claude Code CLI
3. For command hooks: Execute shell command
4. For callback hooks: Execute Go function (existing behavior)

## Dependencies

- Broader "add-prompt-based-hooks" change (openspec/changes/add-prompt-based-hooks)
  - This proposal can be implemented as part of that change
  - OR implemented independently with the same hook configuration types
- TypeScript SDK type definitions (reference only)
- Claude Code CLI with prompt hook support (already available)

## Risks & Mitigation

### Risk: Breaking Changes to Hook API

**Impact**: Existing code using `Hooks map[HookEvent][]HookCallbackMatcher` would break

**Mitigation**:
1. Maintain backward compatibility by supporting both old and new formats
2. Add deprecation warnings for old format
3. Provide migration guide in documentation

### Risk: Prompt Hooks May Take Longer

**Impact**: Stop hooks with prompts could delay shutdown

**Mitigation**:
1. Support configurable timeouts for prompt hooks
2. Document recommended timeout values
3. Provide default timeout of 30 seconds

### Risk: Complex Configuration Structure

**Impact**: Users may find union types (command|prompt|callback) confusing

**Mitigation**:
1. Provide clear examples for each hook type
2. Add validation errors with helpful messages
3. Document common patterns

## Open Questions

### 1. Should Stop hooks be able to prevent stopping?

**Context**: The `stop_hook_active` field suggests hooks might be able to influence stop behavior

**Options**:
- A) Hooks are observational only (cannot prevent stop)
- B) Hooks can return `decision: "block"` to prevent stopping
- C) Different behavior for Stop vs SubagentStop

**Recommendation**: Start with option A (observational only) for safety, add B later if needed

### 2. How should we handle multiple hook types for the same event?

**Context**: User might configure command, prompt, and callback hooks for Stop event

**Options**:
- A) Execute all hooks in order defined
- B) Only allow one hook type per event
- C) Allow multiple but require explicit ordering

**Recommendation**: Option A - execute all in order, collect all outputs

### 3. What should timeout defaults be for Stop prompt hooks?

**Context**: Stop hooks run at termination, need to balance thoroughness vs. responsiveness

**Options**:
- A) Same as other hooks (e.g., 10 seconds)
- B) Longer timeout for Stop hooks (e.g., 30 seconds)
- C) No timeout (wait indefinitely)

**Recommendation**: Option B - 30 second default, allow customization

### 4. Should prompt hooks receive the full transcript?

**Context**: Stop hooks might benefit from analyzing the full session transcript

**Options**:
- A) Only provide hook input fields (session_id, transcript_path, etc.)
- B) Include transcript content in prompt context
- C) Allow users to specify whether to include transcript

**Recommendation**: Option A initially (users can read transcript_path if needed)

## Success Criteria

- [ ] Users can configure Stop hooks with AI prompts
- [ ] Users can configure SubagentStop hooks with AI prompts
- [ ] Prompt hooks are correctly passed to Claude Code CLI
- [ ] AI-generated feedback is captured and returned
- [ ] Examples demonstrate Stop hook usage with prompts
- [ ] Tests verify Stop prompt hooks execute correctly
- [ ] Documentation explains Stop hook use cases
- [ ] No breaking changes to existing hook APIs
- [ ] Timeout configuration works for prompt hooks
- [ ] `stop_hook_active` field is properly handled

## Timeline

**Estimated Implementation**: 2-3 days

**Phase 1: Core Types** (Day 1)
- Add HookConfig types (Command, Prompt, Callback)
- Update Options struct
- Add validation

**Phase 2: Execution** (Day 1-2)
- Update hook execution pipeline
- Add prompt hook handling
- Add command hook handling

**Phase 3: Examples & Tests** (Day 2-3)
- Create Stop hook examples
- Add integration tests
- Add unit tests

**Phase 4: Documentation** (Day 3)
- Update godoc comments
- Add README examples
- Migration guide

## Related Work

- **add-prompt-based-hooks** (openspec/changes/add-prompt-based-hooks): Broader proposal covering all hook types
- **add-agent-disallowed-tools** (openspec/changes/add-agent-disallowed-tools): Agent configuration work

This proposal can be implemented as part of the broader prompt-based hooks work, or independently with coordination on the shared HookConfig types.
