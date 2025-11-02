# Tasks: Add Prompt-Based Hook Support

## Implementation Tasks

### 1. Update Hook Configuration Types
- [ ] Add `PromptHookConfig` struct in `pkg/claude/types.go`
- [ ] Add union type for hook configs (command, prompt, callback)
- [ ] Update `Options` struct to accept new hook type
- [ ] Ensure JSON marshaling handles the union type correctly

### 2. Update Hook Processing
- [ ] Modify hook execution to detect prompt type
- [ ] Send prompt hooks to process via control messages
- [ ] Handle prompt hook responses

### 3. Add Examples
- [ ] Create `examples/prompt-hooks/main.go` showing prompt hook usage
- [ ] Add prompt hook examples to existing `examples/hooks/main.go`
- [ ] Document Stop hook with prompts specifically

### 4. Add Tests
- [ ] Add unit tests for prompt hook configuration
- [ ] Add integration tests for prompt hook execution
- [ ] Test all hook events with prompts (PreToolUse, Stop, etc.)
- [ ] Test timeout handling for prompt hooks

### 5. Update Documentation
- [ ] Update README with prompt hook examples
- [ ] Add godoc comments for new types
- [ ] Update hooks documentation

## Validation Tasks

### 6. Validate Implementation
- [ ] Run existing tests to ensure no regression
- [ ] Run new tests to verify prompt hooks work
- [ ] Manually test with Claude Code CLI
- [ ] Verify with OpenSpec validation

## Estimated Complexity

- **Overall**: Medium
- **Time**: 4-6 hours
- **Risk**: Low (additive change, backward compatible)

## Dependencies

- No blocking dependencies
- TypeScript SDK reference available

## Definition of Done

- All checkboxes above are checked
- `openspec validate add-prompt-based-hooks --strict` passes
- All Go tests pass (`go test ./...`)
- Examples run without errors
- Documentation is complete and accurate
