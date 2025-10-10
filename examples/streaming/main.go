// Package main demonstrates streaming responses with Claude Agent SDK.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/conneroisu/claude-agent-sdk-go/pkg/claude"
)

const (
	// maxTurns defines the maximum number of conversation turns.
	maxTurns = 5
	// separatorLength defines the length of separator lines.
	separatorLength = 60
	// spinnerTickInterval defines the spinner update interval.
	spinnerTickInterval = 100 * time.Millisecond
	// clearLineLength defines the length for clearing spinner line.
	clearLineLength = 20
)

func main() {
	ctx := context.Background()

	// Create client with streaming enabled
	// IncludePartialMessages: true tells the Claude CLI to send
	// SDKStreamEvent messages with incremental updates
	// (MessageStartEvent, ContentBlockDeltaEvent, etc.)
	// instead of only sending the complete AssistantMessage at the end.
	opts := &claude.Options{
		Model:                  "claude-sonnet-4-5",
		MaxTurns:               maxTurns,
		IncludePartialMessages: true, // Enable real-time streaming
	}

	client, err := claude.NewClient(opts)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			log.Printf("Failed to close client: %v", closeErr)
		}
	}()

	// Send a query that will generate a longer response
	query := "Explain what recursion is in programming with " +
		"a simple example. Keep it concise but thorough."
	fmt.Printf("Query: %s\n\n", query)
	fmt.Println("Response (streaming):")
	fmt.Println(strings.Repeat("=", separatorLength))

	err = client.Query(ctx, query)
	if err != nil {
		log.Printf("Failed to send query: %v", err)

		return
	}

	// Process streaming responses
	processStreamingResponse(ctx, client)
}

// processStreamingResponse handles the streaming response loop.
func processStreamingResponse(
	ctx context.Context,
	client *claude.ClaudeSDKClient,
) {
	msgChan, errChan := client.ReceiveMessages(ctx)

	state := &streamState{
		spinnerChars: []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'},
	}

	ticker := time.NewTicker(spinnerTickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			state.updateSpinner()

		case msg := <-msgChan:
			if msg == nil {
				state.finalize()

				return
			}
			handleMessage(msg, state)

		case err := <-errChan:
			if err != nil {
				log.Printf("\n\nError: %v", err)

				return
			}
		}
	}
}

// streamState holds the state for streaming display.
type streamState struct {
	currentText  strings.Builder
	spinnerChars []rune
	spinnerIndex int
	isStreaming  bool
	blockIndex   int
}

// updateSpinner displays the thinking spinner animation.
func (s *streamState) updateSpinner() {
	if !s.isStreaming {
		return
	}

	fmt.Printf("\r%c Thinking...", s.spinnerChars[s.spinnerIndex])
	s.spinnerIndex = (s.spinnerIndex + 1) % len(s.spinnerChars)
}

// finalize clears the spinner and prints completion message.
func (s *streamState) finalize() {
	if s.isStreaming {
		fmt.Print("\r" + strings.Repeat(" ", clearLineLength) + "\r")
	}
	fmt.Println("\n" + strings.Repeat("=", separatorLength))
	fmt.Println("\n✓ Streaming complete")
}

// handleMessage processes different types of messages.
func handleMessage(msg claude.SDKMessage, state *streamState) {
	switch m := msg.(type) {
	case *claude.SDKStreamEvent:
		handleStreamEvent(
			m.Event,
			&state.currentText,
			&state.blockIndex,
			&state.isStreaming,
		)

	case *claude.SDKAssistantMessage:
		handleAssistantMessage(m, state)

	case *claude.SDKResultMessage:
		// Result received - analysis complete
	}
}

// handleAssistantMessage processes full assistant messages.
func handleAssistantMessage(
	msg *claude.SDKAssistantMessage,
	state *streamState,
) {
	if state.currentText.Len() != 0 {
		return
	}

	for _, block := range msg.Message.Content {
		switch b := block.(type) {
		case claude.TextBlock:
			fmt.Print(b.Text)
		case claude.TextContentBlock:
			fmt.Print(b.Text)
		}
	}
}

// handleStreamEvent processes individual stream events.
func handleStreamEvent(
	event claude.RawMessageStreamEvent,
	currentText *strings.Builder,
	blockIndex *int,
	isStreaming *bool,
) {
	switch evt := event.(type) {
	case claude.MessageStartEvent:
		handleMessageStart(isStreaming)

	case claude.ContentBlockStartEvent:
		handleContentBlockStart(evt, currentText, blockIndex)

	case claude.ContentBlockDeltaEvent:
		handleContentBlockDelta(evt, currentText)

	case claude.ContentBlockStopEvent:
		*isStreaming = false

	case claude.MessageStopEvent:
		*isStreaming = false
	}
}

// handleMessageStart processes message start events.
func handleMessageStart(isStreaming *bool) {
	*isStreaming = true
	fmt.Print("\r" + strings.Repeat(" ", clearLineLength) + "\r")
}

// handleContentBlockStart processes content block start events.
func handleContentBlockStart(
	evt claude.ContentBlockStartEvent,
	currentText *strings.Builder,
	blockIndex *int,
) {
	if *blockIndex > 0 {
		fmt.Println()
	}
	*blockIndex++

	switch block := evt.ContentBlock.(type) {
	case claude.TextContentBlock:
		currentText.Reset()
	case claude.ThinkingBlock:
		fmt.Print("🤔 [Thinking] ")
	case claude.ToolUseContentBlock:
		fmt.Printf("🔧 [Tool: %s] ", block.Name)
	}
}

// handleContentBlockDelta processes content block delta events.
func handleContentBlockDelta(
	evt claude.ContentBlockDeltaEvent,
	currentText *strings.Builder,
) {
	if evt.Delta.TextDelta == nil {
		return
	}

	text := *evt.Delta.TextDelta
	currentText.WriteString(text)
	fmt.Print(text)
}
