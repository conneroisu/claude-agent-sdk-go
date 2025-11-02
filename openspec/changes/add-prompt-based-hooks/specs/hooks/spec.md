# Spec: Hooks Configuration

## ADDED Requirements

### Requirement: Support Prompt-Based Hooks

The SDK MUST support prompt-based hooks that send prompts to Claude instead of executing shell commands.

#### Scenario: User configures a Stop hook with a prompt

**Given** a user wants to use AI to generate Stop hook feedback
**When** they configure a Stop hook with `type: "prompt"` and `prompt: "Analyze the current session state..."`
**Then** the SDK sends the prompt configuration to the Claude Code process
**And** the process executes the prompt and returns AI-generated feedback

#### Scenario: Prompt hook for PreToolUse permission check

**Given** a user wants AI-driven permission decisions
**When** they configure a PreToolUse hook with a prompt like "Should we allow this tool use? Analyze: {tool_name}"
**Then** the SDK sends the prompt to Claude
**And** Claude analyzes the context and returns a permission decision
**And** the decision is applied to the tool use

#### Scenario: Timeout configuration for prompt hooks

**Given** a prompt hook that might take longer than the default timeout
**When** the user configures `timeout: 120` for the prompt hook
**Then** the SDK waits up to 120 seconds for the prompt response
**And** returns a timeout error if the prompt exceeds this duration

### Requirement: Maintain Backward Compatibility

The SDK MUST maintain backward compatibility with existing command and callback hooks.

#### Scenario: Existing command hooks still work

**Given** a user has command hooks configured with `command: "echo test"`
**When** they upgrade to the new SDK version
**Then** their command hooks continue to work without modification
**And** no configuration changes are required

#### Scenario: Mix of hook types

**Given** a user configures multiple hook types for Stop event
**When** they have command hooks, prompt hooks, and callback hooks
**Then** all three types execute in the order defined
**And** each receives the same hook input
**And** all responses are collected and processed

### Requirement: Prompt Hook Configuration Structure

Prompt hooks MUST follow a consistent configuration structure matching the TypeScript SDK.

#### Scenario: Prompt hook has required fields

**Given** a user configures a prompt hook
**When** they provide `type: "prompt"` and `prompt: "some prompt text"`
**Then** the configuration is valid
**And** the SDK can serialize it correctly

#### Scenario: Prompt hook with optional timeout

**Given** a user wants to customize the prompt timeout
**When** they configure `type: "prompt"`, `prompt: "text"`, and `timeout: 90`
**Then** the configuration includes the custom timeout
**And** the SDK uses 90 seconds instead of the default

#### Scenario: Invalid prompt hook missing required field

**Given** a user configures a hook with `type: "prompt"` but no `prompt` field
**When** the SDK validates the configuration
**Then** an error is returned indicating the missing `prompt` field
**And** the SDK does not start

## MODIFIED Requirements

None - this is an additive change.

## REMOVED Requirements

None - no functionality is being removed.
