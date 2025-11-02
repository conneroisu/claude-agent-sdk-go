# Implementation Tasks

## 1. Core Implementation
- [ ] 1.1 Add `DisallowedTools []string` field to `AgentDefinition` struct in `pkg/claude/options.go`
- [ ] 1.2 Add appropriate JSON struct tag: `json:"disallowedTools,omitempty"`
- [ ] 1.3 Verify field ordering matches TypeScript SDK for consistency

## 2. Documentation
- [ ] 2.1 Add godoc comment explaining the `DisallowedTools` field
- [ ] 2.2 Document interaction between `Tools` and `DisallowedTools` (precedence, mutual exclusivity)
- [ ] 2.3 Update README.md if it contains agent definition examples

## 3. Testing
- [ ] 3.1 Write unit tests for agent definition marshaling with disallowedTools
- [ ] 3.2 Add integration test that creates agent with disallowedTools and verifies CLI receives it
- [ ] 3.3 Test edge cases (empty array, nil, tools + disallowedTools specified together)

## 4. Validation
- [ ] 4.1 Run linter: `golangci-lint run`
- [ ] 4.2 Run all tests: `go test ./...`
- [ ] 4.3 Verify TypeScript SDK parity by comparing struct fields with sdk.d.ts
- [ ] 4.4 Manual testing with a sample agent that uses disallowedTools
