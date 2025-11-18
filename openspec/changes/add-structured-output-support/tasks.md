# Implementation Tasks

## 1. Result Message Updates
- [x] 1.1 Add `StructuredOutput` field to `SDKResultMessage` struct (type: interface{}) with JSON tag `json:"structured_output,omitempty"`
- [x] 1.2 Add `Errors` field to `SDKResultMessage` struct (type: []string) with JSON tag `json:"errors,omitempty"`
- [x] 1.3 Verify field placement in struct for consistency with TypeScript SDK
- [x] 1.4 Add godoc comments explaining when these fields are populated

## 2. Result Subtypes
- [x] 2.1 Add `ResultSubtypeErrorMaxBudgetUsd` constant to result subtypes with value "error_max_budget_usd"
- [x] 2.2 Add `ResultSubtypeErrorMaxStructuredOutputRetries` constant with value "error_max_structured_output_retries"
- [x] 2.3 Update result subtype documentation to include these new error types
- [x] 2.4 Verify constants are used in proper location (likely pkg/claude/messages.go)

## 3. Testing
- [x] 3.1 Write unit tests for SDKResultMessage with StructuredOutput field
- [x] 3.2 Write unit tests for error subtype constants
- [x] 3.3 Test JSON marshaling/unmarshaling with structured output data
- [x] 3.4 Test JSON marshaling with errors array
- [x] 3.5 Test edge cases: nil structured output, empty errors array, complex output structures

## 4. Documentation & Comments
- [x] 4.1 Document StructuredOutput field: when it's populated, data format expectations
- [x] 4.2 Document Errors field: when populated, error message format
- [x] 4.3 Document error subtypes: what triggers each error, implications
- [x] 4.4 Add examples of structured output result messages (covered by comprehensive tests)

## 5. Integration & Validation
- [x] 5.1 Run `go test ./...` to verify all tests pass
- [x] 5.2 Run `golangci-lint run` to verify code quality
- [x] 5.3 Cross-reference with TypeScript SDK result message format
- [x] 5.4 Verify JSON serialization matches TypeScript SDK format
- [x] 5.5 Manual testing: parse result messages with structured output

