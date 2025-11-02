# Implementation Tasks

## 1. Core Implementation
- [x] 1.1 Add `DisallowedTools []string` field to `AgentDefinition` struct in `pkg/claude/options.go`
- [x] 1.2 Add appropriate JSON struct tag: `json:"disallowedTools,omitempty"`
- [x] 1.3 Verify field ordering matches TypeScript SDK for consistency

## 2. Documentation
- [x] 2.1 Add godoc comment explaining the `DisallowedTools` field
- [x] 2.2 Document interaction between `Tools` and `DisallowedTools` (precedence, mutual exclusivity)
- [x] 2.3 Update README.md if it contains agent definition examples

## 3. Testing
- [x] 3.1 Write unit tests for agent definition marshaling with disallowedTools
- [x] 3.2 Add integration test that creates agent with disallowedTools and verifies CLI receives it
- [x] 3.3 Test edge cases (empty array, nil, tools + disallowedTools specified together)

## 4. Validation
- [x] 4.1 Run linter: `golangci-lint run`
- [x] 4.2 Run all tests: `go test ./...`
- [x] 4.3 Verify TypeScript SDK parity by comparing struct fields with sdk.d.ts
- [x] 4.4 Manual testing with a sample agent that uses disallowedTools
