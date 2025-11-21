# tool-inputs Specification

## MODIFIED Requirements

### Requirement: AgentInput Type Complete Definition
The AgentInput type SHALL include model selection and resume functionality.

#### Scenario: Create subagent with specific model
- **GIVEN** an AgentInput for creating a subagent
- **WHEN** the Model field is set to "opus"
- **THEN** the subagent uses Claude Opus
- **AND** JSON serialization includes `"model": "opus"`

#### Scenario: Inherit model from parent
- **GIVEN** an AgentInput with Model="inherit"
- **WHEN** the subagent is created
- **THEN** it inherits the parent agent's model
- **AND** no explicit model override occurs

#### Scenario: Resume subagent execution
- **GIVEN** an AgentInput with Resume field set
- **WHEN** the subagent is invoked
- **THEN** it resumes from the specified checkpoint
- **AND** execution state is restored

#### Scenario: Default model
- **GIVEN** an AgentInput without Model specified
- **WHEN** marshaled to JSON
- **THEN** the model field is omitted
- **AND** the default model is used

---

### Requirement: BashInput Type with Sandbox Control
The BashInput type SHALL support disabling sandbox restrictions.

#### Scenario: Bash with sandbox enabled (default)
- **GIVEN** a BashInput without DangerouslyDisableSandbox
- **WHEN** the command executes
- **THEN** standard sandbox restrictions apply
- **AND** access is limited to safe directories

#### Scenario: Bash with sandbox disabled
- **GIVEN** a BashInput with DangerouslyDisableSandbox=true
- **WHEN** the command executes
- **THEN** sandbox restrictions are bypassed
- **AND** full system access is possible
- **AND** documentation warns of security implications

#### Scenario: Security warning in documentation
- **WHEN** reviewing BashInput documentation
- **THEN** it clearly warns about security implications
- **AND** recommends only using in controlled environments
- **AND** notes that this bypasses important safety measures

---

## ADDED Requirements

### Requirement: TimeMachineInput Type
The system SHALL provide `TimeMachineInput` for time travel operations.

#### Scenario: Rewind messages with course correction
- **GIVEN** a TimeMachineInput with message_prefix and course_correction
- **WHEN** the time machine is invoked
- **THEN** previous messages up to message_prefix are removed
- **AND** the course_correction message is added to guide the agent
- **AND** execution continues from this new state

#### Scenario: Optional code restoration
- **GIVEN** a TimeMachineInput with restore_code=true
- **WHEN** time travel occurs
- **THEN** the code state is also restored to the checkpoint
- **AND** not just messages are rewound

#### Scenario: Code restoration default
- **GIVEN** a TimeMachineInput without restore_code
- **WHEN** marshaled to JSON
- **THEN** the restore_code field is omitted
- **AND** default behavior applies

---

### Requirement: AskUserQuestionInput Type
The system SHALL provide `AskUserQuestionInput` for interactive user prompts.

#### Scenario: Single select question
- **GIVEN** an AskUserQuestionInput with a question
- **WHEN** multiSelect=false
- **THEN** the user selects exactly one option
- **AND** the answer is a string in the map

#### Scenario: Multiple select question
- **GIVEN** an AskUserQuestionInput with multiSelect=true
- **WHEN** the question is presented
- **THEN** the user can select multiple options
- **AND** answers are provided as comma-separated values or arrays

#### Scenario: Question with options
- **GIVEN** a question with 2-4 options
- **WHEN** displayed to user
- **THEN** each option has label and description
- **AND** user selects from these choices

#### Scenario: Pre-filled answers
- **GIVEN** an AskUserQuestionInput with answers map populated
- **WHEN** the question is processed
- **THEN** the provided answers are used
- **AND** user is not prompted if answers are complete

#### Scenario: Empty answers map
- **GIVEN** an AskUserQuestionInput with empty answers
- **WHEN** marshaled to JSON
- **THEN** the answers field is omitted or empty
- **AND** user is prompted to provide answers

#### Scenario: Question structure validation
- **GIVEN** an AskUserQuestionInput
- **WHEN** created
- **THEN** questions array is non-empty
- **AND** each question has required fields
- **AND** options are within valid range (2-4)

---

