# Implementation Tasks

## 1. AgentInput Type Extensions
- [x] 1.1 Add `Model` field to `AgentInput` struct (string with values: "sonnet", "opus", "haiku", "inherit")
- [x] 1.2 Add `Resume` field to `AgentInput` struct (string)
- [x] 1.3 Update JSON struct tags to use camelCase
- [x] 1.4 Add godoc comments explaining model selection and resume functionality

## 2. BashInput Type Extensions
- [x] 2.1 Add `DangerouslyDisableSandbox` field to `BashInput` struct (bool)
- [x] 2.2 Add JSON struct tag `json:"dangerouslyDisableSandbox,omitempty"`
- [x] 2.3 Add godoc comment with security warning about sandbox bypass

## 3. TimeMachineInput Type
- [x] 3.1 Create `TimeMachineInput` struct in `pkg/claude/tool_inputs.go` with fields:
  - MessagePrefix: string (JSON: "message_prefix")
  - CourseCorrection: string (JSON: "course_correction")
  - RestoreCode: *bool (JSON: "restore_code", optional)
- [x] 3.2 Add validation to ensure required fields are non-empty
- [x] 3.3 Add godoc comment explaining time machine functionality
- [x] 3.4 Document message_prefix format and course_correction semantics

## 4. AskUserQuestionInput Type
- [x] 4.1 Create `AskUserQuestionInput` struct in `pkg/claude/tool_inputs.go` with fields:
  - Questions: []QuestionDefinition (complex nested structure)
  - Answers: map[string]string (JSON: "answers", optional)
- [x] 4.2 Create `QuestionDefinition` struct with fields:
  - Question: string
  - Header: string
  - Options: []QuestionOption
  - MultiSelect: bool
- [x] 4.3 Create `QuestionOption` struct with fields:
  - Label: string
  - Description: string
- [x] 4.4 Implement validation for question structure (non-empty arrays, required fields)
- [x] 4.5 Implement UnmarshalJSON for complex nested structure
- [x] 4.6 Add godoc comments explaining interactive question usage

## 5. JSON Marshaling
- [x] 5.1 Verify all new fields have correct JSON struct tags with camelCase
- [x] 5.2 Test JSON marshaling/unmarshaling for AgentInput with new fields
- [x] 5.3 Test JSON marshaling for BashInput with sandbox field
- [x] 5.4 Test JSON marshaling/unmarshaling for TimeMachineInput
- [x] 5.5 Test JSON marshaling/unmarshaling for AskUserQuestionInput with complex structure

## 6. Testing
- [x] 6.1 Write unit tests for AgentInput marshaling with Model field
- [x] 6.2 Write unit tests for AgentInput marshaling with Resume field
- [x] 6.3 Write unit tests for BashInput with DangerouslyDisableSandbox
- [x] 6.4 Write unit tests for TimeMachineInput marshaling
- [x] 6.5 Write unit tests for TimeMachineInput validation
- [x] 6.6 Write unit tests for AskUserQuestionInput marshaling
- [x] 6.7 Write unit tests for AskUserQuestionInput with answers
- [x] 6.8 Test edge cases: nil values, empty arrays, complex nested structures

## 7. Documentation & Comments
- [x] 7.1 Document AgentInput.Model: valid values and default behavior
- [x] 7.2 Document AgentInput.Resume: when and how to use for resuming subagents
- [x] 7.3 Document BashInput.DangerouslyDisableSandbox: security implications
- [x] 7.4 Document TimeMachineInput: message prefix format, course correction semantics
- [x] 7.5 Document AskUserQuestionInput: question structure, option format, answer mapping
- [x] 7.6 Add examples of each tool input type

## 8. Integration & Validation
- [x] 8.1 Run `go test ./...` to verify all tests pass
- [x] 8.2 Run `golangci-lint run` to verify code quality
- [x] 8.3 Cross-reference with TypeScript SDK tool input types
- [x] 8.4 Verify JSON serialization matches TypeScript SDK format exactly
- [x] 8.5 Manual testing: create and marshal each tool input type

