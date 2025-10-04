// Package messages provides domain models for Claude Agent SDK messages.
//
// This package contains typed representations of messages exchanged between
// the SDK and Claude CLI, following hexagonal architecture principles with
// no infrastructure dependencies.
package messages

// Message is the base interface for all message types.
//
// Messages represent communication between the SDK and Claude CLI:
//   - UserMessage: User input to Claude
//   - AssistantMessage: Claude's response
//   - SystemMessage: System notifications and state
//   - ResultMessage: Query execution results
//   - StreamEvent: Real-time API events
type Message interface {
	message()
}
