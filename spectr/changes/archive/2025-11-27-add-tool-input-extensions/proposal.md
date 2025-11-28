# Add Tool Input Type Extensions

## Why

The TypeScript SDK has defined complete tool input types for agent invocation, bash commands, time machine operations, and user questions - features that allow agents to call other agents with specific parameters, rewind time, and interact with users. The Go SDK currently lacks:

- Complete `AgentInput` type with model and resume fields
- `BashInput.DangerouslyDisableSandbox` option for security-aware bash execution
- Complete `TimeMachineInput` type for message history manipulation
- Complete `AskUserQuestionInput` type for interactive user prompts

These tool input extensions prevent Go users from leveraging advanced agent capabilities and interactive features available in the TypeScript SDK.

## What Changes

- Add `Model` and `Resume` fields to `AgentInput` type for agent model selection and resumption
- Add `DangerouslyDisableSandbox` field to `BashInput` for sandbox control
- Add complete `TimeMachineInput` type with message_prefix, course_correction, and optional restore_code
- Add complete `AskUserQuestionInput` type with questions array and optional answers map

## Impact

- **Affected specs**:
  - `tool-inputs` (enhanced capability)
- **Affected code**:
  - `pkg/claude/tool_inputs.go` - Tool input type definitions
- **Breaking changes**: None - additions are backward compatible
- **TypeScript SDK parity**: Achieves full parity with TypeScript SDK tool input types
- **Dependencies**: None - independent of other proposals

