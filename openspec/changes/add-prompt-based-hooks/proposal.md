# Proposal: Add Prompt-Based Hook Support

## Overview

Add support for **prompt-based hooks** to the Go SDK. The Claude Code CLI and TypeScript SDK just added support for hooks that execute prompts instead of shell commands. This allows hooks to send structured prompts to Claude, enabling more sophisticated hook behaviors that leverage AI instead of just running shell commands.

## Motivation

Currently, the Go SDK supports two types of hooks:
1. **Command hooks** - Execute shell commands
2. **Callback hooks** - Execute Go functions

The TypeScript SDK recently added a third type:
3. **Prompt hooks** - Send prompts to Claude for AI-powered hook responses

This feature enables use cases like:
- **AI-driven decision making** - Let Claude analyze hook inputs and decide on actions
- **Dynamic permission prompts** - Generate contextual permission requests
- **Intelligent stop hooks** - AI can analyze the current state and provide feedback
- **Smart validation** - Use Claude to validate complex inputs before tool execution

Without this feature, the Go SDK is missing critical functionality available in the TypeScript SDK.

## Current State

### TypeScript SDK Hook Types

Looking at the TypeScript CLI code, hooks can now be:

```typescript
type HookConfig =
  | { type: "command", command: string, timeout?: number }
  | { type: "prompt", prompt: string, timeout?: number }  // NEW!
  | { type: "callback", callback: Function }
```

The prompt execution path is in the CLI:
```javascript
async function ce1(A,B,Q,I,G,Z,Y){
  if(G.aborted)return{stdout:"",stderr:"Operation cancelled",status:1,aborted:!0};
  if(A.type==="prompt")return Td2(A,I,G,Z);  // Handle prompt hooks
  else return xl5(A,B,Q,I,G,Y);  // Handle command hooks
}
```

### Go SDK Current State

The Go SDK has the Stop and SubagentStop hooks defined (in `pkg/claude/hooks.go`):

```go
type StopHookInput struct {
	BaseHookInput
	HookEventName  HookEvent `json:"hook_event_name"`
	StopHookActive bool      `json:"stop_hook_active"`
}

type SubagentStopHookInput struct {
	BaseHookInput
	HookEventName  HookEvent `json:"hook_event_name"`
	StopHookActive bool      `json:"stop_hook_active"`
}
```

But the Options struct only supports command and callback hooks - no prompt hooks.

## Proposed Changes

1. **Add prompt hook support to Options struct** - Allow users to define hooks with prompts
2. **Update hook execution** - Send prompt-based hooks to the process for execution
3. **Add examples** - Show how to use prompt-based hooks
4. **Add tests** - Verify prompt hooks work correctly

## Dependencies

- TypeScript SDK definitions (for reference)
- No external dependencies needed

## Risks & Mitigation

- **Risk**: Breaking changes to hook configuration format
  - **Mitigation**: Add new fields without removing old ones; maintain backward compatibility

- **Risk**: Prompt hooks require the process to support them
  - **Mitigation**: The CLI already supports this; we just need to pass the config correctly

## Open Questions

1. Should we support prompt hooks for all hook events or just some?
   - **Answer**: Support for all hook events (Pre/Post tool use, SessionStart, etc.)

2. How should timeouts work for prompt hooks vs command hooks?
   - **Answer**: Use the same timeout mechanism; prompts might take longer so allow configuration

3. Should we validate prompt content before sending?
   - **Answer**: Basic validation (non-empty), but let the process handle complex validation

## Success Criteria

- [ ] Users can configure prompt-based hooks in Go
- [ ] Prompt hooks are correctly passed to the Claude Code process
- [ ] Examples demonstrate prompt hook usage
- [ ] Tests verify prompt hooks execute correctly
- [ ] Documentation updated to explain the feature
- [ ] No breaking changes to existing hook APIs
