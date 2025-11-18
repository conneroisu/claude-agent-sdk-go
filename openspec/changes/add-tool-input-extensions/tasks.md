# Implementation Tasks

## 1. AgentInput Type Extensions
- [ ] 1.1 Add `Model` field to `AgentInput` struct (string with values: "sonnet", "opus", "haiku", "inherit")
- [ ] 1.2 Add `Resume` field to `AgentInput` struct (string)
- [ ] 1.3 Update JSON struct tags to use camelCase
- [ ] 1.4 Add godoc comments explaining model selection and resume functionality

## 2. BashInput Type Extensions
- [ ] 2.1 Add `DangerouslyDisableSandbox` field to `BashInput` struct (bool)
- [ ] 2.2 Add JSON struct tag `json:"dangerouslyDisableSandbox,omitempty"`
- [ ] 2.3 Add godoc comment with security warning about sandbox bypass

## 3. TimeMachineInput Type
- [ ] 3.1 Create `TimeMachineInput` struct in `pkg/claude/tool_inputs.go` with fields:
  - MessagePrefix: string (JSON: "message_prefix")
  - CourseCorrection: string (JSON: "course_correction")
  - RestoreCode: *bool (JSON: "restore_code", optional)
- [ ] 3.2 Add validation to ensure required fields are non-empty
- [ ] 3.3 Add godoc comment explaining time machine functionality
- [ ] 3.4 Document message_prefix format and course_correction semantics

## 4. AskUserQuestionInput Type
- [ ] 4.1 Create `AskUserQuestionInput` struct in `pkg/claude/tool_inputs.go` with fields:
  - Questions: []QuestionDefinition (complex nested structure)
  - Answers: map[string]string (JSON: "answers", optional)
- [ ] 4.2 Create `QuestionDefinition` struct with fields:
  - Question: string
  - Header: string
  - Options: []QuestionOption
  - MultiSelect: bool
- [ ] 4.3 Create `QuestionOption` struct with fields:
  - Label: string
  - Description: string
- [ ] 4.4 Implement validation for question structure (non-empty arrays, required fields)
- [ ] 4.5 Implement UnmarshalJSON for complex nested structure
- [ ] 4.6 Add godoc comments explaining interactive question usage

## 5. JSON Marshaling
- [ ] 5.1 Verify all new fields have correct JSON struct tags with camelCase
- [ ] 5.2 Test JSON marshaling/unmarshaling for AgentInput with new fields
- [ ] 5.3 Test JSON marshaling for BashInput with sandbox field
- [ ] 5.4 Test JSON marshaling/unmarshaling for TimeMachineInput
- [ ] 5.5 Test JSON marshaling/unmarshaling for AskUserQuestionInput with complex structure

## 6. Testing
- [ ] 6.1 Write unit tests for AgentInput marshaling with Model field
- [ ] 6.2 Write unit tests for AgentInput marshaling with Resume field
- [ ] 6.3 Write unit tests for BashInput with DangerouslyDisableSandbox
- [ ] 6.4 Write unit tests for TimeMachineInput marshaling
- [ ] 6.5 Write unit tests for TimeMachineInput validation
- [ ] 6.6 Write unit tests for AskUserQuestionInput marshaling
- [ ] 6.7 Write unit tests for AskUserQuestionInput with answers
- [ ] 6.8 Test edge cases: nil values, empty arrays, complex nested structures

## 7. Documentation & Comments
- [ ] 7.1 Document AgentInput.Model: valid values and default behavior
- [ ] 7.2 Document AgentInput.Resume: when and how to use for resuming subagents
- [ ] 7.3 Document BashInput.DangerouslyDisableSandbox: security implications
- [ ] 7.4 Document TimeMachineInput: message prefix format, course correction semantics
- [ ] 7.5 Document AskUserQuestionInput: question structure, option format, answer mapping
- [ ] 7.6 Add examples of each tool input type

## 8. Integration & Validation
- [ ] 8.1 Run `go test ./...` to verify all tests pass
- [ ] 8.2 Run `golangci-lint run` to verify code quality
- [ ] 8.3 Cross-reference with TypeScript SDK tool input types
- [ ] 8.4 Verify JSON serialization matches TypeScript SDK format exactly
- [ ] 8.5 Manual testing: create and marshal each tool input type

